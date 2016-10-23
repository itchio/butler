package blockpool

import (
	"fmt"
	"testing"
	"time"

	"github.com/alecthomas/assert"
	"github.com/itchio/wharf/tlc"
)

type TestSink struct {
	FailingBlock BlockLocation
}

var _ Sink = (*TestSink)(nil)

func (ts *TestSink) Clone() Sink {
	return ts
}

func (ts *TestSink) Store(location BlockLocation, data []byte) error {
	time.Sleep(10 * time.Millisecond)
	if location.FileIndex == ts.FailingBlock.FileIndex && location.BlockIndex == ts.FailingBlock.BlockIndex {
		return fmt.Errorf("sample fail!")
	}

	return nil
}

func (ts *TestSink) GetContainer() *tlc.Container {
	return nil
}

func Test_FanOut(t *testing.T) {
	t.Logf("Testing fail fast...")

	ts := &TestSink{
		FailingBlock: BlockLocation{
			FileIndex:  2,
			BlockIndex: 2,
		},
	}
	fos, err := NewFanOutSink(ts, 8)
	assert.NoError(t, err)

	fos.Start()

	hadError := false

	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			loc := BlockLocation{
				FileIndex:  int64(i),
				BlockIndex: int64(j),
			}
			sErr := fos.Store(loc, []byte{})
			if sErr != nil {
				hadError = true
			}
		}
	}

	assert.True(t, hadError)

	err = fos.Close()
	assert.NoError(t, err)

	t.Logf("Testing tail errors...")

	fos, err = NewFanOutSink(ts, 8)
	assert.NoError(t, err)

	fos.Start()

	// Store shouldn't err, just queue it...
	err = fos.Store(ts.FailingBlock, []byte{})
	assert.NoError(t, err)

	// but close should catch the error
	err = fos.Close()
	assert.NotNil(t, err)
}
