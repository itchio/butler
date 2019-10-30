package diag

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/elefant"
	"github.com/itchio/headway/counter"
	"github.com/itchio/headway/state"
	"github.com/itchio/headway/tracker"
	"github.com/itchio/httpkit/eos"
	"github.com/itchio/savior"
	"github.com/itchio/savior/zipextractor"
	"github.com/pkg/errors"
)

type Params struct {
	All bool

	Net   bool
	Glibc bool
	Speed bool
}

var paramsAll = Params{
	Net:   true,
	Glibc: true,
	Speed: true,
}

var params Params

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("diag", "(Advanced) Run some diagnostics")
	cmd.Flag("all", "Run all tests").Default("0").BoolVar(&params.All)
	cmd.Flag("net", "Run network connectivity tests").Default("1").BoolVar(&params.Net)
	cmd.Flag("glibc", "Run glibc version test").Default("0").BoolVar(&params.Glibc)
	cmd.Flag("speed", "Run speed test").Default("0").BoolVar(&params.Speed)
	ctx.Register(cmd, do)
}

type fakeSaveConsumer struct {
	c chan struct{}
}

var _ savior.SaveConsumer = (*fakeSaveConsumer)(nil)

func (fsc *fakeSaveConsumer) ShouldSave(copiedBytes int64) bool {
	select {
	case <-fsc.c:
		return true
	default:
		return false
	}
}

func (fsc *fakeSaveConsumer) Save(checkpoint *savior.ExtractorCheckpoint) (savior.AfterSaveAction, error) {
	return savior.AfterSaveStop, nil
}

func do(mc *mansion.Context) {
	if params.All {
		params = paramsAll
	}

	consumer := comm.NewStateConsumer()

	consumer.Opf("Running diagnostics...")
	ctx := context.Background()

	numProblems := 0

	runTest := func(name string, t func() (string, error)) {
		output, err := t()
		consumer.Infof("%25s | %s", name, output)

		if err != nil {
			consumer.Warnf("Failed: %+v", err)
			numProblems++
		}
	}

	httpTest := func(url string, expectedStatusCode int) func() (string, error) {
		return func() (string, error) {
			ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
			defer cancel()

			before := time.Now()

			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				return "", err
			}

			res, err := mc.HTTPClient.Do(req)
			if err != nil {
				return "", err
			}

			if res.StatusCode != expectedStatusCode {
				return "", errors.Errorf("expected HTTP status code (%d), got (%d)", expectedStatusCode, res.StatusCode)
			}

			rtt := time.Since(before)
			return fmt.Sprintf("%s", rtt), nil
		}
	}

	if params.Net {
		runTest("CDN reachable", httpTest("https://static.itch.io/ping.txt", 200))
		runTest("Web reachable", httpTest("https://itch.io/static/ping.txt", 200))
		runTest("API reachable", httpTest("https://api.itch.io/login", 405))
		runTest("Broth reachable", httpTest("https://broth.itch.ovh", 200))
	}

	if params.Glibc {
		runTest("GLIBC version", func() (string, error) {
			if runtime.GOOS != "linux" {
				return "skip", nil
			}

			exe, err := os.Executable()
			if err != nil {
				return "", err
			}

			f, err := eos.Open(exe)
			if err != nil {
				return "", err
			}
			defer f.Close()

			props, err := elefant.Probe(f, &elefant.ProbeParams{
				Consumer: consumer,
			})
			if err != nil {
				return "", err
			}

			tokens := strings.Split(props.GlibcVersion, ".")
			if len(tokens) != 2 {
				return "", errors.Errorf("expected two tokens when splitting %q by '.'", props.GlibcVersion)
			}
			major, err := strconv.ParseInt(tokens[0], 10, 64)
			if err != nil {
				return "", errors.WithStack(err)
			}
			minor, err := strconv.ParseInt(tokens[1], 10, 64)
			if err != nil {
				return "", errors.WithStack(err)
			}

			if major != 2 || minor > 27 {
				return "", fmt.Errorf("butler should require GLIBC 2.27 at most, but this binary requires %s", props.GlibcVersion)
			}
			return props.GlibcVersion, nil
		})
	}

	if params.Speed {
		testFileURL := "https://broth.itch.ovh/speedtest/100mib/LATEST/archive/default"

		runTracker := func(contentLength int64, cancel context.CancelFunc, f func(t tracker.Tracker) error) (string, error) {
			t := tracker.New(tracker.Opts{
				ByteAmount: &tracker.ByteAmount{Value: contentLength},
			})

			go func() {
				defer cancel()
				<-time.NewTimer(5 * time.Second).C
			}()

			err := f(t)
			if err != nil {
				return "", err
			}

			stats := t.Finish()
			toBPS := func(speed float64) tracker.BPS {
				if speed == math.MaxFloat64 {
					speed = 1
				}
				return tracker.BPS{Value: stats.MinSpeed() * float64(stats.ByteAmount().Value)}
			}

			min := toBPS(stats.MinSpeed())
			avg := toBPS(stats.AverageSpeed())
			max := toBPS(stats.MaxSpeed())
			return fmt.Sprintf("min %v :: avg %v :: max %v", min, avg, max), nil
		}

		trackDownload := func(contentLength int64, src io.ReadCloser, dst io.Writer) (string, error) {
			return runTracker(contentLength, func() {
				src.Close()
			}, func(t tracker.Tracker) error {
				cw := counter.NewWriterCallback(func(count int64) {
					prog := float64(count) / float64(contentLength)
					t.SetProgress(prog)
				}, dst)

				_, _ = io.Copy(cw, src)
				return nil
			})
		}

		runTest("go https request", func() (string, error) {
			req, err := http.NewRequest("GET", testFileURL, nil)
			if err != nil {
				return "", err
			}

			res, err := mc.HTTPClient.Do(req)
			if err != nil {
				return "", err
			}

			return trackDownload(res.ContentLength, res.Body, ioutil.Discard)
		})

		runTest("eos Copy", func() (string, error) {
			f, err := eos.Open(testFileURL)
			if err != nil {
				return "", err
			}

			stats, err := f.Stat()
			if err != nil {
				return "", err
			}

			return trackDownload(stats.Size(), f, ioutil.Discard)
		})

		runTest("eos Copy (to disk)", func() (string, error) {
			d, err := ioutil.TempDir("", "")
			if err != nil {
				return "", err
			}
			defer os.RemoveAll(d)

			df, err := os.Create(filepath.Join(d, "speedtest.dat"))
			if err != nil {
				return "", err
			}

			f, err := eos.Open(testFileURL)
			if err != nil {
				return "", err
			}

			stats, err := f.Stat()
			if err != nil {
				return "", err
			}

			return trackDownload(stats.Size(), f, df)
		})

		runTest("eos Extract (to disk)", func() (string, error) {
			d, err := ioutil.TempDir("", "")
			if err != nil {
				return "", err
			}
			defer os.RemoveAll(d)

			f, err := eos.Open(testFileURL)
			if err != nil {
				return "", err
			}

			stats, err := f.Stat()
			if err != nil {
				return "", err
			}

			ze, err := zipextractor.New(f, stats.Size())
			if err != nil {
				return "", err
			}

			sink := savior.FolderSink{
				Directory: d,
			}

			var totalSize int64
			for _, entry := range ze.Entries() {
				totalSize += entry.CompressedSize
			}

			c := make(chan struct{})
			return runTracker(totalSize, func() { close(c) }, func(t tracker.Tracker) error {
				ze.SetConsumer(&state.Consumer{
					OnProgress: t.SetProgress,
				})
				ze.SetSaveConsumer(&fakeSaveConsumer{c: c})

				_, _ = ze.Resume(nil, &sink)
				return nil
			})
		})
	}

	if numProblems > 0 {
		comm.Dief("%d tests failed", numProblems)
	}

	consumer.Statf("Everything went fine!")
}
