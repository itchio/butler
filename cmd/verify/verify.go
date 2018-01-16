package verify

import (
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/savior/seeksource"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/pwr"
)

var args = struct {
	signature *string
	dir       *string
	wounds    *string
	heal      *string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("verify", "(Advanced) Use a signature to verify the integrity of a directory")
	args.signature = cmd.Arg("signature", "Path to read signature file from").Required().String()
	args.dir = cmd.Arg("dir", "Path of directory to verify").Required().String()
	args.wounds = cmd.Flag("wounds", "When given, writes wounds to this path").String()
	args.heal = cmd.Flag("heal", "When given, heal wounds using this path").String()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(ctx, *args.signature, *args.dir, *args.wounds, *args.heal))
}

func Do(ctx *mansion.Context, signaturePath string, dir string, woundsPath string, healPath string) error {
	if woundsPath == "" {
		if healPath == "" {
			comm.Opf("Verifying %s", dir)
		} else {
			comm.Opf("Verifying %s, healing as we go", dir)
		}
	} else {
		if healPath == "" {
			comm.Opf("Verifying %s, writing wounds to %s", dir, woundsPath)
		} else {
			comm.Dief("Options --wounds and --heal cannot be used at the same time")
		}
	}
	startTime := time.Now()

	signatureReader, err := eos.Open(signaturePath)
	if err != nil {
		return errors.Wrap(err, 1)
	}
	defer signatureReader.Close()

	signatureSource := seeksource.FromFile(signatureReader)

	_, err = signatureSource.Resume(nil)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	signature, err := pwr.ReadSignature(signatureSource)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	vc := &pwr.ValidatorContext{
		Consumer:   comm.NewStateConsumer(),
		WoundsPath: woundsPath,
		HealPath:   healPath,
	}

	comm.StartProgressWithTotalBytes(signature.Container.Size)

	err = vc.Validate(dir, signature)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	comm.EndProgress()

	prettySize := humanize.IBytes(uint64(signature.Container.Size))
	perSecond := humanize.IBytes(uint64(float64(signature.Container.Size) / time.Since(startTime).Seconds()))
	comm.Statf("%s (%s) @ %s/s\n", prettySize, signature.Container.Stats(), perSecond)

	if vc.WoundsConsumer.HasWounds() {
		if healer, ok := vc.WoundsConsumer.(pwr.Healer); ok {
			comm.Statf("%s corrupted data found, %s healed", humanize.IBytes(uint64(vc.WoundsConsumer.TotalCorrupted())), humanize.IBytes(uint64(healer.TotalHealed())))
		} else {
			comm.Dief("%s corrupted data found", humanize.IBytes(uint64(vc.WoundsConsumer.TotalCorrupted())))
		}
	}

	return nil
}
