package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/itchio/butler/comm"
)

type CleanPlan struct {
	BasePath string   `json:"basePath"`
	Entries  []string `json:"entries"`
}

func clean(planPath string) {
	startTime := time.Now()

	contents, err := ioutil.ReadFile(planPath)
	must(err)

	plan := CleanPlan{}

	err = json.Unmarshal(contents, &plan)
	must(err)

	comm.Logf("Cleaning %d entries from %s", len(plan.Entries), plan.BasePath)

	for _, entry := range plan.Entries {
		fullPath := filepath.Join(plan.BasePath, entry)
		err := os.Remove(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				// ignore
			} else {
				must(err)
			}
		}
	}

	duration := time.Since(startTime)
	entriesPerSec := float64(len(plan.Entries)) / duration.Seconds()
	comm.Statf("Done in %s (%.2f entries/s)", duration, entriesPerSec)
}
