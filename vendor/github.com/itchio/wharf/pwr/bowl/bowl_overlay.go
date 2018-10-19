package bowl

import (
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/itchio/savior/filesource"
	"github.com/itchio/wharf/pwr/overlay"

	"github.com/itchio/wharf/pools/fspool"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wsync"
	"github.com/pkg/errors"
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
	targetFilesByPath map[string]int64

	// files we'll have to move
	transpositions []Transposition
	// files we'll have to patch using an overlay (indices in SourceContainer)
	overlayFiles []int64
	// files we'll have to move from the staging folder to the dest
	moveFiles []int64
}

type OverlayBowlCheckpoint struct {
	Transpositions []Transposition
	OverlayFiles   []int64
	MoveFiles      []int64
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

	{
		if params.OutputFolder == "" {
			return nil, errors.New("overlaybowl: OutputFolder must not be nil")
		}

		stats, err := os.Stat(params.OutputFolder)
		if err != nil {
			return nil, errors.New("overlaybowl: OutputFolder must exist")
		}

		if !stats.IsDir() {
			return nil, errors.New("overlaybowl: OutputFolder must exist and be a directory")
		}
	}

	if params.StageFolder == "" {
		return nil, errors.New("overlaybowl: StageFolder must not be nil")
	}

	var err error

	err = os.MkdirAll(params.StageFolder, 0755)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	targetPool := fspool.New(params.TargetContainer, params.OutputFolder)
	stagePool := fspool.New(params.SourceContainer, params.StageFolder)

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

func (b *overlayBowl) Save() (*BowlCheckpoint, error) {
	c := &BowlCheckpoint{
		Data: &OverlayBowlCheckpoint{
			MoveFiles:      b.moveFiles,
			OverlayFiles:   b.overlayFiles,
			Transpositions: b.transpositions,
		},
	}
	return c, nil
}

func (b *overlayBowl) Resume(c *BowlCheckpoint) error {
	if c == nil {
		return nil
	}

	if cc, ok := c.Data.(*OverlayBowlCheckpoint); ok {
		b.transpositions = cc.Transpositions
		b.moveFiles = cc.MoveFiles
		b.overlayFiles = cc.OverlayFiles
	}
	return nil
}

func (b *overlayBowl) GetWriter(sourceFileIndex int64) (EntryWriter, error) {
	sourceFile := b.SourceContainer.Files[sourceFileIndex]
	if sourceFile == nil {
		return nil, errors.Errorf("overlayBowl: unknown source file %d", sourceFileIndex)
	}

	if targetIndex, ok := b.targetFilesByPath[sourceFile.Path]; ok {
		debugf("returning overlay writer for '%s'", sourceFile.Path)

		// oh damn, that file already exists in the output - let's make an overlay
		b.markOverlay(sourceFileIndex)

		r, err := b.TargetPool.GetReadSeeker(targetIndex)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		wPath := b.stagePool.GetPath(sourceFileIndex)
		return &overlayEntryWriter{path: wPath, readSeeker: r}, nil
	}

	// guess it's a new file! let's write it to staging anyway
	b.markMove(sourceFileIndex)

	debugf("returning move writer for '%s'", sourceFile.Path)

	wPath := b.stagePool.GetPath(sourceFileIndex)
	return &freshEntryWriter{path: wPath}, nil
}

func (b *overlayBowl) markOverlay(sourceFileIndex int64) {
	// make sure we don't double mark it
	for _, i := range b.overlayFiles {
		if i == sourceFileIndex {
			// oh cool it's already marked
			return
		}
	}

	// mark it
	b.overlayFiles = append(b.overlayFiles, sourceFileIndex)
}

func (b *overlayBowl) markMove(index int64) {
	// make sure we don't double mark it
	for _, i := range b.moveFiles {
		if i == index {
			// oh cool it's already marked
			return
		}
	}

	// mark it
	b.moveFiles = append(b.moveFiles, index)
}

func (b *overlayBowl) Transpose(t Transposition) error {
	// ok, say we resumed, maybe we already have a transposition for this source file?
	for i, tt := range b.transpositions {
		if tt.SourceIndex == t.SourceIndex {
			// and so we do! let's replace it.
			b.transpositions[i] = t
			return nil
		}
	}

	// if we didn't already have one, let's record it for when we commit
	b.transpositions = append(b.transpositions, t)
	return nil
}

func (b *overlayBowl) Commit() error {
	// oy, do we have work to do!
	var err error

	// - close the target pool, in case it still has a reader open!
	err = b.TargetPool.Close()
	if err != nil {
		return errors.WithStack(err)
	}

	// - same with stage pool, we might have it open for overlay purposes
	err = b.stagePool.Close()
	if err != nil {
		return errors.WithStack(err)
	}

	// - ensure dirs and symlinks
	err = b.ensureDirsAndSymlinks()
	if err != nil {
		return errors.WithStack(err)
	}

	// - apply transpositions
	err = b.applyTranspositions()
	if err != nil {
		return errors.WithStack(err)
	}

	// - move files we need to move
	err = b.applyMoves()
	if err != nil {
		return errors.WithStack(err)
	}

	// - merge overlays
	err = b.applyOverlays()
	if err != nil {
		return errors.WithStack(err)
	}

	// - delete ghosts
	err = b.deleteGhosts()
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (b *overlayBowl) ensureDirsAndSymlinks() error {
	outputPath := b.OutputFolder

	for _, dir := range b.SourceContainer.Dirs {
		path := filepath.Join(outputPath, filepath.FromSlash(dir.Path))

		err := os.MkdirAll(path, 0755)
		if err != nil {
			// If path is already a directory, MkdirAll does nothing and returns nil.
			// so if we get a non-nil error, we know it's serious business (permissions, etc.)
			return err
		}
	}

	// TODO: behave like github.com/itchio/savior for symlinks on windows ?

	for _, symlink := range b.SourceContainer.Symlinks {
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

type transpoBehavior int

const (
	transpoBehaviorMove transpoBehavior = 0x1923 + iota
	transpoBehaviorCopy
)

func (b *overlayBowl) applyTranspositions() error {
	transpositions := make(map[string][]*pathTranspo)
	outputPath := b.OutputFolder

	for _, t := range b.transpositions {
		targetFile := b.TargetContainer.Files[t.TargetIndex]
		sourceFile := b.SourceContainer.Files[t.SourceIndex]

		transpositions[targetFile.Path] = append(transpositions[targetFile.Path], &pathTranspo{
			TargetPath: targetFile.Path,
			OutputPath: sourceFile.Path,
		})
	}

	applyMultipleTranspositions := func(behavior transpoBehavior, targetPath string, group []*pathTranspo) error {
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
			err := b.copy(oldAbsolutePath, newAbsolutePath, mkdirBehaviorIfNeeded)
			if err != nil {
				return errors.WithStack(err)
			}
		}

		if noop == nil {
			// we treated the first transpo as being the rename, gotta do it now
			transpo := group[0]
			oldAbsolutePath := filepath.Join(outputPath, filepath.FromSlash(targetPath))
			newAbsolutePath := filepath.Join(outputPath, filepath.FromSlash(transpo.OutputPath))

			switch behavior {
			case transpoBehaviorCopy:
				// no, wait, the target file is itself being patched, meaning it has a pending overlay.
				// in order for that overlay to apply cleanly, we must copy the file, not move it.
				// we should also not need mkdir, since we already ensured dirs and symlinks.
				err := b.copy(oldAbsolutePath, newAbsolutePath, mkdirBehaviorNever)
				if err != nil {
					return errors.WithStack(err)
				}
			case transpoBehaviorMove:
				err := b.move(oldAbsolutePath, newAbsolutePath)
				if err != nil {
					return errors.WithStack(err)
				}
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
				// transpo is doing A=>B, and another transpo is doing B=>C
				// instead, have transpo do A=>B2, the other do B=>C
				// then have a cleanup phase rename B2 to B
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

	overlayFilesByPath := make(map[string]bool)
	for _, overlayFileSourceIndex := range b.overlayFiles {
		f := b.SourceContainer.Files[overlayFileSourceIndex]
		overlayFilesByPath[f.Path] = true
	}

	for groupTargetPath, group := range transpositions {
		if alreadyDone[groupTargetPath] {
			continue
		}
		alreadyDone[groupTargetPath] = true

		behavior := transpoBehaviorMove
		_, hasPendingOverlay := overlayFilesByPath[groupTargetPath]
		if hasPendingOverlay {
			// if the target file is itself patched (it has a pending overlay),
			// then it must never be renamed to something else, only copied.
			behavior = transpoBehaviorCopy
		}

		if len(group) == 1 {
			transpo := group[0]
			if transpo.TargetPath == transpo.OutputPath {
				// file wasn't touched at all
			} else {
				// file was renamed
				oldAbsolutePath := filepath.Join(outputPath, filepath.FromSlash(transpo.TargetPath))
				newAbsolutePath := filepath.Join(outputPath, filepath.FromSlash(transpo.OutputPath))

				switch behavior {
				case transpoBehaviorCopy:
					// we should never need to mkdir, because we already ensured dirs and symlinks.
					err := b.copy(oldAbsolutePath, newAbsolutePath, mkdirBehaviorNever)
					if err != nil {
						return errors.WithStack(err)
					}
				case transpoBehaviorMove:
					err := b.move(oldAbsolutePath, newAbsolutePath)
					if err != nil {
						return errors.WithStack(err)
					}
				}
			}
		} else {
			err := applyMultipleTranspositions(behavior, groupTargetPath, group)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}

	for _, rename := range cleanupRenames {
		oldAbsolutePath := filepath.Join(outputPath, filepath.FromSlash(rename.TargetPath))
		newAbsolutePath := filepath.Join(outputPath, filepath.FromSlash(rename.OutputPath))
		err := b.move(oldAbsolutePath, newAbsolutePath)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func (b *overlayBowl) copy(oldAbsolutePath string, newAbsolutePath string, mkdirBehavior mkdirBehavior) error {
	debugf("cp '%s' '%s'", oldAbsolutePath, newAbsolutePath)
	if mkdirBehavior == mkdirBehaviorIfNeeded {
		err := os.MkdirAll(filepath.Dir(newAbsolutePath), os.FileMode(0755))
		if err != nil {
			return errors.WithStack(err)
		}
	}

	// fall back to copy + remove
	reader, err := os.Open(oldAbsolutePath)
	if err != nil {
		return errors.WithStack(err)
	}
	defer reader.Close()

	stats, err := reader.Stat()
	if err != nil {
		return errors.WithStack(err)
	}

	writer, err := os.OpenFile(newAbsolutePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, stats.Mode()|tlc.ModeMask)
	if err != nil {
		return errors.WithStack(err)
	}
	defer writer.Close()

	_, err = io.Copy(writer, reader)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (b *overlayBowl) move(oldAbsolutePath string, newAbsolutePath string) error {
	debugf("mv '%s' '%s'", oldAbsolutePath, newAbsolutePath)

	err := os.Remove(newAbsolutePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return errors.WithStack(err)
		}
	}

	err = os.MkdirAll(filepath.Dir(newAbsolutePath), os.FileMode(0755))
	if err != nil {
		return errors.WithStack(err)
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

		cErr := b.copy(oldAbsolutePath, newAbsolutePath, mkdirBehaviorNever)
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

func (b *overlayBowl) applyMoves() error {
	for _, moveIndex := range b.moveFiles {
		file := b.SourceContainer.Files[moveIndex]
		if file == nil {
			return errors.Errorf("overlaybowl: applyMoves: no such file %d", moveIndex)
		}
		debugf("applying move '%s'", file.Path)
		nativePath := filepath.FromSlash(file.Path)

		stagePath := filepath.Join(b.StageFolder, nativePath)
		outputPath := filepath.Join(b.OutputFolder, nativePath)
		err := b.move(stagePath, outputPath)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func (b *overlayBowl) applyOverlays() error {
	ctx := &overlay.OverlayPatchContext{}

	handleOverlay := func(overlaySourceFileIndex int64) error {
		file := b.SourceContainer.Files[overlaySourceFileIndex]
		if file == nil {
			return errors.Errorf("overlaybowl: applyOverlays: no such file %d", overlaySourceFileIndex)
		}
		debugf("applying overlay '%s'", file.Path)
		nativePath := filepath.FromSlash(file.Path)

		stagePath := filepath.Join(b.StageFolder, nativePath)
		r, err := filesource.Open(stagePath)
		if err != nil {
			return errors.WithStack(err)
		}
		defer r.Close()

		outputPath := filepath.Join(b.OutputFolder, nativePath)
		w, err := os.OpenFile(outputPath, os.O_WRONLY, 0644)
		if err != nil {
			return errors.WithStack(err)
		}
		defer w.Close()

		err = ctx.Patch(r, w)
		if err != nil {
			return errors.WithStack(err)
		}

		finalSize, err := w.Seek(0, io.SeekCurrent)
		if err != nil {
			return errors.WithStack(err)
		}

		err = w.Truncate(finalSize)
		if err != nil {
			return errors.WithStack(err)
		}

		return nil
	}

	for _, overlayIndex := range b.overlayFiles {
		err := handleOverlay(overlayIndex)
		if err != nil {
			return errors.WithStack(err)
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

func (b *overlayBowl) deleteGhosts() error {
	ghosts := detectGhosts(b.SourceContainer, b.TargetContainer)
	debugf("%d total ghosts", len(ghosts))

	sort.Sort(byDecreasingLength(ghosts))

	for _, ghost := range ghosts {
		debugf("ghost: %v", ghost)
		op := filepath.Join(b.OutputFolder, filepath.FromSlash(ghost.Path))

		err := os.Remove(op)
		if err == nil || os.IsNotExist(err) {
			// removed or already removed, good
			debugf("ghost removed or already gone '%s'", ghost.Path)
		} else {
			if ghost.Kind == GhostKindDir {
				// sometimes we can't delete directories, it's okay
				debugf("ghost dir left behind '%s'", ghost.Path)
			} else {
				return errors.WithStack(err)
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
		rErr = errors.WithStack(err)
		return
	}

	return
}

// overlayEntryWriter

type overlayEntryWriter struct {
	path       string
	readSeeker io.ReadSeeker
	file       *os.File
	overlay    overlay.OverlayWriter

	// this is how far into the source (new) file we are.
	// it doesn't correspond with `OverlayOffset`, which is
	// how many bytes of output the OverlayWriter has produced.
	sourceOffset int64
}

type OverlayEntryWriterCheckpoint struct {
	// This offset is how many bytes we've written into the
	// overlay, not how many bytes into the new file we are.
	OverlayOffset int64

	// This offset is how many bytes we've read from the target (old) file
	ReadOffset int64
}

func (w *overlayEntryWriter) Tell() int64 {
	return w.sourceOffset
}

func (w *overlayEntryWriter) Save() (*WriterCheckpoint, error) {
	err := w.overlay.Flush()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = w.file.Sync()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	debugf("saving checkpoint: Offset = %d, ReadOffset = %d, OverlayOffset = %d",
		w.sourceOffset, w.overlay.ReadOffset(), w.overlay.OverlayOffset())

	c := &WriterCheckpoint{
		Offset: w.sourceOffset,
		Data: &OverlayEntryWriterCheckpoint{
			ReadOffset:    w.overlay.ReadOffset(),
			OverlayOffset: w.overlay.OverlayOffset(),
		},
	}
	return c, nil
}

func (w *overlayEntryWriter) Resume(c *WriterCheckpoint) (int64, error) {
	err := os.MkdirAll(filepath.Dir(w.path), 0755)
	if err != nil {
		return 0, errors.WithStack(err)
	}

	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY, os.FileMode(0644))
	if err != nil {
		return 0, errors.WithStack(err)
	}
	w.file = f

	if c != nil {
		// we might need to seek y'all
		cc, ok := c.Data.(*OverlayEntryWriterCheckpoint)
		if !ok {
			return 0, errors.New("invalid checkpoint for overlayEntryWriter")
		}

		// seek the reader first
		r := w.readSeeker
		_, err = r.Seek(cc.ReadOffset, io.SeekStart)
		if err != nil {
			return 0, errors.WithStack(err)
		}

		// now the writer
		_, err = f.Seek(cc.OverlayOffset, io.SeekStart)
		if err != nil {
			return 0, errors.WithStack(err)
		}

		w.sourceOffset = c.Offset

		debugf("making overlaywriter with ReadOffset %d, OverlayOffset %d", cc.ReadOffset, cc.OverlayOffset)
		w.overlay, err = overlay.NewOverlayWriter(r, cc.ReadOffset, f, cc.OverlayOffset)
		if err != nil {
			return 0, errors.WithStack(err)
		}
	} else {
		// the pool we got the readSeeker from doesn't need to give us a reader from 0,
		// so we need to seek here
		_, err = w.readSeeker.Seek(0, io.SeekStart)
		if err != nil {
			return 0, errors.WithStack(err)
		}

		r := w.readSeeker
		debugf("making overlaywriter with 0 ReadOffset and OverlayOffset")
		w.overlay, err = overlay.NewOverlayWriter(r, 0, f, 0)
		if err != nil {
			return 0, errors.WithStack(err)
		}
	}

	return w.sourceOffset, nil
}

func (w *overlayEntryWriter) Write(buf []byte) (int, error) {
	if w.overlay == nil {
		return 0, ErrUninitializedWriter
	}

	n, err := w.overlay.Write(buf)
	w.sourceOffset += int64(n)
	return n, err
}

func (w *overlayEntryWriter) Finalize() error {
	if w.overlay != nil {
		err := w.overlay.Finalize()
		if err != nil {
			return errors.WithMessage(err, "finalizing overlay writer")
		}
		w.overlay = nil
	}

	err := w.file.Sync()
	if err != nil {
		return errors.WithMessage(err, "syncing overlay patch file")
	}

	return nil
}

func (w *overlayEntryWriter) Close() error {
	return w.file.Close()
}

func init() {
	gob.Register(&OverlayEntryWriterCheckpoint{})
	gob.Register(&OverlayBowlCheckpoint{})
}
