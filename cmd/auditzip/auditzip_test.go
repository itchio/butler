package auditzip_test

import (
	"testing"

	"github.com/itchio/butler/cmd/auditzip"
	"github.com/itchio/wharf/state"
	"github.com/itchio/wharf/wtest"

	_ "github.com/itchio/butler/archive/lzmasupport"
)

func TestProtoZip(t *testing.T) {
	consumer := &state.Consumer{
		OnMessage: func(level string, message string) {
			t.Logf("%s %s", level, message)
		},
	}

	upstream := true
	wtest.Must(t, auditzip.Do(consumer, "./testdata/proto.zip", upstream))

	upstream = false
	wtest.Must(t, auditzip.Do(consumer, "./testdata/proto.zip", upstream))
	wtest.Must(t, auditzip.Do(consumer, "./testdata/proto-with-lzma.zip", upstream))
}
