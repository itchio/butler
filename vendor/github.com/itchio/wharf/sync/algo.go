// Package sync computes a list of operations needed to mutate one file
// into another file, re-using as much of the former as possible.
//
// Base on code from: https://bitbucket.org/kardianos/rsync/
// Original rsync algorithm: http://www.samba.org/~tridge/phd_thesis.pdf
//
// The main change in our fork is supporting blocks of sizes less than
// the context's block size (instead of just passing them as OpData),
// and being able to pick from a hash library that can span multiple
// files, and not just the 'old version' of a file (at the same path).
// This allows us to handle renames gracefully (incl. partial rewrites)
//
// Definitions
//   Source: The final content.
//   Target: The content to be made into final content.
//   Signature: The sequence of hashes used to identify the content.
package sync

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
)

// MaxDataOp is the maximum number of 'fresh bytes' that can be contained
// in a single Data operation
const MaxDataOp = (4 * 1024 * 1024)

// NewContext creates a new Context, given a blocksize.
// It uses MD5 as a 'strong hash' (in the sense of an RSync paper,
// and compared to the very weak rolling hash)
func NewContext(BlockSize int) *Context {
	return &Context{
		blockSize:    BlockSize,
		uniqueHasher: md5.New(),
	}
}

// ApplyPatch applies the difference to the target.
func (ctx *Context) ApplyPatch(output io.Writer, pool FilePool, ops chan Operation) error {
	blockSize := int64(ctx.blockSize)

	for op := range ops {
		switch op.Type {
		case OpBlockRange:
			target, err := pool.GetReadSeeker(op.FileIndex)
			if err != nil {
				return err
			}

			_, err = target.Seek(blockSize*op.BlockIndex, os.SEEK_SET)
			if err != nil {
				return err
			}

			_, err = io.CopyN(output, target, blockSize*op.BlockSpan)
			if err != nil {
				// EOF is actually expected, since we want to copy short
				// blocks at the end of files. Other errors aren't expected.
				if err != io.EOF {
					return fmt.Errorf("While copying %d bytes: %s", blockSize*op.BlockSpan, err.Error())
				}
			}
		case OpData:
			_, err := output.Write(op.Data)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// ComputeDiff creates the operation list to mutate the target signature into the source.
// Any data operation from the OperationWriter must have the data copied out
// within the span of the function; the data buffer underlying the operation
// data is reused.
func (ctx *Context) ComputeDiff(source io.Reader, library *BlockLibrary, ops OperationWriter, preferredFileIndex int64) (err error) {
	minBufferSize := (ctx.blockSize * 2) + MaxDataOp
	if len(ctx.buffer) < minBufferSize {
		ctx.buffer = make([]byte, minBufferSize)
	}
	buffer := ctx.buffer

	type section struct {
		tail int
		head int
	}

	var data, sum section
	var n, validTo int
	var αPop, αPush, β, β1, β2 uint32
	var rolling, lastRun bool
	var shortSize int32

	// Store the previous non-data operation for combining.
	var prevOp *Operation

	// Send the last operation if there is one waiting.
	defer func() {
		if prevOp == nil {
			return
		}

		err = ops(*prevOp)
		prevOp = nil
	}()

	// Combine OpBlockRanges together. To achieve this, we store the previous
	// non-data operation and determine if it can be extended.
	enqueue := func(op Operation) (err error) {
		switch op.Type {
		case OpBlockRange:
			if prevOp != nil {
				if prevOp.Type == OpBlockRange && prevOp.FileIndex == op.FileIndex && prevOp.BlockIndex+prevOp.BlockSpan == op.BlockIndex {
					// combine [prevOp][op] into [ prevOp ]
					prevOp.BlockSpan += op.BlockSpan
					return
				}

				err = ops(*prevOp)
				if err != nil {
					return
				}
				// prevOp has been completely sent off, can no longer be combined with anything
				prevOp = nil
			}
			prevOp = &op
		case OpData:
			// Never save a data operation, as it would corrupt the buffer.
			if prevOp != nil {
				err = ops(*prevOp)
				if err != nil {
					return
				}
			}
			err = ops(op)
			if err != nil {
				return
			}
			prevOp = nil
		}
		return
	}

	for !lastRun {
		// Determine if the buffer should be extended.
		if sum.tail+ctx.blockSize > validTo {
			// Determine if the buffer should be wrapped.
			if validTo+ctx.blockSize > len(buffer) {
				// Before wrapping the buffer, send any trailing data off.
				if data.tail < data.head {
					err = enqueue(Operation{Type: OpData, Data: buffer[data.tail:data.head]})
					if err != nil {
						return err
					}
				}
				// Wrap the buffer.
				l := validTo - sum.tail
				copy(buffer[:l], buffer[sum.tail:validTo])

				// Reset indexes.
				validTo = l
				sum.tail = 0
				data.head = 0
				data.tail = 0
			}

			n, err = io.ReadAtLeast(source, buffer[validTo:validTo+ctx.blockSize], ctx.blockSize)
			validTo += n
			if err != nil {
				if err != io.EOF && err != io.ErrUnexpectedEOF {
					return err
				}
				lastRun = true

				shortSize = int32(n)
			}
		}

		// Set the hash sum window head. Must either be a block size
		// or be at the end of the buffer.
		sum.head = min(sum.tail+ctx.blockSize, validTo)

		// Compute the rolling hash.
		if !rolling {
			β, β1, β2 = βhash(buffer[sum.tail:sum.head])
			rolling = true
		} else {
			αPush = uint32(buffer[sum.head-1])
			β1 = (β1 - αPop + αPush) % _M
			β2 = (β2 - uint32(sum.head-sum.tail)*αPop + β1) % _M
			β = β1 + _M*β2
		}

		var blockHash *BlockHash

		// Determine if there is a hash match.
		if hh, ok := library.hashLookup[β]; ok {
			blockHash = findUniqueHash(hh, ctx.uniqueHash(buffer[sum.tail:sum.head]), shortSize, preferredFileIndex)
		}
		// Send data off if there is data available and a hash is found (so the buffer before it
		// must be flushed first), or the data chunk size has reached it's maximum size (for buffer
		// allocation purposes) or to flush the end of the data.
		if data.tail < data.head && (blockHash != nil || data.head-data.tail >= MaxDataOp) {
			err = enqueue(Operation{Type: OpData, Data: buffer[data.tail:data.head]})
			if err != nil {
				return err
			}
			data.tail = data.head
		}

		if blockHash != nil {
			err = enqueue(Operation{Type: OpBlockRange, FileIndex: blockHash.FileIndex, BlockIndex: blockHash.BlockIndex, BlockSpan: 1})
			if err != nil {
				return err
			}
			rolling = false
			sum.tail += ctx.blockSize

			// There is prior knowledge that any available data
			// buffered will have already been sent. Thus we can
			// assume data.head and data.tail are the same.
			// May trigger "data wrap".
			data.head = sum.tail
			data.tail = sum.tail
		} else {
			if lastRun {
				err = enqueue(Operation{Type: OpData, Data: buffer[data.tail:validTo]})
				if err != nil {
					return err
				}
			} else {
				// The following is for the next loop iteration, so don't try to calculate if last.
				if rolling {
					αPop = uint32(buffer[sum.tail])
				}
				sum.tail++

				// May trigger "data wrap".
				data.head = sum.tail
			}
		}
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
