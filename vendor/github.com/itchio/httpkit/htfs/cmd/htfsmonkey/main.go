package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
)

func main() {
	if len(os.Args) < 2 {
		must(errors.Errorf("Usage: htfsmonkey [random|zip]"))
	}

	suite := os.Args[1]
	switch suite {
	case "random":
		must(doRandom())
	case "zip":
		must(doZip())
	}
}

// -----------

func must(err error) {
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}
}

// -----------

type delayHandler struct {
	realHandler http.Handler
}

func (dh *delayHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	time.Sleep(time.Millisecond * time.Duration(10+rand.Intn(80)))
	dh.realHandler.ServeHTTP(w, req)
}

// -----------

type fakeFileSystem struct {
	fakeData []byte
}

func (ffs *fakeFileSystem) Open(name string) (http.File, error) {
	br := bytes.NewReader(ffs.fakeData)
	ff := &fakeFile{
		Reader: br,
		FS:     ffs,
	}
	return ff, nil
}

type fakeFile struct {
	*bytes.Reader
	FS *fakeFileSystem
}

func (ff *fakeFile) Stat() (os.FileInfo, error) {
	return &fakeStats{fakeFile: ff}, nil
}

func (ff *fakeFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, nil
}

func (ff *fakeFile) Close() error {
	return nil
}

type fakeStats struct {
	fakeFile *fakeFile
}

func (fs *fakeStats) Name() string {
	return "bin.dat"
}

func (fs *fakeStats) IsDir() bool {
	return false
}

func (fs *fakeStats) Size() int64 {
	return int64(len(fs.fakeFile.FS.fakeData))
}

func (fs *fakeStats) Mode() os.FileMode {
	return 0644
}

func (fs *fakeStats) ModTime() time.Time {
	return time.Now()
}

func (fs *fakeStats) Sys() interface{} {
	return nil
}
