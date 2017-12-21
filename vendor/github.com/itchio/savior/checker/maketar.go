package checker

import (
	"bytes"
	"testing"

	"github.com/itchio/arkive/tar"

	"github.com/itchio/savior"
)

func MakeTar(t *testing.T, sink *Sink) []byte {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	for _, item := range sink.Items {
		switch item.Entry.Kind {
		case savior.EntryKindDir:
			must(t, tw.WriteHeader(&tar.Header{
				Name:     item.Entry.CanonicalPath,
				Typeflag: tar.TypeDir,
				Mode:     0755,
			}))
		case savior.EntryKindFile:
			must(t, tw.WriteHeader(&tar.Header{
				Name:     item.Entry.CanonicalPath,
				Typeflag: tar.TypeReg,
				Size:     int64(len(item.Data)),
				Mode:     0644,
			}))

			_, err := tw.Write(item.Data)
			must(t, err)
		case savior.EntryKindSymlink:
			must(t, tw.WriteHeader(&tar.Header{
				Name:     item.Entry.CanonicalPath,
				Typeflag: tar.TypeSymlink,
				Mode:     0644,
				Linkname: item.Entry.Linkname,
			}))
		}
	}

	err := tw.Close()
	must(t, err)

	return buf.Bytes()
}
