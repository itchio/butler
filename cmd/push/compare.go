package push

import (
	"bytes"
	"context"
	"fmt"
	"sort"

	"github.com/itchio/butler/comm"

	"github.com/itchio/headway/state"

	"github.com/itchio/lake"
	"github.com/itchio/lake/tlc"

	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/wsync"

	"github.com/pkg/errors"
)

// pushComparisonCounts is the structured per-entry classification summary
// emitted on the result event from `butler push-preview --json`. Counts
// cover every entry (files, dirs, symlinks together); byte totals only
// reflect file sizes — dirs and symlinks contribute zero. Deleted bytes
// reflect the size of the entries on the previous build that wouldn't
// exist in the new push.
type pushComparisonCounts struct {
	New      int `json:"new"`
	Modified int `json:"modified"`
	Deleted  int `json:"deleted"`
	Same     int `json:"same"`

	NewBytes      int64 `json:"newBytes"`
	ModifiedBytes int64 `json:"modifiedBytes"`
	DeletedBytes  int64 `json:"deletedBytes"`
	SameBytes     int64 `json:"sameBytes"`
}

// topChangedFileEntry is a single row in the "biggest changes" list
// emitted on the result event. Mirrors PublishPushPreviewEntry on the
// butlerd side; kept as a separate type here to keep cmd/push free of
// butlerd imports.
type topChangedFileEntry struct {
	Path   string `json:"path"`
	Status string `json:"status"`
	Size   int64  `json:"size"`
}

// maxTopChangedFiles caps the result-event payload so a 50k-file project
// doesn't bloat the JSON. 20 per category is enough to spot the heavy
// hitters without turning the listing into its own UI problem on the
// client.
const maxTopChangedFiles = 20

// topChangedFiles is the per-category "biggest changes" listing emitted
// on the result event. Each list is capped at maxTopChangedFiles and
// sorted by size desc, path asc; the renderer reconstructs a merged
// "biggest changes overall" view by combining the three lists. Mirrors
// PublishPushTopChangedFiles on the butlerd side.
type topChangedFiles struct {
	New      []topChangedFileEntry `json:"new"`
	Modified []topChangedFileEntry `json:"modified"`
	Deleted  []topChangedFileEntry `json:"deleted"`
}

// computeTopChangedFiles returns up to maxTopChangedFiles entries per
// category (NEW, MODIFIED, DELETED), sorted by size descending. Dirs and
// symlinks are excluded — they have no meaningful size. SAME entries are
// excluded — they aren't part of the change set the caller cares about.
//
// Size is taken from the source side for new/modified, from the target
// side for deleted (the entry doesn't exist in source). Ties are broken
// by path for deterministic ordering.
func computeTopChangedFiles(result *comparisonResult) topChangedFiles {
	var changed []topChangedFileEntry
	for _, e := range result.Files {
		switch e.Status {
		case statusNew:
			if e.SourceFile != nil {
				changed = append(changed, topChangedFileEntry{
					Path:   e.Path,
					Status: "new",
					Size:   e.SourceFile.Size,
				})
			}
		case statusModified:
			if e.SourceFile != nil {
				changed = append(changed, topChangedFileEntry{
					Path:   e.Path,
					Status: "modified",
					Size:   e.SourceFile.Size,
				})
			}
		case statusDeleted:
			if e.TargetFile != nil {
				changed = append(changed, topChangedFileEntry{
					Path:   e.Path,
					Status: "deleted",
					Size:   e.TargetFile.Size,
				})
			}
		}
	}
	sort.SliceStable(changed, func(i, j int) bool {
		if changed[i].Size != changed[j].Size {
			return changed[i].Size > changed[j].Size
		}
		return changed[i].Path < changed[j].Path
	})
	out := topChangedFiles{
		New:      []topChangedFileEntry{},
		Modified: []topChangedFileEntry{},
		Deleted:  []topChangedFileEntry{},
	}
	for _, e := range changed {
		switch e.Status {
		case "new":
			if len(out.New) < maxTopChangedFiles {
				out.New = append(out.New, e)
			}
		case "modified":
			if len(out.Modified) < maxTopChangedFiles {
				out.Modified = append(out.Modified, e)
			}
		case "deleted":
			if len(out.Deleted) < maxTopChangedFiles {
				out.Deleted = append(out.Deleted, e)
			}
		}
	}
	return out
}

type fileStatus int

const (
	statusSame fileStatus = iota
	statusNew
	statusModified
	statusDeleted
)

func (s fileStatus) tag() string {
	switch s {
	case statusNew:
		return "NEW"
	case statusModified:
		return "MODIFIED"
	case statusDeleted:
		return "DELETED"
	default:
		return "SAME"
	}
}

// statusTagWidth keeps the per-line columns aligned. Longest tag is "MODIFIED"
// (8 chars) plus two trailing spaces for readability.
const statusTagWidth = 10

// entryComparison records the classification for a single file, dir, or
// symlink. Source/Target pointers are populated for diagnostics; the choice
// of which is non-nil follows from Status.
type entryComparison struct {
	Status fileStatus
	Path   string

	SourceFile *tlc.File
	TargetFile *tlc.File

	SourceDir *tlc.Dir
	TargetDir *tlc.Dir

	SourceSymlink *tlc.Symlink
	TargetSymlink *tlc.Symlink
}

type comparisonResult struct {
	Files    []entryComparison
	Dirs     []entryComparison
	Symlinks []entryComparison

	Counts pushComparisonCounts
}

// compareContainers classifies every entry in source vs target as
// NEW / MODIFIED / SAME / DELETED. Files in both containers are compared
// block-by-block using the wharf signature, so reporting is unambiguous.
//
// sourcePool is consumed by pwr.ComputeSignature and will be closed by the
// time this function returns.
func compareContainers(
	ctx context.Context,
	sourceContainer *tlc.Container,
	sourcePool lake.Pool,
	targetSig *pwr.SignatureInfo,
	consumer *state.Consumer,
) (*comparisonResult, error) {
	comm.Opf("Hashing source files...")
	hashConsumer := *consumer
	hashConsumer.OnProgress = func(progress float64) {
		readBytes := int64(float64(sourceContainer.Size) * progress)
		comm.ProgressWith(progress, comm.JsonMessage{
			"readBytes":     readBytes,
			"totalBytes":    sourceContainer.Size,
			"uploadedBytes": int64(0),
			"patchBytes":    int64(0),
		})
	}

	comm.StartProgressWithTotalBytes(sourceContainer.Size)
	sourceHashes, err := pwr.ComputeSignature(ctx, sourceContainer, sourcePool, &hashConsumer)
	comm.EndProgress()
	if err != nil {
		return nil, errors.Wrap(err, "computing source signature")
	}

	sourceSig := &pwr.SignatureInfo{
		Container: sourceContainer,
		Hashes:    sourceHashes,
	}

	sourceHashInfo, err := pwr.ComputeHashInfo(sourceSig)
	if err != nil {
		return nil, errors.Wrap(err, "indexing source hashes")
	}

	targetHashInfo, err := pwr.ComputeHashInfo(targetSig)
	if err != nil {
		return nil, errors.Wrap(err, "indexing target hashes")
	}

	result := &comparisonResult{}

	targetFiles := indexFiles(targetSig.Container.Files)
	for srcIdx, sf := range sourceContainer.Files {
		entry := entryComparison{Path: sf.Path, SourceFile: sf}
		if tgtIdx, ok := targetFiles[sf.Path]; ok {
			tf := targetSig.Container.Files[tgtIdx]
			entry.TargetFile = tf
			if filesEqual(sf, tf, sourceHashInfo.Groups[int64(srcIdx)], targetHashInfo.Groups[int64(tgtIdx)]) {
				entry.Status = statusSame
			} else {
				entry.Status = statusModified
			}
		} else {
			entry.Status = statusNew
		}
		result.Files = append(result.Files, entry)
	}
	sourceFiles := indexFiles(sourceContainer.Files)
	for _, tf := range targetSig.Container.Files {
		if _, ok := sourceFiles[tf.Path]; ok {
			continue
		}
		result.Files = append(result.Files, entryComparison{
			Status:     statusDeleted,
			Path:       tf.Path,
			TargetFile: tf,
		})
	}

	targetDirs := indexDirs(targetSig.Container.Dirs)
	for _, sd := range sourceContainer.Dirs {
		entry := entryComparison{Path: sd.Path, SourceDir: sd}
		if tgtIdx, ok := targetDirs[sd.Path]; ok {
			td := targetSig.Container.Dirs[tgtIdx]
			entry.TargetDir = td
			if sd.Mode == td.Mode {
				entry.Status = statusSame
			} else {
				entry.Status = statusModified
			}
		} else {
			entry.Status = statusNew
		}
		result.Dirs = append(result.Dirs, entry)
	}
	sourceDirs := indexDirs(sourceContainer.Dirs)
	for _, td := range targetSig.Container.Dirs {
		if _, ok := sourceDirs[td.Path]; ok {
			continue
		}
		result.Dirs = append(result.Dirs, entryComparison{
			Status:    statusDeleted,
			Path:      td.Path,
			TargetDir: td,
		})
	}

	targetSymlinks := indexSymlinks(targetSig.Container.Symlinks)
	for _, ss := range sourceContainer.Symlinks {
		entry := entryComparison{Path: ss.Path, SourceSymlink: ss}
		if tgtIdx, ok := targetSymlinks[ss.Path]; ok {
			ts := targetSig.Container.Symlinks[tgtIdx]
			entry.TargetSymlink = ts
			if ss.Mode == ts.Mode && ss.Dest == ts.Dest {
				entry.Status = statusSame
			} else {
				entry.Status = statusModified
			}
		} else {
			entry.Status = statusNew
		}
		result.Symlinks = append(result.Symlinks, entry)
	}
	sourceSymlinks := indexSymlinks(sourceContainer.Symlinks)
	for _, ts := range targetSig.Container.Symlinks {
		if _, ok := sourceSymlinks[ts.Path]; ok {
			continue
		}
		result.Symlinks = append(result.Symlinks, entryComparison{
			Status:        statusDeleted,
			Path:          ts.Path,
			TargetSymlink: ts,
		})
	}

	for _, group := range [][]entryComparison{result.Dirs, result.Symlinks, result.Files} {
		for _, e := range group {
			switch e.Status {
			case statusNew:
				result.Counts.New++
			case statusModified:
				result.Counts.Modified++
			case statusDeleted:
				result.Counts.Deleted++
			case statusSame:
				result.Counts.Same++
			}
		}
	}
	// Bytes are only meaningful for files. NewBytes/SameBytes/ModifiedBytes
	// come from the source side; DeletedBytes from the target side (the
	// entry doesn't exist in source).
	for _, e := range result.Files {
		switch e.Status {
		case statusNew:
			if e.SourceFile != nil {
				result.Counts.NewBytes += e.SourceFile.Size
			}
		case statusModified:
			if e.SourceFile != nil {
				result.Counts.ModifiedBytes += e.SourceFile.Size
			}
		case statusDeleted:
			if e.TargetFile != nil {
				result.Counts.DeletedBytes += e.TargetFile.Size
			}
		case statusSame:
			if e.SourceFile != nil {
				result.Counts.SameBytes += e.SourceFile.Size
			}
		}
	}

	sortEntries(result.Files)
	sortEntries(result.Dirs)
	sortEntries(result.Symlinks)

	return result, nil
}

// allNewFromContainer builds a result where every source entry is NEW. Used
// when the channel has no parent build to compare against.
func allNewFromContainer(c *tlc.Container) *comparisonResult {
	result := &comparisonResult{}
	for _, d := range c.Dirs {
		result.Dirs = append(result.Dirs, entryComparison{Status: statusNew, Path: d.Path, SourceDir: d})
	}
	for _, s := range c.Symlinks {
		result.Symlinks = append(result.Symlinks, entryComparison{Status: statusNew, Path: s.Path, SourceSymlink: s})
	}
	for _, f := range c.Files {
		result.Files = append(result.Files, entryComparison{Status: statusNew, Path: f.Path, SourceFile: f})
		result.Counts.NewBytes += f.Size
	}
	result.Counts.New = len(result.Dirs) + len(result.Symlinks) + len(result.Files)
	return result
}

// printComparison writes the per-entry classification using comm.Logf so it
// flows through the same channel as the existing dry-run listing (and gets
// JSON-encoded automatically when --json is set). When changesOnly is true,
// SAME entries are skipped from the listing — counts in the summary still
// reflect every entry.
func printComparison(result *comparisonResult, changesOnly bool) {
	emit := func(entries []entryComparison) {
		for _, e := range entries {
			if changesOnly && e.Status == statusSame {
				continue
			}
			comm.Logf("%s", formatEntry(e))
		}
	}
	emit(result.Dirs)
	emit(result.Symlinks)
	emit(result.Files)
}

func formatEntry(e entryComparison) string {
	tag := fmt.Sprintf("%-*s", statusTagWidth, e.Status.tag())
	switch {
	case e.SourceFile != nil:
		return tag + e.SourceFile.ToString()
	case e.TargetFile != nil:
		return tag + e.TargetFile.ToString()
	case e.SourceDir != nil:
		return tag + e.SourceDir.ToString()
	case e.TargetDir != nil:
		return tag + e.TargetDir.ToString()
	case e.SourceSymlink != nil:
		return tag + e.SourceSymlink.ToString()
	case e.TargetSymlink != nil:
		return tag + e.TargetSymlink.ToString()
	default:
		return tag + e.Path
	}
}

func indexFiles(files []*tlc.File) map[string]int {
	out := make(map[string]int, len(files))
	for i, f := range files {
		out[f.Path] = i
	}
	return out
}

func indexDirs(dirs []*tlc.Dir) map[string]int {
	out := make(map[string]int, len(dirs))
	for i, d := range dirs {
		out[d.Path] = i
	}
	return out
}

func indexSymlinks(symlinks []*tlc.Symlink) map[string]int {
	out := make(map[string]int, len(symlinks))
	for i, s := range symlinks {
		out[s.Path] = i
	}
	return out
}

// statusOrder ranks classifications so the most actionable changes (NEW,
// MODIFIED, DELETED) appear before the noisy SAME tail.
var statusOrder = map[fileStatus]int{
	statusNew:      0,
	statusModified: 1,
	statusDeleted:  2,
	statusSame:     3,
}

func sortEntries(entries []entryComparison) {
	sort.SliceStable(entries, func(i, j int) bool {
		if statusOrder[entries[i].Status] != statusOrder[entries[j].Status] {
			return statusOrder[entries[i].Status] < statusOrder[entries[j].Status]
		}
		return entries[i].Path < entries[j].Path
	})
}

func filesEqual(sf, tf *tlc.File, sourceGroup, targetGroup []wsync.BlockHash) bool {
	if sf.Size != tf.Size {
		return false
	}
	if sf.Mode != tf.Mode {
		return false
	}
	return blockGroupsEqual(sourceGroup, targetGroup)
}

// blockGroupsEqual compares two per-file slices of wsync block hashes for
// content equality. ShortSize covers the trailing partial block; StrongHash
// is sufficient for content equality (weak hashes are only useful for
// finding rolling matches during diffing).
func blockGroupsEqual(a, b []wsync.BlockHash) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].ShortSize != b[i].ShortSize {
			return false
		}
		if !bytes.Equal(a[i].StrongHash, b[i].StrongHash) {
			return false
		}
	}
	return true
}
