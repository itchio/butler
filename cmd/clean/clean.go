package clean

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/pkg/errors"
)

var args = struct {
	plan *string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("clean", "Remove a bunch of files").Hidden()
	args.plan = cmd.Arg("plan", "A .json plan containing a list of entries to remove").Required().String()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(*args.plan))
}

func Do(planPath string) error {
	startTime := time.Now()

	contents, err := ioutil.ReadFile(planPath)
	if err != nil {
		return errors.WithStack(err)
	}

	plan := CleanPlan{}

	err = json.Unmarshal(contents, &plan)
	if err != nil {
		return errors.WithStack(err)
	}

	comm.Logf("Cleaning %d entries from %s", len(plan.Entries), plan.BasePath)

	for _, entry := range plan.Entries {
		fullPath := filepath.Join(plan.BasePath, entry)

		// Resolve any path traversal sequences (e.g., "..")
		fullPath = filepath.Clean(fullPath)
		basePath := filepath.Clean(plan.BasePath)

		// Ensure the resolved path is within the base path
		if !strings.HasPrefix(fullPath, basePath) {
			return errors.Errorf("path traversal attempt detected: %s is not within %s", entry, plan.BasePath)
		}
		// Ensure there's a path separator to prevent /basepathABC matching /basepath
		if len(fullPath) > len(basePath) && fullPath[len(basePath)] != filepath.Separator {
			return errors.Errorf("path traversal attempt detected: %s is not within %s", entry, plan.BasePath)
		}

		stat, err := os.Lstat(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				// good, it's already gone!
				continue
			} else {
				return errors.WithStack(err)
			}
		}

		if stat.IsDir() {
			// it's expected that we won't be able
			// to remove all directories, ignore errors
			os.Remove(fullPath)
		} else {
			// files on the other hand, we really do want to remove
			err := os.Remove(fullPath)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}

	duration := time.Since(startTime)
	entriesPerSec := float64(len(plan.Entries)) / duration.Seconds()
	comm.Statf("Done in %s (%.2f entries/s)", duration, entriesPerSec)

	return nil
}

// CleanPlan describes which files exactly to wipe
type CleanPlan struct {
	BasePath string   `json:"basePath"`
	Entries  []string `json:"entries"`
}
