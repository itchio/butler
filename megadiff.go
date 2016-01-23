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

const DIFF_MAGIC = uint64(0xFEF5F04A)

func writeRepoInfo(w *io.Writer, info *megafile.RepoInfo) {
  binary.Write(w, binary.LittleEndian, data interface{})
}

func printRepoStats(info *megafile.RepoInfo, path string) {
	totalSize := int64(0)
	for _, f := range info.Files {
		totalSize += f.Size
	}
	Logf("%s in %d files, %d links, %d dirs in %s", humanize.Bytes(uint64(totalSize)), len(info.Files),
		len(info.Symlinks), len(info.Dirs), path)
}

func megadiff(target string, source string) {
	blockSize := 16 * 1024
	patch := "patch.dat"

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

	must(binary.Write(patchWriter, binary.LittleEndian, DIFF_MAGIC))
	writeRepoInfo(patchWriter, targetInfo)
	writeRepoInfo(patchWriter, sourceInfo)

	sourceInfo, err := megafile.Walk(source, blockSize)
	must(err)
	sourceReader := sourceInfo.NewReader(source)
	defer sourceReader.Close()

	printRepoStats(sourceInfo, source)

	bar := pb.StartNew(int(sourceInfo.NumBlocks) * blockSize)
	bar.SetUnits(pb.U_BYTES)
	bar.SetMaxWidth(80)

	onRead := func(count int64) {
		bar.Set64(count)
	}
	sourceReaderCounter := counter.NewReaderCallback(onRead, sourceReader)

	opsWriter := func(op rsync.Operation) error {
		// Logf("Writing operation, type %d, index %d - %d, data has %d bytes", op.Type, op.BlockIndex, op.BlockIndexEnd, len(op.Data))
		must(binary.Write(patchWriter, binary.LittleEndian, byte(op.Type)))
		must(binary.Write(patchWriter, binary.LittleEndian, op.BlockIndex))
		must(binary.Write(patchWriter, binary.LittleEndian, op.BlockIndexEnd))
		must(binary.Write(patchWriter, binary.LittleEndian, op.Data))
		return nil
	}
	rs.CreateDelta(sourceReaderCounter, signature, opsWriter)

	must(gzipWriter.Close())
	must(brotliWriter.Close())
	must(brotliWriter9.Close())

	bar.Finish()

	Logf("Wrote compressed patch to %s. Sizes:", patch)
	Logf(" - %s (raw)", humanize.Bytes(uint64(rawCounter.Count())))
	Logf(" - %s (gzip q1)", humanize.Bytes(uint64(gzipCounter.Count())))
	Logf(" - %s (brotli q1)", humanize.Bytes(uint64(brotliCounter.Count())))
	Logf(" - %s (brotli q9)", humanize.Bytes(uint64(brotliCounter9.Count())))
}
