package eos

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alecthomas/assert"

	"github.com/itchio/httpkit/httpfile"
)

func Test_OpenEmptyFile(t *testing.T) {
	f, err := Open("/dev/null")
	assert.NoError(t, err)

	s, err := f.Stat()
	assert.NoError(t, err)

	assert.EqualValues(t, 0, s.Size())
	assert.False(t, s.IsDir())
	assert.EqualValues(t, "/dev/null", s.Name())
	assert.Nil(t, s.Sys())
	assert.EqualValues(t, 0644, s.Mode())
	assert.EqualValues(t, time.Time{}, s.ModTime())

	buf := make([]byte, 1)

	_, err = f.Read(buf)
	assert.Error(t, err)

	_, err = f.ReadAt(buf, 0)
	assert.Error(t, err)

	assert.NoError(t, f.Close())
}

func Test_OpenLocalFile(t *testing.T) {
	mainDir, err := ioutil.TempDir("", "eos-local")
	assert.NoError(t, err)
	defer os.RemoveAll(mainDir)

	assert.NoError(t, os.MkdirAll(mainDir, 0755))

	fileName := filepath.Join(mainDir, "some-file")
	assert.NoError(t, ioutil.WriteFile(fileName, []byte{4, 2, 6, 9}, 0644))

	f, err := Open(fileName)
	assert.NoError(t, err)

	s, err := f.Stat()
	assert.NoError(t, err)

	assert.EqualValues(t, 4, s.Size())
	assert.NoError(t, f.Close())
}

type testfs struct {
	url string
}

func (tfs *testfs) Scheme() string {
	return "testfs"
}

func (tfs *testfs) MakeResource(u *url.URL) (httpfile.GetURLFunc, httpfile.NeedsRenewalFunc, error) {
	return tfs.GetURL, tfs.NeedsRenewal, nil
}

func (tfs *testfs) GetURL() (string, error) {
	return tfs.url, nil
}

func (tfs *testfs) NeedsRenewal(res *http.Response, body []byte) bool {
	return false
}

func Test_OpenRemoteDownloadBuild(t *testing.T) {
	fakeData := []byte("aaaabbbb")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-length", fmt.Sprintf("%d", len(fakeData)))
		w.WriteHeader(200)
		w.Write(fakeData)
	}))
	defer server.CloseClientConnections()

	tfs := &testfs{server.URL}
	assert.NoError(t, RegisterHandler(tfs))
	assert.Error(t, RegisterHandler(tfs))
	defer DeregisterHandler(tfs)

	f, err := Open("nofs:///not-quite")
	assert.Error(t, err)

	f, err = Open("testfs:///now/we/are/talking")
	assert.NoError(t, err)

	stats, err := f.Stat()
	assert.NoError(t, err)

	assert.EqualValues(t, len(fakeData), stats.Size())
	assert.NoError(t, f.Close())
}

func Test_HttpFile(t *testing.T) {
	fakeData := []byte("aaaabbbb")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-length", fmt.Sprintf("%d", len(fakeData)))
		w.WriteHeader(200)
		w.Write(fakeData)
	}))
	defer server.CloseClientConnections()

	f, err := Open(server.URL)
	assert.NoError(t, err)

	s, err := f.Stat()
	assert.NoError(t, err)
	assert.EqualValues(t, len(fakeData), s.Size())

	readData, err := ioutil.ReadAll(f)
	assert.NoError(t, err)
	assert.EqualValues(t, fakeData, readData)

	assert.NoError(t, f.Close())
}
