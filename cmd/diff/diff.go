package diff

import (
	"io"
	"os"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/filtering"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/savior/seeksource"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/pools"
	"github.com/itchio/wharf/pools/nullpool"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
)

var args = struct {
	old    *string
	new    *string
	patch  *string
	verify *bool
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("diff", "(Advanced) Compute the difference between two directories or .zip archives. Stores the patch in `patch.pwr`, and a signature in `patch.pwr.sig` for integrity checks and further diff.")
	args.old = cmd.Arg("old", "Directory or .zip archive (slower) with older files, or signature file generated from old directory.").Required().String()
	args.new = cmd.Arg("new", "Directory or .zip archive (slower) with newer files").Required().String()
	args.patch = cmd.Arg("patch", "Path to write the patch file (recommended extension is `.pwr`) The signature file will be written to the same path, with .sig added to the end.").Default("patch.pwr").String()
	args.verify = cmd.Flag("verify", "Make sure generated patch applies cleanly by applying it (slower)").Bool()
	ctx.Register(cmd, do)
}

type Params struct {
	// Target is the old version of the data
	Target string
	// Source is the new version of the data
	Source string
	// Patch is where to write the patch
	Patch       string
	Compression pwr.CompressionSettings
	// Verify enables dry-run apply patch validation (slow)
	Verify bool
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(&Params{
		Target:      *args.old,
		Source:      *args.new,
		Patch:       *args.patch,
		Compression: ctx.CompressionSettings(),
		Verify:      *args.verify,
	}))
}

func Do(params *Params) error {
	startTime := time.Now()

	targetSignature := &pwr.SignatureInfo{}

	if params.Target == "" {
		return errors.New("diff: must specify Target")
	}
	if params.Source == "" {
		return errors.New("diff: must specify Source")
	}
	if params.Patch == "" {
		return errors.New("diff: must specify Patch")
	}

	readAsSignature := func() error {
		// Signature file perhaps?
		signatureReader, err := eos.Open(params.Target)
		if err != nil {
			return errors.Wrap(err, 0)
		}
		defer signatureReader.Close()

		stats, _ := signatureReader.Stat()
		if stats.IsDir() {
			return wire.ErrFormat
		}

		signatureSource := seeksource.FromFile(signatureReader)
		_, err = signatureSource.Resume(nil)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		readSignature, err := pwr.ReadSignature(signatureSource)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		targetSignature = readSignature

		comm.Opf("Read signature from %s", params.Target)

		return nil
	}

	err := readAsSignature()

	if err != nil {
		if errors.Is(err, wire.ErrFormat) || errors.Is(err, io.EOF) {
			// must be a container then
			targetSignature.Container, err = tlc.WalkAny(params.Target, &tlc.WalkOpts{Filter: filtering.FilterPaths})
			// Container (dir, archive, etc.)
			comm.Opf("Hashing %s", params.Target)

			comm.StartProgress()
			targetPool, err := pools.New(targetSignature.Container, params.Target)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			targetSignature.Hashes, err = pwr.ComputeSignature(targetSignature.Container, targetPool, comm.NewStateConsumer())
			comm.EndProgress()
			if err != nil {
				return errors.Wrap(err, 0)
			}

			{
				prettySize := humanize.IBytes(uint64(targetSignature.Container.Size))
				perSecond := humanize.IBytes(uint64(float64(targetSignature.Container.Size) / time.Since(startTime).Seconds()))
				comm.Statf("%s (%s) @ %s/s\n", prettySize, targetSignature.Container.Stats(), perSecond)
			}
		} else {
			return errors.Wrap(err, 0)
		}
	}

	startTime = time.Now()

	sourceContainer, err := tlc.WalkAny(params.Source, &tlc.WalkOpts{Filter: filtering.FilterPaths})
	if err != nil {
		return errors.Wrap(err, 0)
	}

	sourcePool, err := pools.New(sourceContainer, params.Source)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	patchWriter, err := os.Create(params.Patch)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer patchWriter.Close()

	signaturePath := params.Patch + ".sig"
	signatureWriter, err := os.Create(signaturePath)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer signatureWriter.Close()

	patchCounter := counter.NewWriter(patchWriter)
	signatureCounter := counter.NewWriter(signatureWriter)

	dctx := &pwr.DiffContext{
		SourceContainer: sourceContainer,
		Pool:            sourcePool,

		TargetContainer: targetSignature.Container,
		TargetSignature: targetSignature.Hashes,

		Consumer:    comm.NewStateConsumer(),
		Compression: &params.Compression,
	}

	comm.Opf("Diffing %s", params.Source)
	comm.StartProgress()
	err = dctx.WritePatch(patchCounter, signatureCounter)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	comm.EndProgress()

	totalDuration := time.Since(startTime)
	{
		prettySize := humanize.IBytes(uint64(sourceContainer.Size))
		perSecond := humanize.IBytes(uint64(float64(sourceContainer.Size) / totalDuration.Seconds()))
		comm.Statf("%s (%s) @ %s/s\n", prettySize, sourceContainer.Stats(), perSecond)
	}

	{
		prettyPatchSize := humanize.IBytes(uint64(patchCounter.Count()))
		percReused := 100.0 * float64(dctx.ReusedBytes) / float64(dctx.FreshBytes+dctx.ReusedBytes)
		relToNew := 100.0 * float64(patchCounter.Count()) / float64(sourceContainer.Size)
		prettyFreshSize := humanize.IBytes(uint64(dctx.FreshBytes))

		comm.Statf("Re-used %.2f%% of old, added %s fresh data", percReused, prettyFreshSize)
		comm.Statf("%s patch (%.2f%% of the full size) in %s", prettyPatchSize, relToNew, totalDuration)
	}

	if params.Verify {
		comm.Opf("Applying patch to verify it...")
		_, err := signatureWriter.Seek(0, io.SeekStart)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		signatureSource := seeksource.FromFile(signatureWriter)

		_, err = signatureSource.Resume(nil)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		signature, err := pwr.ReadSignature(signatureSource)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		actx := &pwr.ApplyContext{
			OutputPool: &pwr.ValidatingPool{
				Pool:      nullpool.New(sourceContainer),
				Container: sourceContainer,
				Signature: signature,
			},
			TargetPath:      params.Target,
			TargetContainer: targetSignature.Container,

			SourceContainer: sourceContainer,

			Consumer: comm.NewStateConsumer(),
		}

		patchSource := seeksource.FromFile(patchWriter)

		_, err = patchSource.Resume(nil)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		comm.StartProgress()
		err = actx.ApplyPatch(patchSource)
		comm.EndProgress()
		if err != nil {
			return errors.Wrap(err, 0)
		}

		comm.Statf("Patch applies cleanly!")
	}

	return nil
}
