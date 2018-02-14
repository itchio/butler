package clean

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/itchio/butler/mansion"
	"github.com/stretchr/testify/assert"
)

func initTestDirectory() error {
	return os.Mkdir("tests", os.ModeDir)
}

func removeTestDirectory() error {
	return os.Remove("tests")
}

func TestBadPlanPath(t *testing.T) {
	ctx := &mansion.Context{}
	err := Do(ctx, "notafile.garbage")
	assert.NotNil(t, err)
}

func TestBadJSON(t *testing.T) {
	err := initTestDirectory()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err = removeTestDirectory()
		if err != nil {
			t.Fatal(err)
		}
	}()
	ctx := &mansion.Context{}
	planPath := filepath.Join("tests", "badfile")
	f, err := os.Create(planPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := os.Remove(planPath)
		if err != nil {
			t.Fatal(err)
		}
	}()
	_, err = f.Write([]byte("this is not valid json { { { ] ] ] ]- - -"))
	if err != nil {
		t.Fatal(err)
	}
	err = f.Close()
	if err != nil {
		t.Fatal(err)
	}
	err = Do(ctx, planPath)
	assert.NotNil(t, err)
}

func TestAlreadyRemoved(t *testing.T) {
	err := initTestDirectory()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err = removeTestDirectory()
		if err != nil {
			t.Fatal(err)
		}
	}()
	ctx := &mansion.Context{}
	planPath := filepath.Join("tests", "plan.json")
	f, err := os.Create(planPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := os.Remove(planPath)
		if err != nil {
			t.Fatal(err)
		}
	}()
	_, err = f.Write([]byte(
		`{
		  "basePath": "tests",
		  "entries": [
			"already-removed"
		  ]
		}`,
	))
	if err != nil {
		t.Fatal(err)
	}
	err = f.Close()
	if err != nil {
		t.Fatal(err)
	}
	err = Do(ctx, planPath)
	assert.Nil(t, err)
}

func TestRemoveFail(t *testing.T) {
	err := initTestDirectory()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err = removeTestDirectory()
		if err != nil {
			t.Fatal(err)
		}
	}()
	ctx := &mansion.Context{}
	// Try to remove plan, which in this test, we don't close.
	// This should fail on some platforms and succeed on others.
	planPath := filepath.Join("tests", "plan.json")
	f, err := os.Create(planPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err = f.Close()
		if err != nil {
			t.Fatal(err)
		}
		err := os.Remove(planPath)
		if err != nil {
			t.Fatal(err)
		}
	}()
	_, err = f.Write([]byte(
		`{
		  "basePath": "tests",
		  "entries": [
			"plan.json"
		  ]
		}`,
	))
	if err != nil {
		t.Fatal(err)
	}
	err = Do(ctx, planPath)
	// Todo: update for OSes that this succeeds on,
	// hypothetically linux is ok with removing open file descriptors
	assert.NotNil(t, err)
}

func TestHappyPath(t *testing.T) {
	err := initTestDirectory()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err = removeTestDirectory()
		if err != nil {
			t.Fatal(err)
		}
	}()
	ctx := &mansion.Context{}
	planPath := filepath.Join("tests", "plan.json")
	f, err := os.Create(planPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := os.Remove(planPath)
		if err != nil {
			t.Fatal(err)
		}
	}()
	_, err = f.Write([]byte(
		`{
		  "basePath": "tests",
		  "entries": [
			"exists.txt",
			"a-directory"
		  ]
		}`,
	))
	if err != nil {
		t.Fatal(err)
	}
	err = f.Close()
	if err != nil {
		t.Fatal(err)
	}
	f, err = os.Create(filepath.Join("tests", "exists.txt"))
	if err != nil {
		t.Fatal(err)
	}
	err = f.Close()
	if err != nil {
		t.Fatal(err)
	}
	err = os.Mkdir(filepath.Join("tests", "a-directory"), os.ModeDir)
	if err != nil {
		t.Fatal(err)
	}
	err = Do(ctx, planPath)
	assert.Nil(t, err)
}
