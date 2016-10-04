package pwr

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/pools"
	"github.com/itchio/wharf/pools/fspool"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
	"github.com/itchio/wharf/wsync"
)

var (
	// ErrMalformedPatch is returned when a patch could not be parsed
	ErrMalformedPatch = errors.New("malformed patch")

	// ErrIncompatiblePatch is returned when a patch but parsing
	// and applying it is unsupported (e.g. it's a newer version of the format)
	ErrIncompatiblePatch = errors.New("unsupported patch")
)

// VetApplyFunc gives a chance to the caller to abort the application
// before any ops are read/applied - it's the right place to check for
// limits on container size, or number of files, for example.
// By the time it's called, TargetContainer and SourceContainer are
// valid. A VetApplyFunc should only read data from actx, not write to it.
type VetApplyFunc func(actx *ApplyContext) error

// ApplyContext holds the state while applying a patch
type ApplyContext struct {
	Consumer *StateConsumer

	TargetPath string
	OutputPath string
	InPlace    bool

	TargetContainer *tlc.Container
	TargetPool      wsync.Pool
	SourceContainer *tlc.Container
	OutputPool      wsync.WritablePool

	WoundsPath     string
	WoundsConsumer WoundsConsumer

	VetApply VetApplyFunc

	Signature *SignatureInfo

	TouchedFiles int
	NoopFiles    int
	DeletedFiles int
	StageSize    int64
}

type signature []wsync.BlockHash
type signatureSet map[string]signature
type signatureResult struct {
	path string
	sig  signature
	err  error
}

// ApplyPatch reads a patch, parses it, and generates the new file tree
func (actx *ApplyContext) ApplyPatch(patchReader io.Reader) error {
	actualOutputPath := actx.OutputPath
	if actx.OutputPool == nil {
		if actx.InPlace {
			// applying in-place is a bit tricky: we can't overwrite files in the
			// target directory (old) while we're reading the patch otherwise
			// we might be copying new bytes instead of old bytes into later files
			// so, we rebuild 'touched' files in a staging area
			stagePath := actualOutputPath + "-stage"
			err := os.MkdirAll(stagePath, os.FileMode(0755))
			if err != nil {
				return errors.Wrap(err, 1)
			}

			defer os.RemoveAll(stagePath)
			actx.OutputPath = stagePath
		} else {
			os.MkdirAll(actx.OutputPath, os.FileMode(0755))
		}
	} else {
		if actualOutputPath != "" {
			return fmt.Errorf("cannot specify both OutputPath and OutputPool")
		}
	}

	rawPatchWire := wire.NewReadContext(patchReader)
	err := rawPatchWire.ExpectMagic(PatchMagic)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	header := &PatchHeader{}
	err = rawPatchWire.ReadMessage(header)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	patchWire, err := DecompressWire(rawPatchWire, header.Compression)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	targetContainer := &tlc.Container{}
	err = patchWire.ReadMessage(targetContainer)
	if err != nil {
		return errors.Wrap(err, 1)
	}
	actx.TargetContainer = targetContainer

	sourceContainer := &tlc.Container{}
	err = patchWire.ReadMessage(sourceContainer)
	if err != nil {
		return errors.Wrap(err, 1)
	}
	actx.SourceContainer = sourceContainer

	if actx.VetApply != nil {
		err = actx.VetApply(actx)
		if err != nil {
			return errors.Wrap(err, 1)
		}
	}

	var deletedFiles []string

	// when not working with a custom output pool
	if actx.OutputPool == nil {
		if actx.InPlace {
			// when working in-place, we have to keep track of which files were deleted
			// from one version to the other, so that we too may delete them in the end.
			deletedFiles = detectRemovedFiles(actx.SourceContainer, actx.TargetContainer)
		} else {
			// when rebuilding in a fresh directory, there's no need to worry about
			// deleted files, because they won't even exist in the first place.
			err = sourceContainer.Prepare(actx.OutputPath)
			if err != nil {
				return errors.Wrap(err, 1)
			}
		}
	}

	err = actx.patchAll(patchWire, actx.Signature)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	if actx.InPlace {
		actx.DeletedFiles = len(deletedFiles)

		actx.StageSize, err = mergeFolders(actualOutputPath, actx.OutputPath)
		if err != nil {
			return errors.Wrap(err, 1)
		}

		err = deleteFiles(actualOutputPath, deletedFiles)
		if err != nil {
			return errors.Wrap(err, 1)
		}
		actx.OutputPath = actualOutputPath
	}

	return nil
}

func (actx *ApplyContext) patchAll(patchWire *wire.ReadContext, signature *SignatureInfo) error {
	sourceContainer := actx.SourceContainer

	var validatingPool *ValidatingPool
	errs := make(chan error)
	done := make(chan bool)
	numTasks := 0

	outputPool := actx.OutputPool
	if outputPool == nil {
		outputPool = fspool.New(sourceContainer, actx.OutputPath)
	}

	if signature != nil {
		validatingPool = &ValidatingPool{
			Pool:      outputPool,
			Container: sourceContainer,
			Signature: signature,
		}

		if actx.WoundsPath != "" {
			validatingPool.Wounds = make(chan *Wound)

			actx.WoundsConsumer = &WoundsWriter{
				WoundsPath: actx.WoundsPath,
			}
			numTasks++
		}

		if actx.WoundsConsumer != nil {
			go func() {
				err := actx.WoundsConsumer.Do(signature.Container, validatingPool.Wounds)
				if err != nil {
					errs <- err
					return
				}
				done <- true
			}()
		}

		outputPool = validatingPool
	}

	targetContainer := actx.TargetContainer
	targetPool := actx.TargetPool
	if targetPool == nil {
		if actx.TargetPath == "" {
			return fmt.Errorf("apply: need either TargetPool or TargetPath")
		}
		var cErr error
		targetPool, cErr = pools.New(targetContainer, actx.TargetPath)
		if cErr != nil {
			return cErr
		}
	}

	fileOffset := int64(0)
	sourceBytes := sourceContainer.Size
	onSourceWrite := func(count int64) {
		// we measure patching progress as the number of total bytes written
		// to the source container. no-ops (untouched files) count too, so the
		// progress bar may jump ahead a bit at times, but that's a good surprise
		// measuring progress by bytes of the patch read would just be a different
		// kind of inaccuracy (due to decompression buffers, etc.)
		actx.Consumer.Progress(float64(fileOffset+count) / float64(sourceBytes))
	}

	sctx := mksync()
	sh := &SyncHeader{}

	for fileIndex, f := range sourceContainer.Files {
		actx.Consumer.ProgressLabel(f.Path)
		actx.Consumer.Debug(f.Path)
		fileOffset = f.Offset

		// each series of patch operations is preceded by a SyncHeader giving
		// us the file index - it's a super basic measure to make sure the
		// patch file we're reading and the patching algorithm somewhat agree
		// on what's happening.
		sh.Reset()
		err := patchWire.ReadMessage(sh)
		if err != nil {
			return errors.Wrap(err, 1)
		}

		if sh.FileIndex != int64(fileIndex) {
			fmt.Printf("expected fileIndex = %d, got fileIndex %d\n", fileIndex, sh.FileIndex)
			return errors.Wrap(ErrMalformedPatch, 1)
		}

		ops := make(chan wsync.Operation)
		errc := make(chan error, 1)

		go readOps(patchWire, ops, errc)

		bytesWritten, noop, err := lazilyPatchFile(sctx, targetContainer, targetPool, sourceContainer, outputPool, sh.FileIndex, onSourceWrite, ops, actx.InPlace)
		if err != nil {
			return errors.Wrap(err, 1)
		}

		if noop {
			actx.NoopFiles++
		} else {
			actx.TouchedFiles++
			if bytesWritten != f.Size {
				return fmt.Errorf("%s: expected to write %d bytes, wrote %d bytes", f.Path, f.Size, bytesWritten)
			}
		}

		// using errc to signal the end of processing, rather than having a separate
		// done channel. not sure if there's any upside to either
		err = <-errc
		if err != nil {
			return errors.Wrap(err, 1)
		}

	}

	err := targetPool.Close()
	if err != nil {
		return errors.Wrap(err, 1)
	}

	err = outputPool.Close()
	if err != nil {
		return errors.Wrap(err, 1)
	}

	if validatingPool != nil {
		if validatingPool.Wounds != nil {
			close(validatingPool.Wounds)
		}
	}

	for i := 0; i < numTasks; i++ {
		select {
		case err = <-errs:
			return errors.Wrap(err, 1)
		case <-done:
			// good!
		}
	}

	return nil
}

func detectRemovedFiles(sourceContainer *tlc.Container, targetContainer *tlc.Container) []string {
	// first make a map of all the file paths in source, for later lookup
	sourceFileMap := make(map[string]bool)
	for _, f := range sourceContainer.Files {
		sourceFileMap[f.Path] = true
	}
	for _, s := range sourceContainer.Symlinks {
		sourceFileMap[s.Path] = true
	}
	for _, d := range sourceContainer.Dirs {
		sourceFileMap[d.Path] = true
	}

	// then walk through target container paths, if they're not in source, they were deleted
	var deletedFiles []string
	for _, f := range targetContainer.Files {
		if !sourceFileMap[f.Path] {
			deletedFiles = append(deletedFiles, f.Path)
		}
	}
	for _, s := range targetContainer.Symlinks {
		if !sourceFileMap[s.Path] {
			deletedFiles = append(deletedFiles, s.Path)
		}
	}
	for _, d := range targetContainer.Dirs {
		if !sourceFileMap[d.Path] {
			deletedFiles = append(deletedFiles, d.Path)
		}
	}
	return deletedFiles
}

func mergeFolders(outPath string, stagePath string) (int64, error) {
	var filter tlc.FilterFunc = func(fi os.FileInfo) bool {
		return true
	}

	stageContainer, err := tlc.WalkDir(stagePath, filter)
	if err != nil {
		return 0, errors.Wrap(err, 1)
	}

	move := func(path string) error {
		p := filepath.FromSlash(path)
		op := filepath.Join(outPath, p)
		sp := filepath.Join(stagePath, p)

		err := os.Remove(op)
		if err != nil {
			if !os.IsNotExist(err) {
				return errors.Wrap(err, 1)
			}
		}

		err = os.MkdirAll(filepath.Dir(op), os.FileMode(0755))
		if err != nil {
			return errors.Wrap(err, 1)
		}

		err = os.Rename(sp, op)
		if err != nil {
			return errors.Wrap(err, 1)
		}
		return nil
	}

	for _, f := range stageContainer.Files {
		err := move(f.Path)
		if err != nil {
			return 0, errors.Wrap(err, 1)
		}
	}

	for _, s := range stageContainer.Symlinks {
		err := move(s.Path)
		if err != nil {
			return 0, errors.Wrap(err, 1)
		}
	}

	return stageContainer.Size, nil
}

type byDecreasingLength []string

func (s byDecreasingLength) Len() int {
	return len(s)
}

func (s byDecreasingLength) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byDecreasingLength) Less(i, j int) bool {
	return len(s[j]) < len(s[i])
}

func deleteFiles(outPath string, deletedFiles []string) error {
	sort.Sort(byDecreasingLength(deletedFiles))

	for _, f := range deletedFiles {
		p := filepath.FromSlash(f)
		op := filepath.Join(outPath, p)
		err := os.Remove(op)
		if err != nil {
			if !os.IsNotExist(err) {
				return errors.Wrap(err, 1)
			}
		}
	}

	return nil
}

func lazilyPatchFile(sctx *wsync.Context, targetContainer *tlc.Container, targetPool wsync.Pool, outputContainer *tlc.Container, outputPool wsync.WritablePool,
	fileIndex int64, onSourceWrite counter.CountCallback, ops chan wsync.Operation, inplace bool) (written int64, noop bool, err error) {

	var realops chan wsync.Operation

	errs := make(chan error)
	first := true

	for op := range ops {
		if first {
			first = false

			// if the first operation is a blockrange that copies an
			// entire file from target into a file from source that has
			// the same name and size, then it's a no-op!
			if op.Type == wsync.OpBlockRange && op.BlockIndex == 0 {
				outputFile := outputContainer.Files[fileIndex]
				targetFile := targetContainer.Files[op.FileIndex]
				numOutputBlocks := ComputeNumBlocks(outputFile.Size)

				if inplace &&
					op.BlockSpan == numOutputBlocks &&
					outputFile.Size == targetFile.Size &&
					outputFile.Path == targetFile.Path {
					noop = true
				}
			}

			if noop {
				go func() {
					errs <- nil
				}()
			} else {
				realops = make(chan wsync.Operation)

				var writer io.WriteCloser
				writer, err = outputPool.GetWriter(fileIndex)
				if err != nil {
					return 0, false, errors.Wrap(err, 1)
				}

				writeCounter := counter.NewWriterCallback(onSourceWrite, writer)

				go func() {
					rErr := sctx.ApplyPatch(writeCounter, targetPool, realops)
					if rErr != nil {
						errs <- errors.Wrap(rErr, 1)
						return
					}

					rErr = writer.Close()
					if rErr != nil {
						errs <- errors.Wrap(rErr, 1)
						return
					}

					written = writeCounter.Count()
					errs <- nil
				}()
			}
		}

		if !noop {
			select {
			case cErr := <-errs:
				if cErr != nil {
					return 0, false, errors.Wrap(cErr, 1)
				}
			case realops <- op:
				// muffin
			}
		}
	}

	if !noop {
		close(realops)
	}

	err = <-errs
	if err != nil {
		return 0, false, errors.Wrap(err, 1)
	}

	return
}

func readOps(rc *wire.ReadContext, ops chan wsync.Operation, errc chan error) {
	defer close(ops)
	rop := &SyncOp{}

	readingOps := true
	for readingOps {
		rop.Reset()
		err := rc.ReadMessage(rop)
		if err != nil {
			errc <- errors.Wrap(err, 1)
			return
		}

		switch rop.Type {
		case SyncOp_BLOCK_RANGE:
			ops <- wsync.Operation{
				Type:       wsync.OpBlockRange,
				FileIndex:  rop.FileIndex,
				BlockIndex: rop.BlockIndex,
				BlockSpan:  rop.BlockSpan,
			}

		case SyncOp_DATA:
			ops <- wsync.Operation{
				Type: wsync.OpData,
				Data: rop.Data,
			}

		default:
			switch rop.Type {
			case SyncOp_HEY_YOU_DID_IT:
				// series of patching operations always end with a SyncOp_HEY_YOU_DID_IT.
				// this helps detect truncated patch files, and, again, basic boundary
				// safety measures are cheap and reassuring.
				readingOps = false
			default:
				// if you get this, then you're probably implementing an extension
				// to the wharf patch format in which case, I'd love to get in touch
				// with you to know why & discuss adding it to the spec so other
				// people can share it: amos@itch.io
				fmt.Printf("unrecognized rop type %d\n", rop.Type)
				errc <- errors.Wrap(ErrMalformedPatch, 1)
				return
			}
		}
	}

	errc <- nil
}
