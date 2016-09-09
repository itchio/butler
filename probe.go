package main

import (
	"bufio"
	"fmt"
	"io"
	"time"

	"github.com/Datadog/zstd"
	"github.com/dustin/go-humanize"
	"github.com/itchio/butler/comm"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/splitfunc"
	"github.com/itchio/wharf/tlc"
	"golang.org/x/crypto/sha3"
	"gopkg.in/kothar/brotli-go.v0/enc"
)

type hashedBlock struct {
	data      []byte
	blockpath string
}

func probe(target string) {
	must(doProbe(target))
}

func doProbe(target string) error {
	targetContainer, err := tlc.Walk(target, filterPaths)
	if err != nil {
		return err
	}

	bigBlockSize := 4 * 1024 * 1024 // 4MB blocks
	comm.Opf("Compressing %s as %s blocks", targetContainer.Stats(), humanize.Bytes(uint64(bigBlockSize)))

	pool := targetContainer.NewFilePool(target)

	brotliParams := enc.NewBrotliParams()
	brotliParams.SetQuality(*appArgs.compressionQuality)

	processedSize := int64(0)
	totalCompressedSize := int64(0)

	seenBlocks := make(map[string]bool)
	duplicateBlocks := int64(0)

	comm.StartProgress()

	makeCompressedWriter := func(w io.Writer) io.WriteCloser {
		switch *probeArgs.algo {
		case "brotli":
			return enc.NewBrotliWriter(brotliParams, w)
		case "zstd":
			return zstd.NewWriterLevel(w, *appArgs.compressionQuality)
		default:
			panic(fmt.Sprintf("unknown compression algo %s", *probeArgs.algo))
		}
	}

	startTime := time.Now()

	readBlocks := make(chan []byte)
	hashedBlocks := make(chan hashedBlock, 16)
	uniqueBlocks := make(chan hashedBlock, 16)
	compressedBlocks := make(chan int64, 16)

	totalHashedBlocks := 0
	totalCompressedBlocks := 0

	concurrency := 8
	numHashWorkers := concurrency
	numCompressWorkers := concurrency

	hashOk := make(chan bool)
	lookupOk := make(chan bool)
	compressOk := make(chan bool)
	counterOk := make(chan bool)

	errs := make(chan error)

	var totalHashTime time.Duration

	hashWorker := func() {
		shake128 := sha3.NewShake128()
		hbuf := make([]byte, 32)

		for data := range readBlocks {
			startHashTime := time.Now()

			shake128.Reset()
			shake128.Write(data)

			_, err := io.ReadFull(shake128, hbuf)
			if err != nil {
				errs <- err
				return
			}

			totalHashTime += time.Since(startHashTime)

			blockpath := fmt.Sprintf("shake128-32/%d/%x", len(data), hbuf)
			hashedBlocks <- hashedBlock{
				data:      data,
				blockpath: blockpath,
			}
		}

		hashOk <- true
	}

	lookupWorker := func() {
		for hashedBlock := range hashedBlocks {
			totalHashedBlocks++
			comm.Debugf("in lookup, got %s", hashedBlock.blockpath)

			if seenBlocks[hashedBlock.blockpath] {
				duplicateBlocks++
			} else {
				seenBlocks[hashedBlock.blockpath] = true
				uniqueBlocks <- hashedBlock
			}

			processedSize += int64(len(hashedBlock.data))
			comm.Progress(float64(processedSize) / float64(targetContainer.Size))
		}

		lookupOk <- true
		close(uniqueBlocks)
	}

	var totalCompressTime time.Duration

	compressWorker := func() {
		for uniqueBlock := range uniqueBlocks {
			startCompressTime := time.Now()

			cw := counter.NewWriter(nil)
			bw := makeCompressedWriter(cw)
			bw.Write(uniqueBlock.data)
			bw.Close()

			totalCompressTime += time.Since(startCompressTime)

			compressedBlocks <- cw.Count()
		}

		compressOk <- true
	}

	countWorker := func() {
		for compressedBlock := range compressedBlocks {
			totalCompressedSize += compressedBlock
			totalCompressedBlocks++
		}

		counterOk <- true
	}

	for i := 0; i < numHashWorkers; i++ {
		go hashWorker()
	}

	for i := 0; i < numCompressWorkers; i++ {
		go compressWorker()
	}

	go lookupWorker()
	go countWorker()

	for i := 0; i < len(targetContainer.Files); i++ {
		r, err := pool.GetReader(int64(i))
		if err != nil {
			return err
		}

		s := bufio.NewScanner(r)
		s.Buffer(make([]byte, bigBlockSize), 0)
		s.Split(splitfunc.New(bigBlockSize))

		for s.Scan() {
			readBlocks <- append([]byte{}, s.Bytes()...)
		}
	}
	close(readBlocks)

	for i := 0; i < numHashWorkers; i++ {
		<-hashOk
	}
	close(hashedBlocks)

	<-lookupOk

	for i := 0; i < numCompressWorkers; i++ {
		<-compressOk
	}
	close(compressedBlocks)

	<-counterOk

	comm.EndProgress()

	comm.Statf("total hashed blocks: %d, compressed blocks: %d", totalHashedBlocks, totalCompressedBlocks)
	comm.Statf("spent %v hashing, %v compressing", totalHashTime, totalCompressTime)

	perSecond := humanize.Bytes(uint64(float64(targetContainer.Size) / time.Since(startTime).Seconds()))
	comm.Statf("%s => %s (%.3f) via %s blocks (%d duplicates), %s-q%d @ %s/s",
		humanize.Bytes(uint64(targetContainer.Size)),
		humanize.Bytes(uint64(totalCompressedSize)),
		float64(totalCompressedSize)/float64(targetContainer.Size),
		humanize.Bytes(uint64(bigBlockSize)),
		duplicateBlocks,
		*probeArgs.algo,
		*appArgs.compressionQuality,
		perSecond)

	if !*probeArgs.single {
		return nil
	}

	comm.Opf("Now as a single archive... (no concurrency, but no hashing either)")

	startTime = time.Now()
	comm.StartProgress()

	cw := counter.NewWriter(nil)
	bw := makeCompressedWriter(cw)

	offset := int64(0)

	for i := 0; i < len(targetContainer.Files); i++ {
		r, err := pool.GetReader(int64(i))
		if err != nil {
			return err
		}

		cr := counter.NewReaderCallback(func(count int64) {
			comm.Progress(float64(offset+count) / float64(targetContainer.Size))
		}, r)

		_, err = io.Copy(bw, cr)
		if err != nil {
			return err
		}

		offset += targetContainer.Files[i].Size
	}

	bw.Close()
	totalCompressedSize = cw.Count()

	comm.EndProgress()

	perSecond = humanize.Bytes(uint64(float64(targetContainer.Size) / time.Since(startTime).Seconds()))
	comm.Statf("%s => %s (%.3f) as single archive, %s-q%d @ %s/s",
		humanize.Bytes(uint64(targetContainer.Size)),
		humanize.Bytes(uint64(totalCompressedSize)),
		float64(totalCompressedSize)/float64(targetContainer.Size),
		*probeArgs.algo,
		*appArgs.compressionQuality,
		perSecond)

	return nil
}
