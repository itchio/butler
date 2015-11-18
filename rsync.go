package main

import (
	"bytes"
	"encoding/gob"
	"io"
	"log"
	"os"

	"github.com/dustin/go-humanize"

	"gopkg.in/itchio/rsync-go.v0"
	"gopkg.in/kothar/brotli-go.v0/enc"
)

func testRSync() {
	if len(os.Args) < 4 {
		die("Missing src or dst for dl command")
	}
	src := os.Args[2]
	dst := os.Args[3]

	stats, err := os.Lstat(src)
	if err != nil {
		panic(err)
	}
	log.Printf("src is %d bytes (%s)", stats.Size(), humanize.Bytes(uint64(stats.Size())))

	stats, err = os.Lstat(dst)
	if err != nil {
		panic(err)
	}
	log.Printf("dst is %d bytes (%s)", stats.Size(), humanize.Bytes(uint64(stats.Size())))

	srcReader, err := os.Open(src)
	if err != nil {
		panic(err)
	}
	defer srcReader.Close()

	rs := &rsync.RSync{}
	rsDelta := &rsync.RSync{}
	sig := make([]rsync.BlockHash, 0, 10)
	err = rs.CreateSignature(srcReader, func(bl rsync.BlockHash) error {
		sig = append(sig, bl)
		return nil
	})
	log.Printf("signature is %d blocks long\n", len(sig))

	dstReader, err := os.Open(dst)
	if err != nil {
		panic(err)
	}
	defer dstReader.Close()

	opsOut := make(chan rsync.Operation)

	qualities := []int{1, 4, 6, 9}
	compressedWriters := make([]io.Writer, len(qualities))
	compressedCounters := make([]*counterWriter, len(qualities))

	for i, q := range qualities {
		params := enc.NewBrotliParams()
		params.SetQuality(q)
		counter := &counterWriter{}
		writer := enc.NewBrotliWriter(params, counter)

		compressedCounters[i] = counter
		compressedWriters[i] = writer
	}

	compressedWriter := io.MultiWriter(compressedWriters...)

	uncompressedCounter := &counterWriter{writer: compressedWriter}
	marshal := gob.NewEncoder(uncompressedCounter)

	go func() {
		var opCt, blockCt, blockRangeCt, dataCt, bytes int
		defer close(opsOut)
		err := rsDelta.CreateDelta(dstReader, sig, func(op rsync.Operation) error {
			opCt++
			if opCt%100 == 0 {
				log.Printf("Range Ops:%5d, Block Ops:%5d, Data Ops: %5d, Data Len: %5dKiB.\n", blockRangeCt, blockCt, dataCt, bytes/1024)
			}

			switch op.Type {
			case rsync.OpBlockRange:
				blockRangeCt++
			case rsync.OpBlock:
				blockCt++
			case rsync.OpData:
				// Copy data buffer so it may be reused in internal buffer.
				b := make([]byte, len(op.Data))
				copy(b, op.Data)
				op.Data = b
				dataCt++
				bytes += len(op.Data)
			}
			err := marshal.Encode(&op)
			if err != nil {
				log.Println("err in marshal.Encode")
				return err
			}

			opsOut <- op
			return nil
		}, nil)
		if err != nil {
			panic(err)
		}
		log.Printf("Range Ops:%5d, Block Ops:%5d, Data Ops: %5d, Data Len: %5dKiB.\n", blockRangeCt, blockCt, dataCt, bytes/1024)
	}()

	result := new(bytes.Buffer)
	srcReader.Seek(0, os.SEEK_SET)

	err = rs.ApplyDelta(result, srcReader, opsOut)
	if err != nil {
		panic(err)
	}

	log.Printf("%d bytes of raw diff (%s, %.2f%% of dst)", uncompressedCounter.count, humanize.Bytes(uint64(uncompressedCounter.count)),
		float32(uncompressedCounter.count)/float32(stats.Size())*100)

	for i, q := range qualities {
		counter := compressedCounters[i]
		writer := compressedWriters[i]
		if v, ok := writer.(io.Closer); ok {
			v.Close()
		}

		log.Printf("%d bytes of compressed diff (brotli at q=%d) (%s, %.2f%% of raw diff, %.2f%% of dst)", counter.count, q, humanize.Bytes(uint64(counter.count)),
			float32(counter.count)/float32(uncompressedCounter.count)*100.0,
			float32(counter.count)/float32(stats.Size())*100.0)
	}
}
