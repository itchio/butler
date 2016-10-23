package archiver

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/alecthomas/assert"
	"github.com/itchio/wharf/state"
)

var testSymlinks bool = (runtime.GOOS != "windows")

func makeTestDir(t *testing.T, dir string) {
	assert.NoError(t, os.MkdirAll(dir, 0755))

	assert.NoError(t, os.MkdirAll(filepath.Join(dir, "subdir"), 0755))

	createFile := func(name string) {
		f, fErr := os.Create(filepath.Join(dir, name))
		assert.NoError(t, fErr)
		defer f.Close()

		_, fErr = f.Write([]byte{4, 3, 2, 1})
		assert.NoError(t, fErr)
	}

	createLink := func(name string, dest string) {
		if !testSymlinks {
			return
		}
		assert.NoError(t, os.Symlink(filepath.Join(dir, dest), filepath.Join(dir, name)))
	}

	for i := 0; i < 4; i++ {
		createFile(fmt.Sprintf("file-%d", i))
	}

	assert.NoError(t, os.MkdirAll(filepath.Join(dir, "subdir"), 0755))

	for i := 0; i < 2; i++ {
		createFile(fmt.Sprintf("subdir/file-%d", i))
	}

	createLink("link1", "subdir/file-1")
	createLink("link2", "file-3")
}

func Test_ZipUnzip(t *testing.T) {
	tmpPath, err := ioutil.TempDir("", "zipunzip")
	assert.NoError(t, err)

	defer os.RemoveAll(tmpPath)
	assert.NoError(t, os.MkdirAll(tmpPath, 0755))

	dir := filepath.Join(tmpPath, "dir")
	makeTestDir(t, dir)

	extractedDir := filepath.Join(tmpPath, "extractedDir")
	archivePath := filepath.Join(tmpPath, "archive.zip")

	archiveWriter, err := os.Create(archivePath)
	assert.NoError(t, err)
	defer archiveWriter.Close()

	_, err = CompressZip(archiveWriter, dir, &state.Consumer{})
	assert.NoError(t, err)

	xSettings := ExtractSettings{
		Consumer: &state.Consumer{},
	}

	t.Logf("Extracting over non-existent destination")
	_, err = ExtractPath(archivePath, extractedDir, xSettings)
	assert.NoError(t, err)

	resumeFilePath := filepath.Join(tmpPath, "resumeinfo")

	t.Logf("Extracting over already-extracted dir")
	_, err = ExtractPath(archivePath, extractedDir, ExtractSettings{
		Consumer:   xSettings.Consumer,
		ResumeFrom: resumeFilePath,
	})
	assert.NoError(t, err)

	t.Logf("Extracting over already-extracted dir (with resume)")
	_, err = ExtractPath(archivePath, extractedDir, ExtractSettings{
		Consumer:   xSettings.Consumer,
		ResumeFrom: resumeFilePath,
	})
	assert.NoError(t, err)

	t.Logf("Extracting, one of the dirs is a file now")
	assert.NoError(t, os.RemoveAll(filepath.Join(extractedDir, "subdir")))
	dumbFile, err := os.Create(filepath.Join(extractedDir, "subdir"))
	assert.NoError(t, err)
	assert.NoError(t, dumbFile.Close())

	_, err = ExtractPath(archivePath, extractedDir, xSettings)
	assert.NoError(t, err)
}

func Test_TarUntar(t *testing.T) {
	tmpPath, err := ioutil.TempDir("", "taruntar")
	assert.NoError(t, err)

	defer os.RemoveAll(tmpPath)
	assert.NoError(t, os.MkdirAll(tmpPath, 0755))

	dir := filepath.Join(tmpPath, "dir")
	makeTestDir(t, dir)

	extractedDir := filepath.Join(tmpPath, "extractedDir")
	archivePath := filepath.Join(tmpPath, "archive.tar")

	archiveWriter, err := os.Create(archivePath)
	assert.NoError(t, err)
	defer archiveWriter.Close()

	_, err = CompressTar(archiveWriter, dir, &state.Consumer{})
	assert.NoError(t, err)

	xSettings := ExtractSettings{
		Consumer: &state.Consumer{},
	}

	t.Logf("Extracting over non-existent destination")
	_, err = ExtractTar(archivePath, extractedDir, xSettings)
	assert.NoError(t, err)

	t.Logf("Extracting over already-extracted dir")
	_, err = ExtractTar(archivePath, extractedDir, xSettings)
	assert.NoError(t, err)

	t.Logf("Extracting, one of the dirs is a file now")
	assert.NoError(t, os.RemoveAll(filepath.Join(extractedDir, "subdir")))
	dumbFile, err := os.Create(filepath.Join(extractedDir, "subdir"))
	assert.NoError(t, err)
	assert.NoError(t, dumbFile.Close())

	_, err = ExtractTar(archivePath, extractedDir, xSettings)
	assert.NoError(t, err)
}
