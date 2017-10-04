package cp

import (
	"io"
	"os"
	"path/filepath"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/butler/butler"
	"github.com/itchio/butler/cmd/dl"
	"github.com/itchio/butler/comm"
	"github.com/itchio/httpkit/httpfile"
	"github.com/itchio/httpkit/retrycontext"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/eos"
)

var args = struct {
	src    *string
	dest   *string
	resume *bool
}{}

func Register(ctx *butler.Context) {
	cmd := ctx.App.Command("cp", "Copy src to dest").Hidden()
	args.src = cmd.Arg("src", "File to read from").Required().String()
	args.dest = cmd.Arg("dest", "File to write to").Required().String()
	args.resume = cmd.Flag("resume", "Try to resume if dest is partially written (doesn't check existing data)").Bool()
	ctx.Register(cmd, do)
}

func do(ctx *butler.Context) {
	ctx.Must(Do(ctx, *args.src, *args.dest, *args.resume))
}

func Do(ctx *butler.Context, srcPath string, destPath string, resume bool) error {
	retryCtx := retrycontext.NewDefault()
	retryCtx.Settings.Consumer = comm.NewStateConsumer()

	for retryCtx.ShouldTry() {
		err := Try(ctx, srcPath, destPath, resume)
		if err != nil {
			if dl.IsIntegrityError(err) {
				retryCtx.Retry(err.Error())
				continue
			}

			// if it's not an integrity error, just bubble it up
			return err
		}

		return nil
	}

	return errors.New("cp: too many errors, giving up")
}

func Try(ctx *butler.Context, srcPath string, destPath string, resume bool) error {
	src, err := eos.Open(srcPath)
	if err != nil {
		return err
	}

	defer src.Close()

	dir := filepath.Dir(destPath)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	flags := os.O_CREATE | os.O_WRONLY

	dest, err := os.OpenFile(destPath, flags, 0644)
	if err != nil {
		return err
	}

	defer dest.Close()

	stats, err := src.Stat()
	if err != nil {
		return err
	}

	totalBytes := int64(stats.Size())
	var startOffset int64
	var copiedBytes int64
	start := time.Now()

	err = func() error {
		if resume {
			startOffset, err = dest.Seek(0, io.SeekEnd)
			if err != nil {
				return err
			}

			if startOffset == 0 {
				comm.Logf("Downloading %s", humanize.IBytes(uint64(totalBytes)))
			} else if startOffset > totalBytes {
				comm.Logf("Existing data too big (%s > %s), starting over", humanize.IBytes(uint64(startOffset)), humanize.IBytes(uint64(totalBytes)))
				startOffset, err = dest.Seek(0, io.SeekStart)
				if err != nil {
					return err
				}
			} else if startOffset == totalBytes {
				comm.Logf("All %s already there", humanize.IBytes(uint64(totalBytes)))
				return nil
			}

			comm.Logf("Resuming at %s / %s", humanize.IBytes(uint64(startOffset)), humanize.IBytes(uint64(totalBytes)))

			_, err = src.Seek(startOffset, io.SeekStart)
			if err != nil {
				return err
			}
		} else {
			comm.Logf("Downloading %s", humanize.IBytes(uint64(totalBytes)))
		}

		comm.Progress(float64(startOffset) / float64(totalBytes))
		comm.StartProgressWithTotalBytes(totalBytes)

		cw := counter.NewWriterCallback(func(count int64) {
			alpha := float64(startOffset+count) / float64(totalBytes)
			comm.Progress(alpha)
		}, dest)

		copiedBytes, err = io.Copy(cw, src)
		if err != nil {
			return err
		}
		comm.EndProgress()

		return os.Truncate(destPath, totalBytes)
	}()

	if err != nil {
		return err
	}

	if hf, ok := src.(*httpfile.HTTPFile); ok {
		header := hf.GetHeader()
		if header != nil {
			err = dl.CheckIntegrity(header, totalBytes, destPath)
			if err != nil {
				comm.Log("Integrity checks failed, truncating")
				os.Truncate(destPath, 0)
				return errors.Wrap(err, 1)
			}
		} else {
			comm.Debugf("Not performing integrity checks (no header)")
		}
	} else {
		comm.Debugf("Not performing integrity checks (not an HTTP resource)")
	}

	totalDuration := time.Since(start)
	prettyStartOffset := humanize.IBytes(uint64(startOffset))
	prettySize := humanize.IBytes(uint64(copiedBytes))
	perSecond := humanize.IBytes(uint64(float64(totalBytes-startOffset) / totalDuration.Seconds()))
	comm.Statf("%s + %s copied @ %s/s\n", prettyStartOffset, prettySize, perSecond)

	return nil
}
