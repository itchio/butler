package pwr

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert"
	"github.com/itchio/wharf/tlc"
)

func Test_NewHealer(t *testing.T) {
	_, err := NewHealer("", "/dev/null")
	assert.Error(t, err)

	_, err = NewHealer("nope,/dev/null", "invalid")
	assert.Error(t, err)

	healer, err := NewHealer("archive,/dev/null", "invalid")
	assert.NoError(t, err)

	_, ok := healer.(*ArchiveHealer)
	assert.True(t, ok)
}

func Test_ArchiveHealer(t *testing.T) {
	mainDir, err := ioutil.TempDir("", "archivehealer")
	assert.NoError(t, err)
	defer os.RemoveAll(mainDir)

	archivePath := filepath.Join(mainDir, "archive.zip")
	archiveWriter, err := os.Create(archivePath)
	assert.NoError(t, err)
	defer archiveWriter.Close()

	targetDir := filepath.Join(mainDir, "target")
	assert.NoError(t, os.MkdirAll(targetDir, 0755))

	zw := zip.NewWriter(archiveWriter)
	numFiles := 16
	fakeData := []byte{1, 2, 3, 4}

	nameFor := func(index int) string {
		return fmt.Sprintf("file-%d", index)
	}

	pathFor := func(index int) string {
		return filepath.Join(targetDir, nameFor(index))
	}

	for i := 0; i < numFiles; i++ {
		writer, cErr := zw.Create(nameFor(i))
		assert.NoError(t, cErr)

		_, cErr = writer.Write(fakeData)
		assert.NoError(t, cErr)
	}

	assert.NoError(t, zw.Close())

	container, err := tlc.WalkAny(archivePath, nil)
	assert.NoError(t, err)

	healAll := func() Healer {
		healer, err := NewHealer(fmt.Sprintf("archive,%s", archivePath), targetDir)
		assert.NoError(t, err)

		wounds := make(chan *Wound)
		done := make(chan bool)

		go func() {
			err := healer.Do(container, wounds)
			assert.NoError(t, err)
			done <- true
		}()

		for i := 0; i < numFiles; i++ {
			wounds <- &Wound{
				Kind:  WoundKind_FILE,
				Index: int64(i),
				Start: 0,
				End:   1,
			}
		}

		close(wounds)

		<-done

		return healer
	}

	assertAllFilesHealed := func() {
		for i := 0; i < numFiles; i++ {
			data, err := ioutil.ReadFile(pathFor(i))
			assert.NoError(t, err)

			assert.Equal(t, fakeData, data)
		}
	}

	t.Logf("...with no files present")
	healer := healAll()
	assert.Equal(t, int64(numFiles), healer.TotalCorrupted())
	assert.Equal(t, int64(numFiles*len(fakeData)), healer.TotalHealed())
	assertAllFilesHealed()

	t.Logf("...with one file too long")
	assert.NoError(t, ioutil.WriteFile(pathFor(3), bytes.Repeat(fakeData, 4), 0644))
	healer = healAll()
	assert.Equal(t, int64(numFiles), healer.TotalCorrupted())
	assert.Equal(t, int64(numFiles*len(fakeData)), healer.TotalHealed())
	assertAllFilesHealed()

	t.Logf("...with one file too short")
	assert.NoError(t, ioutil.WriteFile(pathFor(7), fakeData[:1], 0644))
	healer = healAll()
	assert.Equal(t, int64(numFiles), healer.TotalCorrupted())
	assert.Equal(t, int64(numFiles*len(fakeData)), healer.TotalHealed())
	assertAllFilesHealed()

	t.Logf("...with one file slightly corrupted")
	corruptedFakeData := append([]byte{}, fakeData...)
	corruptedFakeData[2] = 255
	assert.NoError(t, ioutil.WriteFile(pathFor(9), corruptedFakeData, 0644))
	healer = healAll()
	assert.Equal(t, int64(numFiles), healer.TotalCorrupted())
	assert.Equal(t, int64(numFiles*len(fakeData)), healer.TotalHealed())
	assertAllFilesHealed()
}
