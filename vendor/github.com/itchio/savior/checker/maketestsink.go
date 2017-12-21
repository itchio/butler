package checker

import (
	"fmt"
	"math/rand"

	"github.com/itchio/savior"
	"github.com/itchio/savior/semirandom"
)

func MakeTestSink() *Sink {
	return MakeTestSinkAdvanced(10)
}

func MakeTestSinkAdvanced(numEntries int) *Sink {
	sink := NewSink()
	rng := rand.New(rand.NewSource(0xf617a899))
	for i := 0; i < numEntries; i++ {
		if rng.Intn(100) < 10 {
			// ok, make a symlink
			name := fmt.Sprintf("symlink-%d", i)
			sink.Items[name] = &Item{
				Entry: &savior.Entry{
					CanonicalPath: name,
					Kind:          savior.EntryKindDir,
					Linkname:      fmt.Sprintf("target-%d", i*2),
				},
			}
		} else if rng.Intn(100) < 20 {
			// ok, make a dir
			name := fmt.Sprintf("dir-%d", i)
			sink.Items[name] = &Item{
				Entry: &savior.Entry{
					CanonicalPath: name,
					Kind:          savior.EntryKindDir,
				},
			}
		} else {
			// ok, make a file
			size := rng.Int63n(4 * 1024 * 1024)
			name := fmt.Sprintf("file-%d", i)
			sink.Items[name] = &Item{
				Entry: &savior.Entry{
					CanonicalPath:    name,
					Kind:             savior.EntryKindFile,
					UncompressedSize: size,
				},
				Data: semirandom.Bytes(size),
			}
		}
	}
	return sink
}
