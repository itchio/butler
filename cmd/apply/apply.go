package apply

import (
	"context"
	"encoding/gob"
	"os"
	"path"
	"time"

	"github.com/itchio/wharf/pwr"

	"github.com/dchest/safefile"
	"github.com/itchio/butler/cmd/sizeof"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/headway/state"
	"github.com/itchio/headway/united"
	"github.com/itchio/httpkit/eos/option"
	"github.com/itchio/lake/pools/fspool"
	"github.com/itchio/savior/filesource"
	"github.com/itchio/wharf/pwr/bowl"
	"github.com/itchio/wharf/pwr/patcher"
	"github.com/pkg/errors"
)

var params = Params{}

type Params struct {
	Patch           string
	Old             string
	Dir             string
	StagingDir      string
	StopEarly       bool
	SimulateRestart bool
	Signature       string
	SaveInterval    float64
	Consumer        *state.Consumer
}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("apply", "(Advanced) Use a patch to resumably patch a directory to a new version")
	cmd.Arg("patch", "Patch file (.pwr), previously generated with the `diff` command.").Required().StringVar(&params.Patch)
	cmd.Arg("old", "Directory with old files").Required().StringVar(&params.Old)
	cmd.Flag("dir", "Directory for patched files and checkpoints").Short('d').StringVar(&params.Dir)
	cmd.Flag("staging-dir", "Directory for temporary files").Required().StringVar(&params.StagingDir)
	cmd.Flag("stop-early", "Stop after emitting checkpoint").BoolVar(&params.StopEarly)
	cmd.Flag("simulate-restart", "Simulate restarting").BoolVar(&params.SimulateRestart)
	cmd.Flag("signature", "Signature file (.pws) to verify build against after patching").StringVar(&params.Signature)
	cmd.Flag("save-interval", "Save interval").Default("2").Float64Var(&params.SaveInterval)
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	params.Consumer = comm.NewStateConsumer()
	for {
		err := Do(params)
		if errors.Cause(err) == patcher.ErrStop {
			if params.SimulateRestart {
				continue
			}
		}
		ctx.Must(err)
		break
	}
}

func Do(params Params) error {
	consumer := params.Consumer
	startTime := time.Now()

	patch := params.Patch
	old := params.Old
	dir := params.Dir
	stagingDir := params.StagingDir

	if dir == "" {
		consumer.Opf("Patching %s (in-place)", old)
	} else {
		consumer.Opf("Patching %s (fresh)", dir)
	}

	patchSource, err := filesource.Open(patch, option.WithConsumer(comm.NewStateConsumer()))
	if err != nil {
		return errors.WithMessage(err, "opening patch")
	}

	p, err := patcher.New(patchSource, consumer)
	if err != nil {
		return errors.WithMessage(err, "creating patcher")
	}

	var checkpoint *patcher.Checkpoint
	checkpointPath := path.Join(stagingDir, "checkpoint.bwl")

	checkpointFile, err := os.Open(checkpointPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return errors.WithMessage(err, "opening checkpoint")
		}
	} else {
		defer checkpointFile.Close()

		checkpoint = &patcher.Checkpoint{}

		dec := gob.NewDecoder(checkpointFile)
		err := dec.Decode(checkpoint)
		if err != nil {
			return errors.WithMessage(err, "decoding checkpoint")
		}

		// yay, we have a checkpoint!
	}

	lastSaveTime := time.Now()
	saveInterval := time.Duration(float64(time.Second) * params.SaveInterval)
	consumer.Infof("Save interval: %s", saveInterval)

	p.SetSaveConsumer(&patcherSaveConsumer{
		shouldSave: func() bool {
			consumer.Progress(p.Progress())
			return time.Since(lastSaveTime) > saveInterval
		},
		save: func(c *patcher.Checkpoint) (patcher.AfterSaveAction, error) {
			lastSaveTime = time.Now()

			checkpointFile, err := safefile.Create(checkpointPath, 0o644)
			if err != nil {
				return patcher.AfterSaveStop, errors.WithMessage(err, "creating checkpoint file")
			}
			defer checkpointFile.Close()

			enc := gob.NewEncoder(checkpointFile)
			err = enc.Encode(c)
			if err != nil {
				return patcher.AfterSaveStop, errors.WithMessage(err, "encoding checkpoint")
			}

			err = checkpointFile.Commit()
			if err != nil {
				return patcher.AfterSaveStop, errors.WithMessage(err, "committing checkpoint file")
			}

			if params.StopEarly || params.SimulateRestart {
				return patcher.AfterSaveStop, nil
			}
			return patcher.AfterSaveContinue, nil
		},
	})

	targetPool := fspool.New(p.GetTargetContainer(), old)

	var bwl bowl.Bowl
	if dir != "" {
		bwl, err = bowl.NewFreshBowl(bowl.FreshBowlParams{
			SourceContainer: p.GetSourceContainer(),
			TargetContainer: p.GetTargetContainer(),
			TargetPool:      targetPool,
			OutputFolder:    dir,
		})
	} else {
		bwl, err = bowl.NewOverlayBowl(bowl.OverlayBowlParams{
			SourceContainer: p.GetSourceContainer(),
			TargetContainer: p.GetTargetContainer(),
			OutputFolder:    old,
			StageFolder:     stagingDir,
		})
	}
	if err != nil {
		return errors.WithMessage(err, "creating fresh bowl")
	}

	comm.StartProgressWithTotalBytes(patchSource.Size())
	err = p.Resume(checkpoint, targetPool, bwl)
	comm.EndProgress()
	if err != nil {
		if errors.Cause(err) == patcher.ErrStop {
			comm.EndProgress()
			consumer.Statf("Stopped early! (@ %.2f%%)", p.Progress()*100.0)
			return err
		}
		return errors.WithMessage(err, "patching")
	}

	if params.StagingDir != "" {
		stagingDirSize, err := sizeof.Do(params.StagingDir)
		if err != nil {
			return err
		}
		consumer.Statf("Before commit, staging dir is %s", united.FormatBytes(stagingDirSize))
	}

	consumer.Opf("Committing...")
	err = bwl.Commit()
	if err != nil {
		return errors.WithMessage(err, "committing bowl")
	}

	out := p.GetSourceContainer()
	duration := time.Since(startTime)
	consumer.Statf("Patched %s (%s) @ %s (%s total)",
		united.FormatBytes(out.Size), out.Stats(),
		united.FormatBPS(out.Size, duration),
		united.FormatDuration(duration))

	if params.Signature != "" {
		sigSource, err := filesource.Open(params.Signature)
		if err != nil {
			return err
		}
		defer sigSource.Close()

		consumer.Opf("Verifying against signature...")

		sigInfo, err := pwr.ReadSignature(context.Background(), sigSource)
		if err != nil {
			return err
		}

		outputDir := dir
		if outputDir == "" {
			outputDir = old
		}

		vctx := &pwr.ValidatorContext{
			FailFast: true,
			Consumer: consumer,
		}

		comm.Progress(0.0)
		comm.StartProgress()
		err = vctx.Validate(context.Background(), outputDir, sigInfo)
		comm.EndProgress()
		if err != nil {
			return err
		}

		err = pwr.AssertNoGhosts(outputDir, sigInfo)
		if err != nil {
			return err
		}

		consumer.Statf("Phew, everything checks out!")
	}

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
