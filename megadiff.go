package main

import (
	"compress/gzip"
	"encoding/binary"
	"io"
	"os"

	"github.com/cheggaaa/pb"
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
		binary.Write(w, binary.LittleEndian, l.Dest)
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

func megadiff(target string, source string, patch string) {
	// csv columns:
	// target, source, targetSize, targetFiles, targetSymlinks, targetDirs,
	// sourceSize, sourceFiles, sourceSymlinks, sourceDirs, paddedSize,
	// rawPatch, gzipPatch, brotli1Patch, brotli9Patch
	CsvCol(target, source)

	blockSize := 16 * 1024

	targetInfo, err := megafile.Walk(target, blockSize)
	must(err)
	targetReader := targetInfo.NewReader(target)
	defer targetReader.Close()
	printRepoStats(targetInfo, target)

	rs := &rsync.RSync{
		BlockSize: targetInfo.BlockSize,
	}
	signature := make([]rsync.BlockHash, 0)

	sigWriter := func(bl rsync.BlockHash) error {
		signature = append(signature, bl)
		return nil
	}
	rs.CreateSignature(targetReader, sigWriter)

	compressedWriter, err := os.Create(patch)
	must(err)
	defer compressedWriter.Close()

	brotliCounter := counter.NewWriter(compressedWriter)
	brotliCounter9 := counter.NewWriter(nil)
	gzipCounter := counter.NewWriter(nil)

	brotliParams := enc.NewBrotliParams()
	brotliParams.SetQuality(1)
	brotliWriter := enc.NewBrotliWriter(brotliParams, brotliCounter)

	brotliParams9 := enc.NewBrotliParams()
	brotliParams9.SetQuality(9)
	brotliWriter9 := enc.NewBrotliWriter(brotliParams9, brotliCounter9)

	gzipWriter, err := gzip.NewWriterLevel(gzipCounter, 1)
	must(err)

	multiWriter := io.MultiWriter(brotliWriter, brotliWriter9, gzipWriter)
	rawCounter := counter.NewWriter(multiWriter)
	patchWriter := rawCounter

	sourceInfo, err := megafile.Walk(source, blockSize)
	must(err)
	sourceReader := sourceInfo.NewReader(source)
	defer sourceReader.Close()

	printRepoStats(sourceInfo, source)

	must(binary.Write(patchWriter, binary.LittleEndian, MP_MAGIC))
	writeRepoInfo(patchWriter, targetInfo)
	writeRepoInfo(patchWriter, sourceInfo)

	must(binary.Write(patchWriter, binary.LittleEndian, MP_RSYNC_OPS))

	paddedBytes := sourceInfo.NumBlocks * int64(blockSize)
	CsvCol(paddedBytes)

	bar := pb.New(int(paddedBytes))
	bar.SetUnits(pb.U_BYTES)
	bar.SetMaxWidth(80)
	if !*appArgs.csv {
		bar.Start()
	}

	onRead := func(count int64) {
		bar.Set64(count)
	}
	sourceReaderCounter := counter.NewReaderCallback(onRead, sourceReader)

	opsWriter := func(op rsync.Operation) error {
		// Logf("Writing operation, type %d, index %d - %d, data has %d bytes", op.Type, op.BlockIndex, op.BlockIndexEnd, len(op.Data))
		must(binary.Write(patchWriter, binary.LittleEndian, MP_RSYNC_OP))
		must(binary.Write(patchWriter, binary.LittleEndian, byte(op.Type)))
		must(binary.Write(patchWriter, binary.LittleEndian, op.BlockIndex))
		must(binary.Write(patchWriter, binary.LittleEndian, op.BlockIndexEnd))
		must(binary.Write(patchWriter, binary.LittleEndian, op.Data))
		return nil
	}
	rs.CreateDelta(sourceReaderCounter, signature, opsWriter)

	must(binary.Write(patchWriter, binary.LittleEndian, MP_EOF))

	must(gzipWriter.Close())
	must(brotliWriter.Close())
	must(brotliWriter9.Close())

	if !*appArgs.csv {
		bar.Finish()
	}

	CsvCol(rawCounter.Count(), gzipCounter.Count(), brotliCounter.Count(), brotliCounter9.Count())

	Logf("Wrote compressed patch to %s. Sizes:", patch)
	Logf(" - %s (raw)", humanize.Bytes(uint64(rawCounter.Count())))
	Logf(" - %s (gzip q1)", humanize.Bytes(uint64(gzipCounter.Count())))
	Logf(" - %s (brotli q1)", humanize.Bytes(uint64(brotliCounter.Count())))
	Logf(" - %s (brotli q9)", humanize.Bytes(uint64(brotliCounter9.Count())))

	if *megadiffArgs.verify {
		tmpDir := os.TempDir()
		defer os.RemoveAll(tmpDir)

		Logf("Verifying patch by rebuilding source in %s", tmpDir)
		megapatch(patch, source, tmpDir)
	}
}
