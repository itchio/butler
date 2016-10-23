package wsync

import (
	"bytes"
	"io"
	"math/rand"
	"testing"
)

type RandReader struct {
	rand.Source
}

func (rr RandReader) Read(sink []byte) (int, error) {
	var tail, head int
	buf := make([]byte, 8)
	var r uint64
	for {
		head = min(tail+8, len(sink))
		if tail == head {
			return head, nil
		}

		r = (uint64)(rr.Int63())
		buf[0] = (byte)(r)
		buf[1] = (byte)(r >> 8)
		buf[2] = (byte)(r >> 16)
		buf[3] = (byte)(r >> 24)
		buf[4] = (byte)(r >> 32)
		buf[5] = (byte)(r >> 40)
		buf[6] = (byte)(r >> 48)
		buf[7] = (byte)(r >> 56)

		tail += copy(sink[tail:head], buf)
	}
}

type pair struct {
	Source, Target content
	Description    string
}
type content struct {
	Len   int
	Seed  int64
	Alter int
	Data  []byte
}

func must(t *testing.T, err error) {
	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	}
}

func (c *content) Fill(t *testing.T) {
	c.Data = make([]byte, c.Len)
	src := rand.NewSource(c.Seed)
	_, err := RandReader{src}.Read(c.Data)
	must(t, err)

	if c.Alter > 0 {
		r := rand.New(src)
		for i := 0; i < c.Alter; i++ {
			at := r.Intn(len(c.Data))
			c.Data[at] += byte(r.Int())
		}
	}
}

func Test_GenData(t *testing.T) {
	// Use a seeded generator to get consistent results.
	// This allows testing the package without bundling many test files.

	var pairs = []pair{
		{
			Source:      content{Len: 8*512*1024 + 89, Seed: 42, Alter: 0},
			Target:      content{Len: 8*512*1024 + 89, Seed: 42, Alter: 5},
			Description: "Same length, slightly different content.",
		},
		{
			Source:      content{Len: 8*512*1024 + 89, Seed: 9824, Alter: 0},
			Target:      content{Len: 8*512*1024 + 89, Seed: 2345, Alter: 0},
			Description: "Same length, very different content.",
		},
		{
			Source:      content{Len: 8*512*1024 + 89, Seed: 42, Alter: 0},
			Target:      content{Len: 8*256*1024 + 19, Seed: 42, Alter: 0},
			Description: "Target shorter then source, same content.",
		},
		{
			Source:      content{Len: 8*512*1024 + 89, Seed: 42, Alter: 0},
			Target:      content{Len: 8*256*1024 + 19, Seed: 42, Alter: 5},
			Description: "Target shorter then source, slightly different content.",
		},
		{
			Source:      content{Len: 8*256*1024 + 19, Seed: 42, Alter: 0},
			Target:      content{Len: 8*512*1024 + 89, Seed: 42, Alter: 0},
			Description: "Source shorter then target, same content.",
		},
		{
			Source:      content{Len: 8*512*1024 + 89, Seed: 42, Alter: 5},
			Target:      content{Len: 8*256*1024 + 19, Seed: 42, Alter: 0},
			Description: "Source shorter then target, slightly different content.",
		},
		{
			Source:      content{Len: 8*512*1024 + 89, Seed: 42, Alter: 0},
			Target:      content{Len: 0, Seed: 42, Alter: 0},
			Description: "Target empty and source has content.",
		},
		{
			Source:      content{Len: 0, Seed: 42, Alter: 0},
			Target:      content{Len: 8*512*1024 + 89, Seed: 42, Alter: 0},
			Description: "Source empty and target has content.",
		},
		{
			Source:      content{Len: 8 * 872, Seed: 9824, Alter: 0},
			Target:      content{Len: 8 * 235, Seed: 2345, Alter: 0},
			Description: "Source and target both smaller then a block size.",
		},
	}
	rs := NewContext(16 * 1024)
	rsDelta := NewContext(16 * 1024)
	for _, p := range pairs {
		(&p.Source).Fill(t)
		(&p.Target).Fill(t)

		sourceBuffer := bytes.NewReader(p.Source.Data)
		targetBuffer := bytes.NewReader(p.Target.Data)

		sig := make([]BlockHash, 0, 10)
		err := rs.CreateSignature(0, targetBuffer, func(bl BlockHash) error {
			sig = append(sig, bl)
			return nil
		})
		if err != nil {
			t.Errorf("Failed to create signature: %s", err)
		}
		lib := NewBlockLibrary(sig)

		opsOut := make(chan Operation)
		go func() {
			var blockRangeCt, dataCt, bytes int
			defer close(opsOut)
			deltaErr := rsDelta.ComputeDiff(sourceBuffer, lib, func(op Operation) error {
				switch op.Type {
				case OpBlockRange:
					blockRangeCt++
				case OpData:
					// Copy data buffer so it may be reused in internal buffer.
					b := make([]byte, len(op.Data))
					copy(b, op.Data)
					op.Data = b
					dataCt++
					bytes += len(op.Data)
				}
				opsOut <- op
				return nil
			}, -1)
			t.Logf("Range Ops:%5d, Data Ops: %5d, Data Len: %5dKiB, For %s.", blockRangeCt, dataCt, bytes/1024, p.Description)
			if deltaErr != nil {
				t.Errorf("Failed to create delta: %s", deltaErr)
			}
		}()

		result := new(bytes.Buffer)
		pool := &SinglePool{reader: targetBuffer}

		_, err = targetBuffer.Seek(0, 0)
		must(t, err)
		err = rs.ApplyPatch(result, pool, opsOut)
		if err != nil {
			t.Errorf("Failed to apply delta: %s", err)
		}

		if result.Len() != len(p.Source.Data) {
			t.Errorf("Result not same size as source: %s", p.Description)
		} else if bytes.Equal(result.Bytes(), p.Source.Data) == false {
			t.Errorf("Result is different from the source: %s", p.Description)
		}

		p.Source.Data = nil
		p.Target.Data = nil
	}
}

type SinglePool struct {
	reader io.ReadSeeker
}

var _ Pool = (*SinglePool)(nil)

func (sp *SinglePool) GetReader(fileIndex int64) (io.Reader, error) {
	return sp.GetReadSeeker(fileIndex)
}

func (sp *SinglePool) GetReadSeeker(fileIndex int64) (io.ReadSeeker, error) {
	return sp.reader, nil
}

func (sp *SinglePool) Close() error {
	return nil
}
