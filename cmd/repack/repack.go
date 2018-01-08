package repack

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/wire"
)

var args = struct {
	inPath *string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("repack", "Recompress a wharf patch using a different compression algorithm/format")
	args.inPath = cmd.Arg("inpath", "Path of patch to recompress").Required().String()
	ctx.Register(cmd, do)
}

type Params struct {
	InPath      string
	Compression *pwr.CompressionSettings
}

func do(ctx *mansion.Context) {
	headers := []string{
		"algorithm", "relative size", "compression speed",
	}
	fmt.Printf("%s\n", strings.Join(headers, ","))

	algos := []pwr.CompressionAlgorithm{
		pwr.CompressionAlgorithm_ZSTD,
		pwr.CompressionAlgorithm_BROTLI,
	}
	qualities := []int32{
		1,
		3,
		6,
		9,
	}
	for _, algo := range algos {
		for _, quality := range qualities {
			comp := &pwr.CompressionSettings{
				Algorithm: algo,
				Quality:   quality,
			}

			ctx.Must(Do(&Params{
				InPath:      *args.inPath,
				Compression: comp,
			}))
		}
	}
}

func Do(params *Params) error {
	var columns []string

	r, err := eos.Open(params.InPath)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer r.Close()

	stats, err := r.Stat()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	dw := ioutil.Discard
	w := counter.NewWriter(dw)

	rawInWire := wire.NewReadContext(r)

	err = rawInWire.ExpectMagic(pwr.PatchMagic)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	header := &pwr.PatchHeader{}
	err = rawInWire.ReadMessage(header)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	inWire, err := pwr.DecompressWire(rawInWire, header.Compression)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	rawOutWire := wire.NewWriteContext(w)

	err = rawOutWire.WriteMagic(pwr.PatchMagic)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	header.Compression = params.Compression

	err = rawOutWire.WriteMessage(header)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	var megaBytesPerSec float64

	err = func() error {
		outWire, err := pwr.CompressWire(rawOutWire, header.Compression)
		if err != nil {
			return errors.Wrap(err, 0)
		}
		defer outWire.Close()

		startTime := time.Now()
		rd := inWire.Reader()
		wr := outWire.Writer()
		numBytes, err := io.Copy(wr, rd)
		if err != nil {
			return errors.Wrap(err, 0)
		}
		duration := time.Since(startTime)

		megaBytesPerSec = float64(numBytes) / 1024.0 / 1024.0 / duration.Seconds()

		return nil
	}()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	inSize := stats.Size()

	outSize := w.Count()

	columns = append(columns, fmt.Sprintf("%s-q%d", header.Compression.Algorithm, header.Compression.Quality))
	columns = append(columns, fmt.Sprintf("%f", float64(outSize)/float64(inSize)))
	columns = append(columns, fmt.Sprintf("%f", megaBytesPerSec))

	fmt.Printf("%s\n", strings.Join(columns, ","))

	return nil
}
