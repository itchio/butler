package verify

import (
	"context"
	"time"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/httpkit/progress"
	"github.com/itchio/savior/seeksource"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/pwr"
	"github.com/pkg/errors"
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
	ctx.Must(Do(*args.signature, *args.dir, *args.wounds, *args.heal))
}

func Do(signaturePath string, dir string, woundsPath string, healPath string) error {
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
		return errors.Wrap(err, "opening signature file")
	}
	defer signatureReader.Close()

	signatureSource := seeksource.FromFile(signatureReader)

	_, err = signatureSource.Resume(nil)
	if err != nil {
		return errors.WithStack(err)
	}

	signature, err := pwr.ReadSignature(context.Background(), signatureSource)
	if err != nil {
		return errors.Wrap(err, "reading signature file")
	}

	vc := &pwr.ValidatorContext{
		Consumer:   comm.NewStateConsumer(),
		WoundsPath: woundsPath,
		HealPath:   healPath,
	}

	comm.StartProgressWithTotalBytes(signature.Container.Size)

	err = vc.Validate(context.Background(), dir, signature)
	if err != nil {
		return errors.Wrap(err, "while validating")
	}

	comm.EndProgress()

	perSecond := progress.FormatBPS(signature.Container.Size, time.Since(startTime))
	comm.Statf("%s @ %s\n", signature.Container, perSecond)

	if vc.WoundsConsumer.HasWounds() {
		if healer, ok := vc.WoundsConsumer.(pwr.Healer); ok {
			comm.Statf("%s corrupted data found, %s healed", progress.FormatBytes(vc.WoundsConsumer.TotalCorrupted()), progress.FormatBytes(healer.TotalHealed()))
		} else {
			comm.Dief("%s corrupted data found", progress.FormatBytes(vc.WoundsConsumer.TotalCorrupted()))
		}
	}

	return nil
}
