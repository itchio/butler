package diag

import (
	"context"
	"net/http"
	"time"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/pkg/errors"
)

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("diag", "(Advanced) Run some diagnostics")
	ctx.Register(cmd, do)
}

func do(mc *mansion.Context) {
	comm.Opf("Running diagnostics...")
	ctx := context.Background()

	numProblems := 0

	runTest := func(name string, t func() error) {
		comm.Opf("Test: %s...", name)
		err := t()
		if err != nil {
			comm.Warnf("Failed: %+v", err)
			numProblems++
		}
	}

	httpTest := func(url string, expectedStatusCode int) func() error {
		return func() error {
			ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				return err
			}

			res, err := mc.HTTPClient.Do(req)
			if err != nil {
				return err
			}

			if res.StatusCode != expectedStatusCode {
				return errors.Errorf("expected HTTP status code (%d), got (%d)", expectedStatusCode, res.StatusCode)
			}
			return nil
		}
	}

	runTest("CDN reachable", httpTest("https://static.itch.io/ping.txt", 200))
	runTest("Web reachable", httpTest("https://itch.io/static/ping.txt", 200))
	runTest("API reachable", httpTest("https://api.itch.io/login", 405))
	runTest("Broth reachable", httpTest("https://broth.itch.ovh", 200))

	if numProblems > 0 {
		comm.Dief("%d tests failed", numProblems)
	}

	comm.Statf("Everything went fine!")
}
