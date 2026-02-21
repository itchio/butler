package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"runtime"
	"strconv"
	"testing"

	"github.com/itchio/headway/united"

	"github.com/itchio/butler/cmd/apply"
	"github.com/itchio/butler/cmd/diff"
	"github.com/itchio/butler/cmd/ditto"
	"github.com/itchio/butler/cmd/sign"
	"github.com/itchio/butler/comm"
	"github.com/itchio/httpkit/eos"
	"github.com/itchio/savior/seeksource"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/wtest"
	"github.com/stretchr/testify/assert"
)

func octal(perm os.FileMode) string {
	return strconv.FormatInt(int64(perm), 8)
}

func permFor(t *testing.T, path string) os.FileMode {
	t.Logf("Getting perm of %s", path)
	stat, err := os.Lstat(path)
	wtest.Must(t, err)
	return stat.Mode()
}

func putfile(t *testing.T, basePath string, i int, data []byte) {
	putfileEx(t, basePath, i, data, os.FileMode(0o777))
}

func putfileEx(t *testing.T, basePath string, i int, data []byte, perm os.FileMode) {
	samplePath := path.Join(basePath, fmt.Sprintf("dummy%d.dat", i))
	wtest.Must(t, ioutil.WriteFile(samplePath, data, perm))
}

func TestAllTheThings(t *testing.T) {
	perm := os.FileMode(0o777)
	workingDir, err := ioutil.TempDir("", "butler-tests")
	wtest.Must(t, err)
	defer os.RemoveAll(workingDir)

	sample := path.Join(workingDir, "sample")
	wtest.Must(t, os.MkdirAll(sample, perm))
	wtest.Must(t, ioutil.WriteFile(path.Join(sample, "hello.txt"), []byte("hello!"), perm))

	sample2 := path.Join(workingDir, "sample2")
	wtest.Must(t, os.MkdirAll(sample2, perm))
	for i := 0; i < 5; i++ {
		if i == 3 {
			// e.g. .gitkeep
			putfile(t, sample2, i, []byte{})
		} else {
			putfile(t, sample2, i, bytes.Repeat([]byte{0x42, 0x69}, i*200+1))
		}
	}

	sample3 := path.Join(workingDir, "sample3")
	wtest.Must(t, os.MkdirAll(sample3, perm))
	for i := 0; i < 60; i++ {
		putfile(t, sample3, i, bytes.Repeat([]byte{0x42, 0x69}, i*300+1))
	}

	sample4 := path.Join(workingDir, "sample4")
	wtest.Must(t, os.MkdirAll(sample4, perm))
	for i := 0; i < 120; i++ {
		putfile(t, sample4, i, bytes.Repeat([]byte{0x42, 0x69}, i*150+1))
	}

	sample5 := path.Join(workingDir, "sample5")
	wtest.Must(t, os.MkdirAll(sample5, perm))
	rg := rand.New(rand.NewSource(0x239487))

	for i := 0; i < 25; i++ {
		l := 1024 * (i + 2)
		// our own little twist on fizzbuzz to look out for 1-off errors
		if i%5 == 0 {
			l = int(pwr.BlockSize)
		} else if i%3 == 0 {
			l = 0
		}

		buf := make([]byte, l)
		_, err := io.CopyN(bytes.NewBuffer(buf), rg, int64(l))
		wtest.Must(t, err)
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

	for _, q := range []int{1, 9} {
		t.Logf("============ Quality %d ============", q)
		compression := &pwr.CompressionSettings{
			Algorithm: pwr.CompressionAlgorithm_BROTLI,
			Quality:   int32(q),
		}

		for lhs := range files {
			for rhs := range files {
				wtest.Must(t, diff.Do(&diff.Params{
					Target:      files[lhs],
					Source:      files[rhs],
					Patch:       patch,
					Compression: compression,
				}))
				stat, err := os.Lstat(patch)
				wtest.Must(t, err)
				t.Logf("%10s -> %10s = %s", lhs, rhs, united.FormatBytes(stat.Size()))
			}
		}
	}

	compression := &pwr.CompressionSettings{
		Algorithm: pwr.CompressionAlgorithm_BROTLI,
		Quality:   1,
	}

	for _, filepath := range files {
		t.Logf("Signing %s\n", filepath)

		sigPath := path.Join(workingDir, "signature.pwr.sig")
		wtest.Must(t, sign.Do(filepath, sigPath, compression, false))

		sigReader, err := eos.Open(sigPath)
		wtest.Must(t, err)

		sigSource := seeksource.FromFile(sigReader)
		_, err = sigSource.Resume(nil)
		wtest.Must(t, err)

		signature, err := pwr.ReadSignature(context.Background(), sigSource)
		wtest.Must(t, err)

		wtest.Must(t, sigReader.Close())

		validator := &pwr.ValidatorContext{
			FailFast: true,
		}

		wtest.Must(t, validator.Validate(context.Background(), filepath, signature))
	}

	// K windows you just sit this one out we'll catch you on the flip side
	if runtime.GOOS != "windows" {
		// In-place preserve permissions tests
		t.Logf("In-place patching should preserve permissions")

		eperm := os.FileMode(0o750)

		samplePerm1 := path.Join(workingDir, "samplePerm1")
		wtest.Must(t, os.MkdirAll(samplePerm1, perm))
		putfileEx(t, samplePerm1, 1, bytes.Repeat([]byte{0x42, 0x69}, 8192), eperm)

		assert.Equal(t, octal(eperm), octal(permFor(t, path.Join(samplePerm1, "dummy1.dat"))))

		samplePerm2 := path.Join(workingDir, "samplePerm2")
		wtest.Must(t, os.MkdirAll(samplePerm2, perm))
		putfileEx(t, samplePerm2, 1, bytes.Repeat([]byte{0x69, 0x42}, 16384), eperm)

		assert.Equal(t, octal(eperm), octal(permFor(t, path.Join(samplePerm2, "dummy1.dat"))))

		wtest.Must(t, diff.Do(&diff.Params{
			Target:      samplePerm1,
			Source:      samplePerm2,
			Patch:       patch,
			Compression: compression,
		}))
		_, err := os.Lstat(patch)
		wtest.Must(t, err)

		cave := path.Join(workingDir, "cave")
		ditto.Do(ditto.Params{Src: samplePerm1, Dst: cave, PreservePermissions: true})

		assert.Equal(t, octal(eperm), octal(permFor(t, path.Join(cave, "dummy1.dat"))))

		staging := path.Join(workingDir, "staging")

		wtest.Must(t, apply.Do(apply.Params{
			Patch:      patch,
			Old:        cave,
			StagingDir: staging,
		}))
		assert.Equal(t, octal(eperm), octal(permFor(t, path.Join(cave, "dummy1.dat"))))
	}
}
