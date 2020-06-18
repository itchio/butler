package verify

import (
	"context"
	"time"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"

	"github.com/itchio/headway/united"

	"github.com/itchio/httpkit/eos"
	"github.com/itchio/savior/seeksource"

	"github.com/itchio/wharf/pwr"
	"github.com/pkg/errors"
)

type Args struct {
	SignaturePath string
	Dir           string
	WoundsPath    string
	HealPath      string
}

var args = Args{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("verify", "(Advanced) Use a signature to verify the integrity of a directory")
	cmd.Arg("signature", "Path to read signature file from").Required().StringVar(&args.SignaturePath)
	cmd.Arg("dir", "Path of directory to verify").Required().StringVar(&args.Dir)
	cmd.Flag("wounds", "When given, writes wounds to this path").StringVar(&args.WoundsPath)
	cmd.Flag("heal", "When given, heal wounds using this path").StringVar(&args.HealPath)
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(args))
}

func Do(args Args) error {
	if args.WoundsPath == "" {
		if args.HealPath == "" {
			comm.Opf("Verifying %s", args.Dir)
		} else {
			comm.Opf("Verifying %s, healing as we go", args.Dir)
		}
	} else {
		if args.HealPath == "" {
			comm.Opf("Verifying %s, writing wounds to %s", args.Dir, args.WoundsPath)
		} else {
			comm.Dief("Options --wounds and --heal cannot be used at the same time")
		}
	}
	startTime := time.Now()

	signatureReader, err := eos.Open(args.SignaturePath)
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
		WoundsPath: args.WoundsPath,
		HealPath:   args.HealPath,
	}

	comm.StartProgressWithTotalBytes(signature.Container.Size)

	err = vc.Validate(context.Background(), args.Dir, signature)
	if err != nil {
		return errors.Wrap(err, "while validating")
	}

	comm.EndProgress()

	perSecond := united.FormatBPS(signature.Container.Size, time.Since(startTime))
	comm.Statf("%s @ %s\n", signature.Container, perSecond)

	if vc.WoundsConsumer.HasWounds() {
		if healer, ok := vc.WoundsConsumer.(pwr.Healer); ok {
			comm.Statf("%s corrupted data found, %s healed", united.FormatBytes(vc.WoundsConsumer.TotalCorrupted()), united.FormatBytes(healer.TotalHealed()))
		} else {
			comm.Dief("%s corrupted data found", united.FormatBytes(vc.WoundsConsumer.TotalCorrupted()))
		}
	}

	return nil
}
