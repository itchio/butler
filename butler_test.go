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
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/sync"
	"github.com/itchio/wharf/tlc"
	"github.com/stretchr/testify/assert"
)

// reverse must
func mist(t *testing.T, err error) {
	if err != nil {
		panic(err)
	}
}

func putfile(t *testing.T, basePath string, i int, data []byte) {
	perm := os.FileMode(0777)
	samplePath := path.Join(basePath, fmt.Sprintf("dummy%d.dat", i))
	mist(t, ioutil.WriteFile(samplePath, data, perm))
}

func shortSizeCount(hashes []sync.BlockHash) string {
	count := 0
	for _, hash := range hashes {
		if hash.ShortSize != 0 {
			count++
		}
	}
	return fmt.Sprintf("%d/%d", count, len(hashes))
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
	for i := 0; i < 5; i++ {
		if i == 3 {
			// e.g. .gitkeep
			putfile(t, sample2, i, []byte{})
		} else {
			putfile(t, sample2, i, bytes.Repeat([]byte{0x42, 0x69}, i*200+1))
		}
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
		// our own little twist on fizzbuzz to look out for 1-off errors
		if i%5 == 0 {
			l = pwr.BlockSize
		} else if i%3 == 0 {
			l = 0
		}

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

	comm.Configure(true, true, false, false, false, false, false)

	if false {
		for _, q := range []int{1, 9} {
			t.Logf("============ Quality %d ============", q)
			compression := pwr.CompressionSettings{
				Algorithm: pwr.CompressionAlgorithm_BROTLI,
				Quality:   int32(q),
			}

			for lhs := range files {
				for rhs := range files {
					mist(t, doDiff(files[lhs], files[rhs], patch, compression))
					stat, err := os.Lstat(patch)
					mist(t, err)
					t.Logf("%10s -> %10s = %s", lhs, rhs, humanize.Bytes(uint64(stat.Size())))
				}
			}
		}
	}

	compression := pwr.CompressionSettings{
		Algorithm: pwr.CompressionAlgorithm_BROTLI,
		Quality:   1,
	}

	for _, filepath := range files {
		t.Logf("Signing %s\n", filepath)

		sigpath := path.Join(workingDir, "signature.pwr.sig")
		mist(t, doSign(filepath, sigpath, compression, false))

		sigr, err := os.Open(sigpath)
		mist(t, err)

		readcontainer, readsig, err := pwr.ReadSignature(sigr)
		mist(t, err)

		mist(t, sigr.Close())

		computedcontainer, err := tlc.Walk(filepath, filterPaths)
		mist(t, err)

		computedsig, err := pwr.ComputeSignature(computedcontainer, computedcontainer.NewFilePool(filepath), &pwr.StateConsumer{})
		mist(t, err)

		assert.Equal(t, len(readcontainer.Files), len(computedcontainer.Files))
		for i, rf := range readcontainer.Files {
			cf := computedcontainer.Files[i]
			assert.Equal(t, *rf, *cf)
		}

		assert.Equal(t, readsig, computedsig)
		mist(t, pwr.CompareHashes(readsig, computedsig, computedcontainer))
	}
}
