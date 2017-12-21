package checker

import (
	"bytes"
	"log"
	"os"
	"testing"

	"github.com/itchio/arkive/zip"

	"github.com/itchio/savior"
)

func MakeZip(t *testing.T, sink *Sink) []byte {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	shouldCompress := true
	numDeflate := 0
	numStore := 0

	for _, item := range sink.Items {
		fh := &zip.FileHeader{
			Name: item.Entry.CanonicalPath,
		}

		switch item.Entry.Kind {
		case savior.EntryKindDir:
			fh.SetMode(os.ModeDir | 0755)
			_, err := zw.CreateHeader(fh)
			must(t, err)
		case savior.EntryKindFile:
			fh.SetMode(0644)
			if shouldCompress {
				fh.Method = zip.Deflate
				numDeflate++
			} else {
				fh.Method = zip.Store
				numStore++
			}
			shouldCompress = !shouldCompress
			writer, err := zw.CreateHeader(fh)
			must(t, err)

			_, err = writer.Write(item.Data)
			must(t, err)
		case savior.EntryKindSymlink:
			fh.SetMode(os.ModeSymlink | 0644)
			writer, err := zw.CreateHeader(fh)
			must(t, err)

			_, err = writer.Write([]byte(item.Entry.Linkname))
			must(t, err)
		}
	}

	err := zw.Close()
	must(t, err)

	log.Printf("Made zip with %d deflate files, %d store files", numDeflate, numStore)

	return buf.Bytes()
}
