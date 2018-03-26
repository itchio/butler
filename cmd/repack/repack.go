package repack

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/savior/countingsource"
	"github.com/itchio/savior/seeksource"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/wire"
	"github.com/pkg/errors"
)

var args = struct {
	inPath  *string
	outPath *string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("repack", "Recompress a wharf patch using a different compression algorithm/format").Hidden()
	args.inPath = cmd.Arg("inpath", "Path of patch to recompress").Required().String()
	args.outPath = cmd.Flag("outpath", "Path of patch to recompress").Short('o').String()
	ctx.Register(cmd, do)
}

type Params struct {
	InPath      string
	OutPath     string
	Compression *pwr.CompressionSettings
}

func do(ctx *mansion.Context) {
	if *args.outPath == "" {
		// benchmark!
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
	} else {
		// output!
		comp := ctx.CompressionSettings()
		ctx.Must(Do(&Params{
			InPath:      *args.inPath,
			OutPath:     *args.outPath,
			Compression: &comp,
		}))
	}
}

func Do(params *Params) error {
	bench := params.OutPath == ""
	consumer := comm.NewStateConsumer()

	dr, err := eos.Open(params.InPath)
	if err != nil {
		return errors.WithStack(err)
	}
	defer dr.Close()

	stats, err := dr.Stat()
	if err != nil {
		return errors.WithStack(err)
	}

	source := seeksource.FromFile(dr)

	cs := countingsource.New(source, func(count int64) {
		comm.Progress(source.Progress())
	})

	_, err = cs.Resume(nil)
	if err != nil {
		return errors.WithStack(err)
	}

	var dw io.Writer
	if bench {
		dw = ioutil.Discard
	} else {
		dw, err = os.Create(params.OutPath)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	w := counter.NewWriter(dw)

	rawInWire := wire.NewReadContext(source)

	err = rawInWire.ExpectMagic(pwr.PatchMagic)
	if err != nil {
		return errors.WithStack(err)
	}

	header := &pwr.PatchHeader{}
	err = rawInWire.ReadMessage(header)
	if err != nil {
		return errors.WithStack(err)
	}

	inWire, err := pwr.DecompressWire(rawInWire, header.Compression)
	if err != nil {
		return errors.WithStack(err)
	}

	rawOutWire := wire.NewWriteContext(w)

	err = rawOutWire.WriteMagic(pwr.PatchMagic)
	if err != nil {
		return errors.WithStack(err)
	}

	if !bench {
		consumer.Opf("Repacking %s (%s) from %s to %s", stats.Name(), humanize.IBytes(uint64(source.Size())), header.Compression, params.Compression)
		comm.StartProgressWithTotalBytes(source.Size())
	}

	header.Compression = params.Compression

	err = rawOutWire.WriteMessage(header)
	if err != nil {
		return errors.WithStack(err)
	}

	var megaBytesPerSec float64

	err = func() error {
		outWire, err := pwr.CompressWire(rawOutWire, header.Compression)
		if err != nil {
			return errors.WithStack(err)
		}
		defer outWire.Close()

		startTime := time.Now()
		rd := inWire.GetSource()
		wr := outWire.Writer()
		numBytes, err := io.Copy(wr, rd)
		if err != nil {
			return errors.WithStack(err)
		}
		duration := time.Since(startTime)

		megaBytesPerSec = float64(numBytes) / 1024.0 / 1024.0 / duration.Seconds()

		if !bench {
			comm.EndProgress()
			perSec := humanize.IBytes(uint64(float64(numBytes) / duration.Seconds()))
			consumer.Statf("Repacked %s @ %s/s", humanize.IBytes(uint64(numBytes)), perSec)
		}

		return nil
	}()
	if err != nil {
		return errors.WithStack(err)
	}

	inSize := source.Size()

	outSize := w.Count()

	if bench {
		columns := []string{
			fmt.Sprintf("%s-q%d", header.Compression.Algorithm, header.Compression.Quality),
			fmt.Sprintf("%f", float64(outSize)/float64(inSize)),
			fmt.Sprintf("%f", megaBytesPerSec),
		}

		fmt.Printf("%s\n", strings.Join(columns, ","))
	} else {
		consumer.Statf("%s => %s (%.3f as large as the input)",
			humanize.IBytes(uint64(inSize)),
			humanize.IBytes(uint64(outSize)),
			float64(outSize)/float64(inSize),
		)
		consumer.Statf("Wrote to %s", params.OutPath)
	}

	return nil
}
