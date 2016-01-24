package main

import (
	"encoding/binary"
	"io"
	"io/ioutil"
	"os"

	"github.com/dustin/go-humanize"
	"gopkg.in/kothar/brotli-go.v0/enc"

	"github.com/itchio/wharf.proto/counter"
	"github.com/itchio/wharf.proto/megafile"
	"github.com/itchio/wharf.proto/rsync"
)

func writeString(w io.Writer, s string) error {
	err := binary.Write(w, binary.LittleEndian, int32(len(s)))
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(s))
	if err != nil {
		return err
	}

	return nil
}

func writeRepoInfo(w io.Writer, info *megafile.RepoInfo) {
	binary.Write(w, binary.LittleEndian, MP_REPO_INFO)

	binary.Write(w, binary.LittleEndian, MP_NUM_BLOCKS)
	binary.Write(w, binary.LittleEndian, info.NumBlocks)

	binary.Write(w, binary.LittleEndian, MP_DIRS)
	binary.Write(w, binary.LittleEndian, int32(len(info.Dirs)))
	for _, d := range info.Dirs {
		must(writeString(w, d.Path))
		binary.Write(w, binary.LittleEndian, d.Mode)
	}

	binary.Write(w, binary.LittleEndian, MP_FILES)
	binary.Write(w, binary.LittleEndian, int32(len(info.Files)))
	for _, f := range info.Files {
		must(writeString(w, f.Path))
		binary.Write(w, binary.LittleEndian, f.Mode)
		binary.Write(w, binary.LittleEndian, f.Size)
		binary.Write(w, binary.LittleEndian, f.BlockIndex)
		binary.Write(w, binary.LittleEndian, f.BlockIndexEnd)
	}

	binary.Write(w, binary.LittleEndian, MP_SYMLINKS)
	binary.Write(w, binary.LittleEndian, int32(len(info.Symlinks)))
	for _, l := range info.Symlinks {
		must(writeString(w, l.Path))
		binary.Write(w, binary.LittleEndian, l.Mode)
		must(writeString(w, l.Dest))
	}
}

func printRepoStats(info *megafile.RepoInfo, path string) {
	totalSize := int64(0)
	for _, f := range info.Files {
		totalSize += f.Size
	}

	CsvCol(totalSize, len(info.Files), len(info.Symlinks), len(info.Dirs))
	Logf("%s in %d files, %d links, %d dirs in %s", humanize.Bytes(uint64(totalSize)), len(info.Files),
		len(info.Symlinks), len(info.Dirs), path)
}

func megadiff(target string, source string, patch string, brotliQuality int) {
	// csv columns:
	// target, source, targetSize, targetFiles, targetSymlinks, targetDirs,
	// sourceSize, sourceFiles, sourceSymlinks, sourceDirs, paddedSize,
	// rawPatch, brotliPatch
	CsvCol(target, source)

	targetInfo, err := megafile.Walk(target, MP_BLOCK_SIZE)
	must(err)
	targetReader := targetInfo.NewReader(target)
	defer targetReader.Close()
	printRepoStats(targetInfo, target)

	rs := &rsync.RSync{
		BlockSize: targetInfo.BlockSize,
	}
	signature := make([]rsync.BlockHash, 0)

	targetPaddedBytes := targetInfo.NumBlocks * int64(MP_BLOCK_SIZE)
	onTargetRead := func(count int64) {
		Progress(100.0 * float64(count) / float64(targetPaddedBytes))
	}
	targetReaderCounter := counter.NewReaderCallback(onTargetRead, targetReader)

	Log("Computing source signature")
	StartProgress()

	sigWriter := func(bl rsync.BlockHash) error {
		signature = append(signature, bl)
		return nil
	}
	rs.CreateSignature(targetReaderCounter, sigWriter)

	EndProgress()

	compressedWriter, err := os.Create(patch)
	must(err)
	defer compressedWriter.Close()

	brotliCounter := counter.NewWriter(compressedWriter)
	brotliParams := enc.NewBrotliParams()
	brotliParams.SetQuality(brotliQuality)
	brotliWriter := enc.NewBrotliWriter(brotliParams, brotliCounter)

	rawCounter := counter.NewWriter(brotliWriter)
	patchWriter := rawCounter

	sourceInfo, err := megafile.Walk(source, MP_BLOCK_SIZE)
	must(err)
	sourceReader := sourceInfo.NewReader(source)
	defer sourceReader.Close()

	printRepoStats(sourceInfo, source)

	must(binary.Write(patchWriter, binary.LittleEndian, MP_MAGIC))
	writeRepoInfo(patchWriter, targetInfo)
	writeRepoInfo(patchWriter, sourceInfo)

	must(binary.Write(patchWriter, binary.LittleEndian, MP_RSYNC_OPS))

	sourcePaddedBytes := sourceInfo.NumBlocks * int64(MP_BLOCK_SIZE)
	CsvCol(sourcePaddedBytes)

	Log("Computing target->source recipe")
	StartProgress()

	onSourceRead := func(count int64) {
		Progress(100.0 * float64(count) / float64(sourcePaddedBytes))
	}
	sourceReaderCounter := counter.NewReaderCallback(onSourceRead, sourceReader)

	numOps := 0

	opsWriter := func(op rsync.Operation) error {
		numOps++
		// Logf("Writing operation, type %d, index %d - %d, data has %d bytes", op.Type, op.BlockIndex, op.BlockIndexEnd, len(op.Data))
		must(binary.Write(patchWriter, binary.LittleEndian, MP_RSYNC_OP))
		must(binary.Write(patchWriter, binary.LittleEndian, byte(op.Type)))

		switch op.Type {
		case rsync.OpBlock:
			must(binary.Write(patchWriter, binary.LittleEndian, op.BlockIndex))
		case rsync.OpBlockRange:
			must(binary.Write(patchWriter, binary.LittleEndian, op.BlockIndex))
			must(binary.Write(patchWriter, binary.LittleEndian, op.BlockIndexEnd))
		case rsync.OpData:
			must(binary.Write(patchWriter, binary.LittleEndian, int64(len(op.Data))))
			_, err := patchWriter.Write(op.Data)
			must(err)
		default:
			Dief("unknown rsync op type: %d", op.Type)
		}
		return nil
	}
	rs.CreateDelta(sourceReaderCounter, signature, opsWriter)

	must(binary.Write(patchWriter, binary.LittleEndian, MP_EOF))
	must(brotliWriter.Close())

	EndProgress()
	Logf("Wrote %d ops", numOps)

	CsvCol(rawCounter.Count(), brotliCounter.Count())

	Logf("Wrote patch to %s (%s, expands to %s)", patch,
		humanize.Bytes(uint64(brotliCounter.Count())),
		humanize.Bytes(uint64(rawCounter.Count())))

	if *megadiffArgs.verify {
		tmpDir, err := ioutil.TempDir(os.TempDir(), "megadiff")
		must(err)
		defer os.RemoveAll(tmpDir)

		Logf("Verifying patch by rebuilding source in %s", tmpDir)
		megapatch(patch, source, tmpDir)
	}
}
