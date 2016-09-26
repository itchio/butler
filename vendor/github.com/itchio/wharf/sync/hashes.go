package sync

import (
	"bufio"
	"bytes"
	"io"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/splitfunc"
)

// CreateSignature calculate the signature of target.
func (ctx *Context) CreateSignature(fileIndex int64, fileReader io.Reader, writeHash SignatureWriter) error {
	s := bufio.NewScanner(fileReader)
	s.Buffer(make([]byte, ctx.blockSize), 0)
	s.Split(splitfunc.New(ctx.blockSize))

	blockIndex := int64(0)

	hashBlock := func(block []byte) error {
		weakHash, _, _ := βhash(block)
		strongHash := ctx.uniqueHash(block)

		blockHash := BlockHash{
			FileIndex:  fileIndex,
			BlockIndex: blockIndex,
			WeakHash:   weakHash,
			StrongHash: strongHash,
		}

		if len(block) < ctx.blockSize {
			blockHash.ShortSize = int32(len(block))
		}

		err := writeHash(blockHash)
		if err != nil {
			return errors.Wrap(err, 1)
		}
		blockIndex++
		return nil
	}

	for s.Scan() {
		err := hashBlock(s.Bytes())
		if err != nil {
			return errors.Wrap(err, 1)
		}
	}

	err := s.Err()
	if err != nil {
		return errors.Wrap(err, 1)
	}

	// let empty files have a 0-length shortblock
	if blockIndex == 0 {
		err := hashBlock([]byte{})
		if err != nil {
			return errors.Wrap(err, 1)
		}
	}

	return nil
}

func (ctx *Context) HashBlock(block []byte) (weakHash uint32, strongHash []byte) {
	weakHash, _, _ = βhash(block)
	strongHash = ctx.uniqueHash(block)
	return
}

// Use a more unique way to identify a set of bytes.
func (ctx *Context) uniqueHash(v []byte) []byte {
	ctx.uniqueHasher.Reset()
	_, err := ctx.uniqueHasher.Write(v)
	if err != nil {
		// FIXME: libs shouldn't panic
		panic(err)
	}
	return ctx.uniqueHasher.Sum(nil)
}

// Searches for a given strong hash among all strong hashes in this bucket.
func findUniqueHash(hh []BlockHash, hashValue []byte, shortSize int32, preferredFileIndex int64) *BlockHash {
	if len(hashValue) == 0 {
		return nil
	}

	// try to find block in preferred file first
	// this helps detect files that aren't touched by patches
	if preferredFileIndex != -1 {
		for _, block := range hh {
			if block.FileIndex == preferredFileIndex {
				if block.ShortSize == shortSize && bytes.Equal(block.StrongHash, hashValue) {
					return &block
				}
			}
		}
	}

	for _, block := range hh {
		// full blocks have 0 shortSize
		if block.ShortSize == shortSize && bytes.Equal(block.StrongHash, hashValue) {
			return &block
		}
	}
	return nil
}

// βhash implements the rolling hash when signing an entire block at a time
func βhash(block []byte) (β uint32, β1 uint32, β2 uint32) {
	var a, b uint32
	for i, val := range block {
		a += uint32(val)
		b += (uint32(len(block)-1) - uint32(i) + 1) * uint32(val)
	}
	β = (a % _M) + (_M * (b % _M))
	β1 = a % _M
	β2 = b % _M
	return
}
