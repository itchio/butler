package pwr

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	osync "sync"

	"github.com/dustin/go-humanize"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/sync"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
)

var (
	// ErrMalformedPatch is returned when a patch could not be parsed
	ErrMalformedPatch = errors.New("malformed patch")

	// ErrIncompatiblePatch is returned when a patch but parsing
	// and applying it is unsupported (e.g. it's a newer version of the format)
	ErrIncompatiblePatch = errors.New("unsupported patch")
)

type DataFetchFunction func([]byte, string) ([]byte, error)
type DataStoreFunction func([]byte) error

// ApplyContext holds the state while applying a patch
type ApplyContext struct {
	Consumer *StateConsumer

	TargetPath string
	OutputPath string
	InPlace    bool

	TargetContainer *tlc.Container
	TargetPool      sync.FilePool
	SourceContainer *tlc.Container

	SignatureFilePath string

	TouchedFiles int
	NoopFiles    int
	DeletedFiles int
	StageSize    int64

	DataFetch DataFetchFunction
	DataStore DataStoreFunction
}

type signature []sync.BlockHash
type signatureSet map[string]signature
type signatureResult struct {
	path string
	sig  signature
	err  error
}

// ApplyPatch reads a patch, parses it, and generates the new file tree
func (actx *ApplyContext) ApplyPatch(patchReader io.Reader) error {
	actualOutputPath := actx.OutputPath
	if actx.InPlace {
		// applying in-place is a bit tricky: we can't overwrite files in the
		// target directory (old) while we're reading the patch otherwise
		// we might be copying new bytes instead of old bytes into later files
		// so, we rebuild 'touched' files in a staging area
		stagePath := actualOutputPath + "-stage"
		err := os.MkdirAll(stagePath, os.FileMode(0755))
		if err != nil {
			return err
		}

		defer os.RemoveAll(stagePath)
		actx.OutputPath = stagePath
	}

	rawPatchWire := wire.NewReadContext(patchReader)
	err := rawPatchWire.ExpectMagic(PatchMagic)
	if err != nil {
		return err
	}

	header := &PatchHeader{}
	err = rawPatchWire.ReadMessage(header)
	if err != nil {
		return fmt.Errorf("while reading message: %s", err)
	}

	patchWire, err := DecompressWire(rawPatchWire, header.Compression)
	if err != nil {
		return err
	}

	targetContainer := &tlc.Container{}
	err = patchWire.ReadMessage(targetContainer)
	if err != nil {
		return err
	}
	actx.TargetContainer = targetContainer

	sourceContainer := &tlc.Container{}
	err = patchWire.ReadMessage(sourceContainer)
	if err != nil {
		return err
	}
	actx.SourceContainer = sourceContainer

	var deletedFiles []string

	if actx.InPlace {
		// when working in-place, we have to keep track of which files were deleted
		// from one version to the other, so that we too may delete them in the end.
		deletedFiles = detectRemovedFiles(actx.SourceContainer, actx.TargetContainer)
	} else {
		// when rebuilding in a fresh directory, there's no need to worry about
		// deleted files, because they won't even exist in the first place.
		err = sourceContainer.Prepare(actx.OutputPath)
		if err != nil {
			return err
		}
	}

	hashPaths := make(chan string, 16)
	done := make(chan bool)
	errs := make(chan error)
	ss := make(signatureSet)

	go actx.hashThings(ss, hashPaths, done, errs)
	go actx.patchThings(patchWire, hashPaths, done, errs)

	for i := 0; i < 2; i++ {
		select {
		case <-done:
			// woo
		case err := <-errs:
			return err
		}
	}

	if actx.SignatureFilePath != "" {
		err := actx.checkHashes(ss)
		if err != nil {
			return err
		}
	}

	if actx.InPlace {
		actx.DeletedFiles = len(deletedFiles)

		actx.StageSize, err = mergeFolders(actualOutputPath, actx.OutputPath)
		if err != nil {
			return fmt.Errorf("in mergeFolders: %s", err.Error())
		}

		err = deleteFiles(actualOutputPath, deletedFiles)
		if err != nil {
			return fmt.Errorf("in deleteFiles: %s", err.Error())
		}
		actx.OutputPath = actualOutputPath
	}

	return nil
}

// behold, a good example of resisting the urge to optimize
// prematurely: while *computing* hashes from files is done using 4
// workers, comparing the hashes is done completely linearly,
// by reading the entire signature first, and advancing 1 block
// at most, whereas we could 1) stream read 2) parallel check 3)
// skip untouched files. but it's so far from mattering, that code
// will probably remain untouched for years.
func (actx *ApplyContext) checkHashes(ss signatureSet) error {
	reader, err := os.Open(actx.SignatureFilePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	container, allSigs, err := ReadSignature(reader)
	if err != nil {
		return err
	}

	skip := int64(-1)
	check := int64(-1)
	var checkSig signature
	checkOffset := 0

	for _, bh := range allSigs {
		if bh.FileIndex == skip {
			continue
		}

		if bh.FileIndex != check {
			if check > 0 && checkOffset != len(checkSig) {
				return fmt.Errorf("In %s, expected %d blocks, got %d", container.Files[check].Path, checkOffset, len(checkSig))
			}

			checkOffset = 0
			checkSig = ss[container.Files[bh.FileIndex].Path]
			if checkSig != nil {
				check = bh.FileIndex
			} else {
				skip = bh.FileIndex
			}
		}

		if bh.FileIndex == check {
			ah := checkSig[checkOffset]
			if ah.WeakHash != bh.WeakHash {
				return fmt.Errorf("%s: weak hash mismatch @ block %d (%s into the file)",
					container.Files[bh.FileIndex].Path,
					checkOffset,
					humanize.Bytes(uint64(BlockSize*checkOffset)))
			}
			if !bytes.Equal(ah.StrongHash, bh.StrongHash) {
				return fmt.Errorf("%s: strong hash mismatch @ block %d (%s into the file)",
					container.Files[bh.FileIndex].Path,
					checkOffset,
					humanize.Bytes(uint64(BlockSize*checkOffset)))
			}
			checkOffset++
		}
	}

	return nil
}

func (actx *ApplyContext) hashThings(ss signatureSet, hashPaths chan string, doneOut chan bool, errOut chan error) {
	c := make(chan signatureResult)
	done := make(chan struct{})
	var wg osync.WaitGroup

	const numWorkers = 4
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		sctx := sync.NewContext(BlockSize)
		go func() {
			for hashPath := range hashPaths {
				sig, err := func() (signature, error) {
					var sig signature
					onWrite := func(h sync.BlockHash) error {
						sig = append(sig, h)
						return nil
					}

					fullPath := filepath.Join(actx.OutputPath, hashPath)
					reader, err := os.Open(fullPath)
					if err != nil {
						return nil, err
					}
					defer reader.Close()

					err = sctx.CreateSignature(0, reader, onWrite)
					if err != nil {
						return nil, err
					}

					return sig, nil
				}()

				select {
				case <-done:
					return
				case c <- signatureResult{path: hashPath, sig: sig, err: err}:
					// muffin
				}
			}
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(c)
	}()

	for r := range c {
		if r.err != nil {
			errOut <- r.err
			close(done)
		}
		ss[r.path] = r.sig
	}

	doneOut <- true
}

func (actx *ApplyContext) patchThings(patchWire *wire.ReadContext, hashPaths chan string, done chan bool, errs chan error) {
	err := func() error {
		sourceContainer := actx.SourceContainer
		outputPool := sourceContainer.NewFilePool(actx.OutputPath)

		targetContainer := actx.TargetContainer
		targetPool := actx.TargetPool
		if targetPool == nil {
			if actx.TargetPath == "/dev/null" {
				targetPool = targetContainer.NewFilePool(actx.TargetPath)
			} else {
				targetInfo, err := os.Lstat(actx.TargetPath)
				if err != nil {
					return err
				}

				if targetInfo.IsDir() {
					targetPool = targetContainer.NewFilePool(actx.TargetPath)
				} else {
					fr, err := os.Open(actx.TargetPath)
					if err != nil {
						return err
					}

					zr, err := zip.NewReader(fr, targetInfo.Size())
					if err != nil {
						return err
					}

					targetPool = targetContainer.NewZipPool(zr)
				}
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
				return err
			}

			if sh.FileIndex != int64(fileIndex) {
				fmt.Printf("expected fileIndex = %d, got fileIndex %d\n", fileIndex, sh.FileIndex)
				return ErrMalformedPatch
			}

			ops := make(chan sync.Operation)
			errc := make(chan error, 1)

			go readOps(patchWire, ops, errc, actx.DataFetch, actx.DataStore)

			bytesWritten, noop, err := lazilyPatchFile(sctx, targetContainer, targetPool, sourceContainer, outputPool, sh.FileIndex, onSourceWrite, ops, actx.InPlace)
			if err != nil {
				return err
			}

			if noop {
				actx.NoopFiles++
			} else {
				actx.TouchedFiles++
				if bytesWritten != f.Size {
					return fmt.Errorf("%s: expected to write %d bytes, wrote %d bytes", f.Path, f.Size, bytesWritten)
				}
				hashPaths <- f.Path
			}

			// using errc to signal the end of processing, rather than having a separate
			// done channel. not sure if there's any upside to either
			err = <-errc
			if err != nil {
				return fmt.Errorf("while reading patch: %s", err.Error())
			}

		}

		err := targetPool.Close()
		if err != nil {
			return err
		}

		return nil
	}()

	if err != nil {
		errs <- err
		return
	}

	close(hashPaths)
	done <- true
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

	stageContainer, err := tlc.Walk(stagePath, filter)
	if err != nil {
		return 0, err
	}

	move := func(path string) error {
		p := filepath.FromSlash(path)
		op := filepath.Join(outPath, p)
		sp := filepath.Join(stagePath, p)

		err := os.Remove(op)
		if err != nil {
			if !os.IsNotExist(err) {
				return err
			}
		}

		err = os.MkdirAll(filepath.Dir(op), os.FileMode(0755))
		if err != nil {
			return err
		}

		err = os.Rename(sp, op)
		if err != nil {
			return err
		}
		return nil
	}

	for _, f := range stageContainer.Files {
		err := move(f.Path)
		if err != nil {
			return 0, err
		}
	}

	for _, s := range stageContainer.Symlinks {
		err := move(s.Path)
		if err != nil {
			return 0, err
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
				return err
			}
		}
	}

	return nil
}

func lazilyPatchFile(sctx *sync.Context, targetContainer *tlc.Container, targetPool sync.FilePool, outputContainer *tlc.Container, outputPool *tlc.ContainerFilePool,
	fileIndex int64, onSourceWrite counter.CountCallback, ops chan sync.Operation, inplace bool) (written int64, noop bool, err error) {

	var realops chan sync.Operation

	errs := make(chan error)
	first := true

	for op := range ops {
		if first {
			first = false

			// if the first operation is a blockrange that copies an
			// entire file from target into a file from source that has
			// the same name and size, then it's a no-op!
			if op.Type == sync.OpBlockRange && op.BlockIndex == 0 {
				outputFile := outputContainer.Files[fileIndex]
				targetFile := targetContainer.Files[op.FileIndex]
				numOutputBlocks := numBlocks(outputFile.Size)

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
				realops = make(chan sync.Operation)

				outputFile := outputContainer.Files[fileIndex]

				outputPath := outputPool.GetPath(fileIndex)
				err = os.MkdirAll(filepath.Dir(outputPath), os.FileMode(0755))
				if err != nil {
					return 0, false, err
				}

				var writer io.WriteCloser
				writer, err = os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(outputFile.Mode)|ModeMask)
				if err != nil {
					return 0, false, err
				}

				writeCounter := counter.NewWriterCallback(onSourceWrite, writer)

				go func() {
					err := sctx.ApplyPatch(writeCounter, targetPool, realops)
					if err != nil {
						errs <- err
						return
					}

					err = writer.Close()
					if err != nil {
						errs <- err
						return
					}

					written = writeCounter.Count()
					errs <- nil
				}()
			}
		}

		if !noop {
			select {
			case err := <-errs:
				if err != nil {
					return 0, false, err
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
		return 0, false, err
	}

	return
}

func readOps(rc *wire.ReadContext, ops chan sync.Operation, errc chan error, dataFetch DataFetchFunction, dataStore DataStoreFunction) {
	defer close(ops)
	var buf []byte

	rop := &SyncOp{}

	readingOps := true
	for readingOps {
		rop.Reset()
		err := rc.ReadMessage(rop)
		if err != nil {
			errc <- fmt.Errorf("while reading op message: %s", err)
			return
		}

		switch rop.Type {
		case SyncOp_BLOCK_RANGE:
			ops <- sync.Operation{
				Type:       sync.OpBlockRange,
				FileIndex:  rop.FileIndex,
				BlockIndex: rop.BlockIndex,
				BlockSpan:  rop.BlockSpan,
			}

		case SyncOp_DATA:
			ops <- sync.Operation{
				Type: sync.OpData,
				Data: rop.Data,
			}

			if dataStore != nil {
				err := dataStore(rop.Data)
				if err != nil {
					errc <- err
					return
				}
			}

		case SyncOp_REMOTE_DATA:
			if dataFetch == nil {
				errc <- fmt.Errorf("encountered remote data but no datafetch")
				return
			}
			buf, err = dataFetch(buf[:0], rop.RemotePath)
			if err != nil {
				errc <- err
				return
			}

			if int64(len(buf)) != rop.RemoteSize {
				errc <- fmt.Errorf("remote data size mismatch: expected %d got %d", rop.RemoteSize, len(buf))
				return
			}

			ops <- sync.Operation{
				Type: sync.OpData,
				Data: buf,
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
				errc <- ErrMalformedPatch
				return
			}
		}
	}

	errc <- nil
}
