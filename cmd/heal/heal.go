package heal

import (
	"context"
	"io"
	"os"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/savior/seeksource"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/lake/tlc"
	"github.com/itchio/wharf/wire"
	"github.com/pkg/errors"
)

var args = struct {
	dir    *string
	wounds *string
	spec   *string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("heal", "(Advanced) Heal a directory using a list of wounds and a heal spec")
	args.dir = cmd.Arg("dir", "Path of directory to heal").Required().String()
	args.wounds = cmd.Arg("wounds", "Path of wounds file").Required().String()
	args.spec = cmd.Arg("spec", "Path of spec to heal with").Required().String()
	ctx.Register(cmd, do)
}

type Params struct {
	Dir        string
	WoundsPath string
	HealSpec   string
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(&Params{
		Dir:        *args.dir,
		WoundsPath: *args.wounds,
		HealSpec:   *args.spec,
	}))
}

func Do(params *Params) error {
	dir := params.Dir
	woundsPath := params.WoundsPath
	spec := params.HealSpec

	reader, err := os.Open(woundsPath)
	if err != nil {
		return errors.Wrap(err, "opening wounds")
	}
	defer reader.Close()

	healer, err := pwr.NewHealer(spec, dir)
	if err != nil {
		return errors.Wrap(err, "creating healer")
	}

	consumer := comm.NewStateConsumer()

	healer.SetConsumer(consumer)

	source := seeksource.FromFile(reader)

	_, err = source.Resume(nil)
	if err != nil {
		return errors.Wrap(err, "reading wounds")
	}

	rctx := wire.NewReadContext(source)

	err = rctx.ExpectMagic(pwr.WoundsMagic)
	if err != nil {
		return errors.Wrap(err, "reading wounds magic")
	}

	wh := &pwr.WoundsHeader{}
	err = rctx.ReadMessage(wh)
	if err != nil {
		return errors.Wrap(err, "reading wounds header")
	}

	container := &tlc.Container{}
	err = rctx.ReadMessage(container)
	if err != nil {
		return errors.Wrap(err, "reading container from wounds file")
	}

	wounds := make(chan *pwr.Wound)
	errs := make(chan error)

	comm.StartProgress()

	go func() {
		errs <- healer.Do(context.Background(), container, wounds)
	}()

	wound := &pwr.Wound{}
	for {
		wound.Reset()
		err = rctx.ReadMessage(wound)
		if err != nil {
			if err == io.EOF {
				// all good
				break
			}
		}

		select {
		case wounds <- wound:
			// all good
		case healErr := <-errs:
			return errors.WithStack(healErr)
		}
	}

	close(wounds)
	healErr := <-errs

	comm.EndProgress()

	if healErr != nil {
		return errors.WithStack(healErr)
	}

	comm.Opf("All healed!")
	return nil
}
