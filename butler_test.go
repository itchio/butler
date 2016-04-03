package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"testing"

	"github.com/dustin/go-humanize"
	"github.com/itchio/butler/comm"
)

// reverse must
func mist(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

func putfile(t *testing.T, basePath string, i int, data []byte) {
	perm := os.FileMode(0777)
	samplePath := path.Join(basePath, fmt.Sprintf("dummy%d.dat", i))
	mist(t, ioutil.WriteFile(samplePath, data, perm))
}

func TestAllTheThings(t *testing.T) {
	perm := os.FileMode(0777)
	workingDir, err := ioutil.TempDir("", "butler-tests")
	mist(t, err)
	defer os.RemoveAll(workingDir)

	sample := path.Join(workingDir, "sample")
	mist(t, os.MkdirAll(sample, perm))
	mist(t, ioutil.WriteFile(path.Join(sample, "hello.txt"), []byte("hello!"), perm))

	sample2 := path.Join(workingDir, "sample2")
	mist(t, os.MkdirAll(sample2, perm))
	for i := 0; i < 80; i++ {
		putfile(t, sample2, i, bytes.Repeat([]byte{0x42, 0x69}, i*200+1))
	}

	sample3 := path.Join(workingDir, "sample3")
	mist(t, os.MkdirAll(sample3, perm))
	for i := 0; i < 60; i++ {
		putfile(t, sample3, i, bytes.Repeat([]byte{0x42, 0x69}, i*300+1))
	}

	sample4 := path.Join(workingDir, "sample4")
	mist(t, os.MkdirAll(sample4, perm))
	for i := 0; i < 120; i++ {
		putfile(t, sample4, i, bytes.Repeat([]byte{0x42, 0x69}, i*150+1))
	}

	sample5 := path.Join(workingDir, "sample5")
	mist(t, os.MkdirAll(sample5, perm))
	rg := rand.New(rand.NewSource(0x239487))

	for i := 0; i < 25; i++ {
		l := 1024 * (i + 2)
		buf := make([]byte, l)
		_, err := io.CopyN(bytes.NewBuffer(buf), rg, int64(l))
		mist(t, err)
		putfile(t, sample5, i, buf)
	}

	files := map[string]string{
		"hello":     sample,
		"80-fixed":  sample2,
		"60-fixed":  sample3,
		"120-fixed": sample4,
		"random":    sample5,
		"null":      "/dev/null",
	}

	patch := path.Join(workingDir, "patch.pwr")

	comm.Configure(true, true, false, false, false)

	for _, q := range []int{1, 9} {
		t.Logf("============ Quality %d ============", q)
		for lhs := range files {
			for rhs := range files {
				mist(t, doDiff(files[lhs], files[rhs], patch, q))
				stat, err := os.Lstat(patch)
				mist(t, err)
				t.Logf("%10s -> %10s = %s", lhs, rhs, humanize.Bytes(uint64(stat.Size())))
			}
		}
	}
}
