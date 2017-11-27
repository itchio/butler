package httpfile

import (
	"os"
	"time"
)

// httpFileInfo implements os.FileInfo for httpfiles
type httpFileInfo struct {
	file *HTTPFile
}

var _ os.FileInfo = (*httpFileInfo)(nil)

func (hfi *httpFileInfo) Name() string {
	return hfi.file.name
}

func (hfi *httpFileInfo) Size() int64 {
	return hfi.file.size
}

func (hfi *httpFileInfo) Mode() os.FileMode {
	return os.FileMode(0)
}

func (hfi *httpFileInfo) ModTime() time.Time {
	return time.Now()
}

func (hfi *httpFileInfo) IsDir() bool {
	return false
}

func (hfi *httpFileInfo) Sys() interface{} {
	return nil
}
