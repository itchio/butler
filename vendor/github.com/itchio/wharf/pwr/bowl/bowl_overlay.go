package bowl

import (
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/itchio/wharf/pwr/overlay"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/pools/fspool"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wsync"
)

var debugBrokenRename = os.Getenv("BOWL_DEBUG_BROKEN_RENAME") == "1"
var overlayVerbose = os.Getenv("BOWL_OVERLAY_VERBOSE") == "1"

func debugf(format string, args ...interface{}) {
	if overlayVerbose {
		fmt.Printf("[overlayBowl] %s\n", fmt.Sprintf(format, args...))
	}
}

type overlayBowl struct {
	TargetContainer *tlc.Container
	TargetPool      wsync.Pool
	SourceContainer *tlc.Container

	OutputFolder string
	StageFolder  string

	stagePool         *fspool.FsPool
	transpositions    []Transposition
	targetFilesByPath map[string]int64

	// files we'll have to patch using an overlay
	overlayFiles []int64
	// files we'll have to move from the staging folder to the dest
	moveFiles []int64
}

var _ Bowl = (*overlayBowl)(nil)

type OverlayBowlParams struct {
	TargetContainer *tlc.Container
	SourceContainer *tlc.Container

	OutputFolder string
	StageFolder  string
}

func NewOverlayBowl(params *OverlayBowlParams) (Bowl, error) {
	// input validation

	if params.TargetContainer == nil {
		return nil, errors.New("overlaybowl: TargetContainer must not be nil")
	}

	if params.SourceContainer == nil {
		return nil, errors.New("overlaybowl: SourceContainer must not be nil")
	}

	if params.OutputFolder == "" {
		return nil, errors.New("overlaybowl: OutputFolder must not be nil")
	}

	{
		stats, err := os.Stat(params.OutputFolder)
		if err != nil {
			return nil, errors.Wrap(fmt.Errorf("overlaybowl: OutputFolder must exist, but got: %s", err.Error()), 0)
		}

		if !stats.IsDir() {
			return nil, errors.New("overlaybowl: OutputFolder must exist and be a directory")
		}
	}

	if params.StageFolder == "" {
		return nil, errors.New("overlaybowl: StageFolder must not be nil")
	}

	var err error

	err = os.MkdirAll(params.OutputFolder, 0755)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	err = os.MkdirAll(params.StageFolder, 0755)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	stagePool := fspool.New(params.SourceContainer, params.StageFolder)
	targetPool := fspool.New(params.TargetContainer, params.OutputFolder)

	targetFilesByPath := make(map[string]int64)
	for index, tf := range params.TargetContainer.Files {
		targetFilesByPath[tf.Path] = int64(index)
	}

	return &overlayBowl{
		TargetContainer: params.TargetContainer,
		TargetPool:      targetPool,
		SourceContainer: params.SourceContainer,

		OutputFolder: params.OutputFolder,
		StageFolder:  params.StageFolder,

		stagePool:         stagePool,
		targetFilesByPath: targetFilesByPath,
	}, nil
}

func (ob *overlayBowl) GetWriter(index int64) (EntryWriter, error) {
	sourceFile := ob.SourceContainer.Files[index]
	if sourceFile == nil {
		return nil, errors.Wrap(fmt.Errorf("overlayBowl: unknown source file %d", index), 0)
	}

	if targetIndex, ok := ob.targetFilesByPath[sourceFile.Path]; ok {
		debugf("returning overlay writer for '%s'", sourceFile.Path)

		// oh damn, that file already exists in the output - let's make an overlay
		ob.markOverlay(index)

		r, err := ob.TargetPool.GetReadSeeker(targetIndex)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		wPath := ob.stagePool.GetPath(index)
		return &overlayEntryWriter{path: wPath, readSeeker: r}, nil
	}

	// guess it's a new file! let's write it to staging anyway
	ob.markMove(index)

	debugf("returning move writer for '%s'", sourceFile.Path)

	wPath := ob.stagePool.GetPath(index)
	return &freshEntryWriter{path: wPath}, nil
}

func (ob *overlayBowl) markOverlay(index int64) {
	// make sure we don't double mark it
	for _, i := range ob.overlayFiles {
		if i == index {
			// oh cool it's already marked
			return
		}
	}

	// mark it
	ob.overlayFiles = append(ob.overlayFiles, index)
}

func (ob *overlayBowl) markMove(index int64) {
	// make sure we don't double mark it
	for _, i := range ob.moveFiles {
		if i == index {
			// oh cool it's already marked
			return
		}
	}

	// mark it
	ob.moveFiles = append(ob.moveFiles, index)
}

func (ob *overlayBowl) Transpose(t Transposition) error {
	// ok, say we resumed, maybe we already have a transposition for this source file?
	for i, tt := range ob.transpositions {
		if tt.SourceIndex == t.SourceIndex {
			// and so we do! let's replace it.
			ob.transpositions[i] = t
			return nil
		}
	}

	// if we didn't already have one, let's record it for when we commit
	ob.transpositions = append(ob.transpositions, t)
	return nil
}

func (ob *overlayBowl) Commit() error {
	// oy, do we have work to do!
	var err error

	// - close the target pool, in case it still has a reader open!
	err = ob.TargetPool.Close()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	// - ensure dirs and symlinks
	err = ob.ensureDirsAndSymlinks()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	// - apply transpositions
	err = ob.applyTranspositions()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	// - move files we need to move
	err = ob.applyMoves()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	// - merge overlays
	err = ob.applyOverlays()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	// - delete ghosts
	err = ob.deleteGhosts()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}

func (ob *overlayBowl) ensureDirsAndSymlinks() error {
	outputPath := ob.OutputFolder

	for _, dir := range ob.SourceContainer.Dirs {
		path := filepath.Join(outputPath, filepath.FromSlash(dir.Path))

		err := os.MkdirAll(path, 0755)
		if err != nil {
			// If path is already a directory, MkdirAll does nothing and returns nil.
			// so if we get a non-nil error, we know it's serious business (permissions, etc.)
			return err
		}
	}

	// TODO: behave like github.com/itchio/savior for symlinks on windows ?

	for _, symlink := range ob.SourceContainer.Symlinks {
		path := filepath.Join(outputPath, filepath.FromSlash(symlink.Path))
		dest, err := os.Readlink(path)
		if err != nil {
			if os.IsNotExist(err) {
				// symlink was missing
				err = os.Symlink(filepath.FromSlash(symlink.Dest), path)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}

		// symlink is there
		if dest != filepath.FromSlash(symlink.Dest) {
			// wrong dest, fixing that
			err = os.Remove(path)
			if err != nil {
				return err
			}

			err = os.Symlink(filepath.FromSlash(symlink.Dest), path)
			if err != nil {
				return err
			}

			return nil
		}
	}

	return nil
}

type pathTranspo struct {
	TargetPath string
	OutputPath string
}

type mkdirBehavior int

const (
	mkdirBehaviorNever mkdirBehavior = 0xf8792 + iota
	mkdirBehaviorIfNeeded
)

func (ob *overlayBowl) applyTranspositions() error {
	transpositions := make(map[string][]*pathTranspo)
	outputPath := ob.OutputFolder

	for _, t := range ob.transpositions {
		targetFile := ob.TargetContainer.Files[t.TargetIndex]
		sourceFile := ob.SourceContainer.Files[t.SourceIndex]

		transpositions[targetFile.Path] = append(transpositions[targetFile.Path], &pathTranspo{
			TargetPath: targetFile.Path,
			OutputPath: sourceFile.Path,
		})
	}

	applyMultipleTranspositions := func(targetPath string, group []*pathTranspo) error {
		// a file got duplicated!
		var noop *pathTranspo
		for _, transpo := range group {
			if targetPath == transpo.OutputPath {
				noop = transpo
				break
			}
		}

		for i, transpo := range group {
			if noop == nil {
				if i == 0 {
					// arbitrary pick first transposition as being the rename - do
					// all the others as copies first
					continue
				}
			} else if transpo == noop {
				// no need to copy for the noop
				continue
			}

			oldAbsolutePath := filepath.Join(outputPath, filepath.FromSlash(targetPath))
			newAbsolutePath := filepath.Join(outputPath, filepath.FromSlash(transpo.OutputPath))
			err := ob.copy(oldAbsolutePath, newAbsolutePath, mkdirBehaviorIfNeeded)
			if err != nil {
				return errors.Wrap(err, 0)
			}
		}

		if noop == nil {
			// we treated the first transpo as being the rename, gotta do it now
			transpo := group[0]
			oldAbsolutePath := filepath.Join(outputPath, filepath.FromSlash(targetPath))
			newAbsolutePath := filepath.Join(outputPath, filepath.FromSlash(transpo.OutputPath))
			err := ob.move(oldAbsolutePath, newAbsolutePath)
			if err != nil {
				return errors.Wrap(err, 0)
			}
		} else {
			// muffin!
		}

		return nil
	}

	var cleanupRenames []*pathTranspo
	alreadyDone := make(map[string]bool)
	renameSeed := int64(0)

	for _, group := range transpositions {
		for _, transpo := range group {
			if transpo.TargetPath == transpo.OutputPath {
				// no-ops can't clash
				continue
			}

			if _, ok := transpositions[transpo.OutputPath]; ok {
				// transpo is writing to the source of swapBuddy, this will blow shit up
				// make it write to a safe path instead, then rename it to the correct path
				renameSeed++
				safePath := transpo.OutputPath + fmt.Sprintf(".butler-rename-%d", renameSeed)
				cleanupRenames = append(cleanupRenames, &pathTranspo{
					TargetPath: safePath,
					OutputPath: transpo.OutputPath,
				})
				transpo.OutputPath = safePath
			}
		}
	}

	for groupTargetPath, group := range transpositions {
		if alreadyDone[groupTargetPath] {
			continue
		}
		alreadyDone[groupTargetPath] = true

		if len(group) == 1 {
			transpo := group[0]
			if transpo.TargetPath == transpo.OutputPath {
				// file wasn't touched at all
			} else {
				// file was renamed
				oldAbsolutePath := filepath.Join(outputPath, filepath.FromSlash(transpo.TargetPath))
				newAbsolutePath := filepath.Join(outputPath, filepath.FromSlash(transpo.OutputPath))
				err := ob.move(oldAbsolutePath, newAbsolutePath)
				if err != nil {
					return errors.Wrap(err, 0)
				}
			}
		} else {
			err := applyMultipleTranspositions(groupTargetPath, group)
			if err != nil {
				return errors.Wrap(err, 0)
			}
		}
	}

	for _, rename := range cleanupRenames {
		oldAbsolutePath := filepath.Join(outputPath, filepath.FromSlash(rename.TargetPath))
		newAbsolutePath := filepath.Join(outputPath, filepath.FromSlash(rename.OutputPath))
		err := ob.move(oldAbsolutePath, newAbsolutePath)
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	return nil
}

func (ob *overlayBowl) copy(oldAbsolutePath string, newAbsolutePath string, mkdirBehavior mkdirBehavior) error {
	debugf("cp '%s' '%s'", oldAbsolutePath, newAbsolutePath)
	if mkdirBehavior == mkdirBehaviorIfNeeded {
		err := os.MkdirAll(filepath.Dir(newAbsolutePath), os.FileMode(0755))
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	// fall back to copy + remove
	reader, err := os.Open(oldAbsolutePath)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer reader.Close()

	stats, err := reader.Stat()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	writer, err := os.OpenFile(newAbsolutePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, stats.Mode()|tlc.ModeMask)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer writer.Close()

	_, err = io.Copy(writer, reader)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}

func (ob *overlayBowl) move(oldAbsolutePath string, newAbsolutePath string) error {
	debugf("mv '%s' '%s'", oldAbsolutePath, newAbsolutePath)

	err := os.Remove(newAbsolutePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return errors.Wrap(err, 0)
		}
	}

	err = os.MkdirAll(filepath.Dir(newAbsolutePath), os.FileMode(0755))
	if err != nil {
		return errors.Wrap(err, 0)
	}

	if debugBrokenRename {
		err = &os.PathError{}
	} else {
		err = os.Rename(oldAbsolutePath, newAbsolutePath)
	}
	if err != nil {
		debugf("falling back to copy because of %s", err.Error())
		if os.IsNotExist(err) {
			debugf("mhh our rename error was that old does not exist")
		}

		cErr := ob.copy(oldAbsolutePath, newAbsolutePath, mkdirBehaviorNever)
		if cErr != nil {
			return cErr
		}

		cErr = os.Remove(oldAbsolutePath)
		if cErr != nil {
			return cErr
		}
	}

	return nil
}

func (ob *overlayBowl) applyMoves() error {
	for _, moveIndex := range ob.moveFiles {
		file := ob.SourceContainer.Files[moveIndex]
		if file == nil {
			return errors.Wrap(fmt.Errorf("overlaybowl: applyMoves: no such file %d", moveIndex), 0)
		}
		debugf("applying move '%s'", file.Path)
		nativePath := filepath.FromSlash(file.Path)

		stagePath := filepath.Join(ob.StageFolder, nativePath)
		outputPath := filepath.Join(ob.OutputFolder, nativePath)
		err := ob.move(stagePath, outputPath)
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	return nil
}

func (ob *overlayBowl) applyOverlays() error {
	ctx := &overlay.OverlayPatchContext{}

	handleOverlay := func(overlayIndex int64) error {
		file := ob.SourceContainer.Files[overlayIndex]
		if file == nil {
			return errors.Wrap(fmt.Errorf("overlaybowl: applyOverlays: no such file %d", overlayIndex), 0)
		}
		debugf("applying overlay '%s'", file.Path)
		nativePath := filepath.FromSlash(file.Path)

		stagePath := filepath.Join(ob.StageFolder, nativePath)
		r, err := os.Open(stagePath)
		if err != nil {
			return errors.Wrap(err, 0)
		}
		defer r.Close()

		outputPath := filepath.Join(ob.OutputFolder, nativePath)
		w, err := os.OpenFile(outputPath, os.O_WRONLY, 0644)
		if err != nil {
			return errors.Wrap(err, 0)
		}
		defer w.Close()

		err = ctx.Patch(r, w)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		finalSize, err := w.Seek(0, io.SeekCurrent)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		err = w.Truncate(finalSize)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		return nil
	}

	for _, overlayIndex := range ob.overlayFiles {
		err := handleOverlay(overlayIndex)
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	return nil
}

// ghosts

// GhostKind determines what went missing: a file, a directory, or a symlink
type GhostKind int

const (
	// GhostKindDir indicates that a directory has disappeared between two containers
	GhostKindDir GhostKind = iota + 0xfaf0
	// GhostKindFile indicates that a file has disappeared between two containers
	GhostKindFile
	// GhostKindSymlink indicates that a symbolic link has disappeared between two containers
	GhostKindSymlink
)

// A Ghost is a file, directory, or symlink, that has disappeared from one
// container (target) to the next (source)
type Ghost struct {
	Kind GhostKind
	Path string
}

func detectGhosts(sourceContainer *tlc.Container, targetContainer *tlc.Container) []Ghost {
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
	var ghosts []Ghost
	for _, f := range targetContainer.Files {
		if !sourceFileMap[f.Path] {
			ghosts = append(ghosts, Ghost{
				Kind: GhostKindFile,
				Path: f.Path,
			})
		}
	}
	for _, s := range targetContainer.Symlinks {
		if !sourceFileMap[s.Path] {
			ghosts = append(ghosts, Ghost{
				Kind: GhostKindSymlink,
				Path: s.Path,
			})
		}
	}
	for _, d := range targetContainer.Dirs {
		if !sourceFileMap[d.Path] {
			ghosts = append(ghosts, Ghost{
				Kind: GhostKindDir,
				Path: d.Path,
			})
		}
	}
	return ghosts
}

type byDecreasingLength []Ghost

func (s byDecreasingLength) Len() int {
	return len(s)
}

func (s byDecreasingLength) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byDecreasingLength) Less(i, j int) bool {
	return len(s[j].Path) < len(s[i].Path)
}

func (ob *overlayBowl) deleteGhosts() error {
	ghosts := detectGhosts(ob.SourceContainer, ob.TargetContainer)
	debugf("%d total ghosts", len(ghosts))

	sort.Sort(byDecreasingLength(ghosts))

	for _, ghost := range ghosts {
		debugf("ghost: %v", ghost)
		op := filepath.Join(ob.OutputFolder, filepath.FromSlash(ghost.Path))

		err := os.Remove(op)
		if err == nil || os.IsNotExist(err) {
			// removed or already removed, good
			debugf("ghost removed or already gone '%s'", ghost.Path)
		} else {
			if ghost.Kind == GhostKindDir {
				// sometimes we can't delete directories, it's okay
				debugf("ghost dir left behind '%s'", ghost.Path)
			} else {
				return errors.Wrap(err, 0)
			}
		}
	}

	return nil
}

// notifyWriteCloser

type onCloseFunc func() error

type notifyWriteCloser struct {
	w       io.WriteCloser
	onClose onCloseFunc
}

var _ io.WriteCloser = (*notifyWriteCloser)(nil)

func (nwc *notifyWriteCloser) Write(buf []byte) (int, error) {
	return nwc.w.Write(buf)
}

func (nwc *notifyWriteCloser) Close() (rErr error) {
	defer func() {
		if nwc.onClose != nil {
			cErr := nwc.onClose()
			if cErr != nil && rErr == nil {
				rErr = cErr
			}
		}
	}()

	err := nwc.w.Close()
	if err != nil {
		rErr = errors.Wrap(err, 0)
		return
	}

	return
}

// overlayEntryWriter

type overlayEntryWriter struct {
	path       string
	readSeeker io.ReadSeeker
	ow         overlay.WriteFlushCloser

	readOffset  int64
	writeOffset int64
}

type OverlayEntryWriterCheckpoint struct {
	ReadOffset int64
}

func init() {
	gob.Register(&OverlayEntryWriterCheckpoint{})
}

func (oew *overlayEntryWriter) Tell() int64 {
	return oew.writeOffset
}

func (oew *overlayEntryWriter) Save() (*Checkpoint, error) {
	err := oew.ow.Flush()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	readOffset, err := oew.readSeeker.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	c := &Checkpoint{
		Offset: oew.writeOffset,
		Data: &OverlayEntryWriterCheckpoint{
			ReadOffset: readOffset,
		},
	}
	return c, nil
}

func (oew *overlayEntryWriter) Resume(c *Checkpoint) (int64, error) {
	err := os.MkdirAll(filepath.Dir(oew.path), 0755)
	if err != nil {
		return 0, errors.Wrap(err, 0)
	}

	w, err := os.OpenFile(oew.path, os.O_CREATE|os.O_WRONLY, os.FileMode(0644))
	if err != nil {
		return 0, errors.Wrap(err, 0)
	}

	if c != nil {
		// we might need to seek y'all
		oewc, ok := c.Data.(*OverlayEntryWriterCheckpoint)
		if !ok {
			return 0, errors.New("invalid checkpoint for overlayEntryWriter")
		}

		// seek the reader first
		_, err = oew.readSeeker.Seek(oewc.ReadOffset, io.SeekStart)
		if err != nil {
			return 0, errors.Wrap(err, 0)
		}

		// now the writer
		_, err = w.Seek(c.Offset, io.SeekStart)
		if err != nil {
			return 0, errors.Wrap(err, 0)
		}

		oew.writeOffset = c.Offset
		oew.readOffset = oewc.ReadOffset
	}

	r := oew.readSeeker
	oew.ow = overlay.NewOverlayWriter(r, w)

	return oew.writeOffset, nil
}

func (oew *overlayEntryWriter) Write(buf []byte) (int, error) {
	if oew.ow == nil {
		return 0, ErrUninitializedWriter
	}

	n, err := oew.ow.Write(buf)
	oew.writeOffset += int64(n)
	return n, err
}

func (oew *overlayEntryWriter) Close() error {
	if oew.ow == nil {
		return nil
	}

	ow := oew.ow
	oew.ow = nil
	err := ow.Close()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
