package diag

import (
	"context"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/elefant"
	"github.com/itchio/httpkit/eos"
	"github.com/pkg/errors"
)

type Params struct {
	Net   bool
	Glibc bool
}

var params Params

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("diag", "(Advanced) Run some diagnostics")
	cmd.Flag("net", "Run network connectivity tests").Default("1").BoolVar(&params.Net)
	cmd.Flag("glibc", "Run glibc version test").Default("0").BoolVar(&params.Glibc)
	ctx.Register(cmd, do)
}

func do(mc *mansion.Context) {
	consumer := comm.NewStateConsumer()

	consumer.Opf("Running diagnostics...")
	ctx := context.Background()

	numProblems := 0

	runTest := func(name string, t func() error) {
		consumer.Opf("Test: %s...", name)
		err := t()
		if err != nil {
			consumer.Warnf("Failed: %+v", err)
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

	if params.Net {
		runTest("CDN reachable", httpTest("https://static.itch.io/ping.txt", 200))
		runTest("Web reachable", httpTest("https://itch.io/static/ping.txt", 200))
		runTest("API reachable", httpTest("https://api.itch.io/login", 405))
		runTest("Broth reachable", httpTest("https://broth.itch.ovh", 200))
	}
	if params.Glibc {
		if runtime.GOOS != "linux" {
			consumer.Infof("Skipping glibc check, not on Linux")
		} else {
			runTest("GLIBC version", func() error {
				exe, err := os.Executable()
				if err != nil {
					return err
				}

				f, err := eos.Open(exe)
				if err != nil {
					return err
				}
				defer f.Close()

				props, err := elefant.Probe(f, &elefant.ProbeParams{
					Consumer: consumer,
				})
				if err != nil {
					return err
				}

				tokens := strings.Split(props.GlibcVersion, ".")
				if len(tokens) != 2 {
					return errors.Errorf("expected two tokens when splitting %q by '.'", props.GlibcVersion)
				}
				major, err := strconv.ParseInt(tokens[0], 10, 64)
				if err != nil {
					return errors.WithStack(err)
				}
				minor, err := strconv.ParseInt(tokens[1], 10, 64)
				if err != nil {
					return errors.WithStack(err)
				}

				if major != 2 || minor > 27 {
					return errors.Errorf("butler should require GLIBC 2.27 at most, but this binary requires %s", props.GlibcVersion)
				}
				consumer.Infof("Required glibc version: %s", props.GlibcVersion)
				return nil
			})
		}
	}

	if numProblems > 0 {
		comm.Dief("%d tests failed", numProblems)
	}

	consumer.Statf("Everything went fine!")
}
