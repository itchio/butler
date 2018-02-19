package clean_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/cmd/clean"
	"github.com/itchio/wharf/wtest"
	"github.com/stretchr/testify/assert"
)

func withTestDirectory(f func(testDir string) error) error {
	testDir, err := ioutil.TempDir("", "cmd-clean-tests")
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = os.MkdirAll(testDir, 0755)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer os.RemoveAll(testDir)

	err = f(testDir)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	return nil
}

func TestBadPlanPath(t *testing.T) {
	err := clean.Do("notafile.garbage")
	assert.NotNil(t, err)
}

func TestBadJSON(t *testing.T) {
	wtest.Must(t, withTestDirectory(func(testDir string) error {
		planPath := filepath.Join(testDir, "plan.json")
		invalidJSON := "this is not valid json { { { ] ] ] ]- - -"
		err := ioutil.WriteFile(planPath, []byte(invalidJSON), 0644)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		assert.Error(t, clean.Do(planPath))
		return nil
	}))
}

func TestAlreadyRemoved(t *testing.T) {
	wtest.Must(t, withTestDirectory(func(testDir string) error {
		planPath := filepath.Join(testDir, "plan.json")
		planContents := fmt.Sprintf(`{
		  "basePath": %#v,
		  "entries": [
			"already-removed"
		  ]
		}`, testDir)
		err := ioutil.WriteFile(planPath, []byte(planContents), 0644)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		assert.NoError(t, clean.Do(planPath))
		return nil
	}))
}

func TestRemoveFail(t *testing.T) {
	wtest.Must(t, withTestDirectory(func(testDir string) error {
		planPath := filepath.Join(testDir, "badfile")
		// Try to remove plan, which in this test, we don't close.
		// This should fail on some platforms and succeed on others.
		planContents := fmt.Sprintf(`{
		  "basePath": %#v,
		  "entries": [
			"nonempty"
		  ]
		}`, testDir)
		pf, err := os.OpenFile(planPath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		_, err = pf.Write([]byte(planContents))
		if err != nil {
			return errors.Wrap(err, 0)
		}

		// Well, since it fails on some platforms and succeeds on others,
		// we can't assert anything here.
		clean.Do(planPath)

		return nil
	}))
}

func TestHappyPath(t *testing.T) {
	wtest.Must(t, withTestDirectory(func(testDir string) error {
		planPath := filepath.Join(testDir, "plan.json")
		planContents := fmt.Sprintf(`{
		  "basePath": %#v,
		  "entries": [
			"exists.txt",
			"a-directory"
		  ]
		}`, testDir)
		err := ioutil.WriteFile(planPath, []byte(planContents), 0644)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		// prepare files to be cleaned
		aFilePath := filepath.Join(testDir, "exists.txt")
		err = ioutil.WriteFile(aFilePath, []byte{'P', 'K'}, 0644)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		aDirPath := filepath.Join(testDir, "a-directory")
		err = os.Mkdir(aDirPath, 0755)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		assert.NoError(t, clean.Do(planPath))

		// make sure they're gone
		_, err = os.Stat(aFilePath)
		assert.Error(t, err)
		assert.True(t, os.IsNotExist(err))
		_, err = os.Stat(aDirPath)
		assert.Error(t, err)
		assert.True(t, os.IsNotExist(err))

		return nil
	}))
}
