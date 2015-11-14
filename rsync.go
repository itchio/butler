package main

import (
	"bytes"
	"fmt"
	"os"

	"bitbucket.org/kardianos/rsync"
)

func testRSync() {
	if len(os.Args) < 4 {
		die("Missing src or dst for dl command")
	}
	src := os.Args[2]
	dst := os.Args[3]

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
	fmt.Printf("signature is %d blocks long\n", len(sig))

	dstReader, err := os.Open(dst)
	if err != nil {
		panic(err)
	}
	defer dstReader.Close()

	opsOut := make(chan rsync.Operation)

	go func() {
		var opCt, blockCt, blockRangeCt, dataCt, bytes int
		defer close(opsOut)
		err := rsDelta.CreateDelta(dstReader, sig, func(op rsync.Operation) error {
			opCt++
			if opCt%100 == 0 {
				fmt.Printf("Range Ops:%5d, Block Ops:%5d, Data Ops: %5d, Data Len: %5dKiB.\n", blockRangeCt, blockCt, dataCt, bytes/1024)
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
			opsOut <- op
			return nil
		}, nil)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Range Ops:%5d, Block Ops:%5d, Data Ops: %5d, Data Len: %5dKiB.\n", blockRangeCt, blockCt, dataCt, bytes/1024)
	}()

	result := new(bytes.Buffer)
	srcReader.Seek(0, os.SEEK_SET)

	err = rs.ApplyDelta(result, srcReader, opsOut, nil)
	if err != nil {
		panic(err)
	}
}
