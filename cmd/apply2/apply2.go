package apply2

import (
	"encoding/gob"
	"os"
	"path"
	"time"

	"github.com/dchest/safefile"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/httpkit/progress"
	"github.com/itchio/savior/filesource"
	"github.com/itchio/wharf/eos/option"
	"github.com/itchio/wharf/pools/fspool"
	"github.com/itchio/wharf/pwr/bowl"
	"github.com/itchio/wharf/pwr/patcher"
	"github.com/pkg/errors"
)

var args = struct {
	patch     *string
	dir       *string
	old       *string
	stopEarly *bool
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("apply2", "(Advanced) Use a patch to resumably patch a directory to a new version")
	args.patch = cmd.Arg("patch", "Patch file (.pwr), previously generated with the `diff` command.").Required().String()
	args.old = cmd.Arg("old", "Directory with old files").Required().String()
	args.dir = cmd.Flag("dir", "Directory for patched files and checkpoints").Short('d').Required().String()
	args.stopEarly = cmd.Flag("stop-early", "Stop after emitting checkpoint").Bool()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(&Params{
		Patch:     *args.patch,
		Old:       *args.old,
		Dir:       *args.dir,
		StopEarly: *args.stopEarly,
	}))
}

type Params struct {
	Patch     string
	Old       string
	Dir       string
	StopEarly bool
}

func Do(params *Params) error {
	startTime := time.Now()

	patch := params.Patch
	old := params.Old
	dir := params.Dir

	consumer := comm.NewStateConsumer()

	patchSource, err := filesource.Open(patch, option.WithConsumer(comm.NewStateConsumer()))
	if err != nil {
		return errors.Wrap(err, "opening patch")
	}

	p, err := patcher.New(patchSource, consumer)
	if err != nil {
		return errors.Wrap(err, "creating patcher")
	}

	consumer.Opf("Patching %s", dir)
	comm.StartProgressWithTotalBytes(patchSource.Size())

	var checkpoint *patcher.Checkpoint
	checkpointPath := path.Join(dir, "checkpoint.bwl")

	checkpointFile, err := os.Open(checkpointPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return errors.Wrap(err, "opening checkpoint")
		}
	} else {
		defer checkpointFile.Close()

		checkpoint = &patcher.Checkpoint{}

		dec := gob.NewDecoder(checkpointFile)
		err := dec.Decode(checkpoint)
		if err != nil {
			return errors.Wrap(err, "decoding checkpoint")
		}

		// yay, we have a checkpoint!
	}

	lastSaveTime := time.Now()
	saveInterval := 2 * time.Second

	p.SetSaveConsumer(&patcherSaveConsumer{
		shouldSave: func() bool {
			consumer.Progress(p.Progress())
			return time.Since(lastSaveTime) > saveInterval
		},
		save: func(c *patcher.Checkpoint) (patcher.AfterSaveAction, error) {
			lastSaveTime = time.Now()

			checkpointFile, err := safefile.Create(checkpointPath, 0644)
			if err != nil {
				return patcher.AfterSaveStop, errors.Wrap(err, "creating checkpoint file")
			}
			defer checkpointFile.Close()

			enc := gob.NewEncoder(checkpointFile)
			err = enc.Encode(c)
			if err != nil {
				return patcher.AfterSaveStop, errors.Wrap(err, "encoding checkpoint")
			}

			err = checkpointFile.Commit()
			if err != nil {
				return patcher.AfterSaveStop, errors.Wrap(err, "committing checkpoint file")
			}

			if params.StopEarly {
				return patcher.AfterSaveStop, nil
			}
			return patcher.AfterSaveContinue, nil
		},
	})

	targetPool := fspool.New(p.GetTargetContainer(), old)

	bowl, err := bowl.NewFreshBowl(&bowl.FreshBowlParams{
		SourceContainer: p.GetSourceContainer(),
		TargetContainer: p.GetTargetContainer(),
		TargetPool:      targetPool,
		OutputFolder:    dir,
	})
	if err != nil {
		return errors.Wrap(err, "creating fresh bowl")
	}

	err = p.Resume(checkpoint, targetPool, bowl)
	if err != nil {
		if errors.Cause(err) == patcher.ErrStop {
			comm.EndProgress()
			consumer.Statf("Stopped early!")
			return nil
		}
		return errors.Wrap(err, "patching")
	}
	comm.EndProgress()

	out := p.GetSourceContainer()
	duration := time.Since(startTime)
	consumer.Statf("%s (%s) @ %s / s (%s total)", progress.FormatBytes(out.Size), out.Stats(), progress.FormatBPS(out.Size, duration), duration)

	return nil
}

type patcherSaveConsumer struct {
	shouldSave func() bool
	save       func(checkpoint *patcher.Checkpoint) (patcher.AfterSaveAction, error)
}

var _ patcher.SaveConsumer = (*patcherSaveConsumer)(nil)

func (psc *patcherSaveConsumer) ShouldSave() bool {
	return psc.shouldSave()
}

func (psc *patcherSaveConsumer) Save(checkpoint *patcher.Checkpoint) (patcher.AfterSaveAction, error) {
	return psc.save(checkpoint)
}
