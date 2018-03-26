package blockpool

import (
	"bufio"
	"bytes"
	"fmt"

	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/splitfunc"
	"github.com/itchio/wharf/state"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wsync"
	"github.com/pkg/errors"
)

// A ValidatingSink only stores blocks if they match the signature provided
// in Signature
type ValidatingSink struct {
	// required
	Sink      Sink
	Signature *pwr.SignatureInfo

	// optional
	Consumer *state.Consumer

	// internal
	hashGroups map[BlockLocation][]wsync.BlockHash
	blockBuf   []byte
	split      bufio.SplitFunc
	sctx       *wsync.Context
}

var _ Sink = (*ValidatingSink)(nil)

func (vs *ValidatingSink) Store(loc BlockLocation, data []byte) error {
	vs.log("validating %+v (%d bytes)", loc, len(data))

	if vs.hashGroups == nil {
		vs.log("making hash groups")
		err := vs.makeHashGroups()
		if err != nil {
			return errors.WithStack(err)
		}

		vs.blockBuf = make([]byte, pwr.BlockSize)
		vs.split = splitfunc.New(int(pwr.BlockSize))
		vs.sctx = wsync.NewContext(int(pwr.BlockSize))
	}

	hashGroup := vs.hashGroups[loc]

	// see also wsync.CreateSignature
	s := bufio.NewScanner(bytes.NewReader(data))
	s.Buffer(vs.blockBuf, 0)
	s.Split(vs.split)

	hashIndex := 0

	for ; s.Scan(); hashIndex++ {
		smallBlock := s.Bytes()
		vs.log("validating sub #%d of %+v (%d bytes)", hashIndex, loc, len(smallBlock))

		weakHash, strongHash := vs.sctx.HashBlock(smallBlock)
		bh := hashGroup[hashIndex]

		if bh.WeakHash != weakHash {
			err := fmt.Errorf("at %+v, expected weak hash %x, got %x", loc, bh.WeakHash, weakHash)
			return errors.WithStack(err)
		}

		if !bytes.Equal(bh.StrongHash, strongHash) {
			err := fmt.Errorf("at %+v, expected strong hash %x, got %x", loc, bh.StrongHash, strongHash)
			return errors.WithStack(err)
		}
	}

	return vs.Sink.Store(loc, data)
}

func (vs *ValidatingSink) GetContainer() *tlc.Container {
	return vs.Sink.GetContainer()
}

func (vs *ValidatingSink) Clone() Sink {
	return &ValidatingSink{
		Sink:      vs.Sink.Clone(),
		Signature: vs.Signature,
	}
}

func (vs *ValidatingSink) makeHashGroups() error {
	smallBlockSize := int64(pwr.BlockSize)

	pathToFileIndex := make(map[string]int64)
	for fileIndex, f := range vs.GetContainer().Files {
		pathToFileIndex[f.Path] = int64(fileIndex)
	}

	vs.hashGroups = make(map[BlockLocation][]wsync.BlockHash)
	hashIndex := int64(0)

	for _, f := range vs.Signature.Container.Files {
		fileIndex := pathToFileIndex[f.Path]

		if f.Size == 0 {
			// empty files have a 0-length shortblock for historical reasons.
			hashIndex++
			continue
		}

		numBigBlocks := ComputeNumBlocks(f.Size)
		for blockIndex := int64(0); blockIndex < numBigBlocks; blockIndex++ {
			loc := BlockLocation{
				FileIndex:  fileIndex,
				BlockIndex: blockIndex,
			}

			blockSize := ComputeBlockSize(f.Size, blockIndex)
			numSmallBlocks := (blockSize + smallBlockSize - 1) / smallBlockSize

			vs.hashGroups[loc] = vs.Signature.Hashes[hashIndex : hashIndex+numSmallBlocks]
			hashIndex += numSmallBlocks
		}
	}

	return nil
}

func (vs *ValidatingSink) log(format string, args ...interface{}) {
	if vs.Consumer == nil {
		return
	}

	vs.Consumer.Infof(format, args...)
}
