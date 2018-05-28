package apply2

import (
	"encoding/gob"
	"log"
	"os"
	"path"
	"time"

	"github.com/dchest/safefile"
	"github.com/itchio/butler/cmd/sizeof"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/httpkit/progress"
	"github.com/itchio/savior/filesource"
	"github.com/itchio/wharf/eos/option"
	"github.com/itchio/wharf/pools/fspool"
	"github.com/itchio/wharf/pwr/bowl"
	"github.com/itchio/wharf/pwr/patcher"
	"github.com/itchio/wharf/state"
	"github.com/pkg/errors"
)

var args = struct {
	patch           string
	old             string
	dir             string
	stagingDir      string
	stopEarly       bool
	simulateRestart bool
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("apply2", "(Advanced) Use a patch to resumably patch a directory to a new version")
	cmd.Arg("patch", "Patch file (.pwr), previously generated with the `diff` command.").Required().StringVar(&args.patch)
	cmd.Arg("old", "Directory with old files").Required().StringVar(&args.old)
	cmd.Flag("dir", "Directory for patched files and checkpoints").Short('d').Required().StringVar(&args.dir)
	cmd.Flag("staging-dir", "Directory for temporary files").StringVar(&args.stagingDir)
	cmd.Flag("stop-early", "Stop after emitting checkpoint").Hidden().BoolVar(&args.stopEarly)
	cmd.Flag("simulate-restart", "Simulate restarting").Hidden().BoolVar(&args.simulateRestart)
	ctx.Register(cmd, func(ctx *mansion.Context) {
		consumer := comm.NewStateConsumer()
		for {
			err := Do(ctx, consumer)
			if errors.Cause(err) == patcher.ErrStop {
				consumer.Statf("Stopped early!")
				if args.simulateRestart {
					log.Printf("Restarting!")
					time.Sleep(500 * time.Millisecond)
					continue
				}
			}
			ctx.Must(err)
			break
		}
	})
}

func Do(ctx *mansion.Context, consumer *state.Consumer) error {
	startTime := time.Now()

	patch := args.patch
	old := args.old
	dir := args.dir

	patchSource, err := filesource.Open(patch, option.WithConsumer(comm.NewStateConsumer()))
	if err != nil {
		return errors.WithMessage(err, "opening patch")
	}

	p, err := patcher.New(patchSource, consumer)
	if err != nil {
		return errors.WithMessage(err, "creating patcher")
	}

	consumer.Opf("Patching %s", dir)
	comm.StartProgressWithTotalBytes(patchSource.Size())

	var checkpoint *patcher.Checkpoint
	checkpointPath := path.Join(dir, "checkpoint.bwl")

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

			if args.stopEarly || args.simulateRestart {
				return patcher.AfterSaveStop, nil
			}
			return patcher.AfterSaveContinue, nil
		},
	})

	targetPool := fspool.New(p.GetTargetContainer(), old)

	var bwl bowl.Bowl
	if args.stagingDir == "" {
		consumer.Infof("Using fresh bowl")
		bwl, err = bowl.NewFreshBowl(&bowl.FreshBowlParams{
			SourceContainer: p.GetSourceContainer(),
			TargetContainer: p.GetTargetContainer(),
			TargetPool:      targetPool,
			OutputFolder:    dir,
		})
	} else {
		consumer.Infof("Using overlay bowl")
		bwl, err = bowl.NewOverlayBowl(&bowl.OverlayBowlParams{
			SourceContainer: p.GetSourceContainer(),
			TargetContainer: p.GetTargetContainer(),
			TargetPool:      targetPool,
			OutputFolder:    dir,
			StageFolder:     args.stagingDir,
		})
	}
	if err != nil {
		return errors.WithMessage(err, "creating fresh bowl")
	}

	err = p.Resume(checkpoint, targetPool, bwl)
	if err != nil {
		if errors.Cause(err) == patcher.ErrStop {
			comm.EndProgress()
			return err
		}
		return errors.WithMessage(err, "patching")
	}

	if args.stagingDir != "" {
		stagingDirSize, err := sizeof.Do(args.stagingDir)
		if err != nil {
			return err
		}
		consumer.Statf("Before commit, staging dir is %s", progress.FormatBytes(stagingDirSize))
	}

	consumer.Opf("Committing...")
	err = bwl.Commit()
	if err != nil {
		return errors.WithMessage(err, "committing bowl")
	}

	comm.EndProgress()

	out := p.GetSourceContainer()
	duration := time.Since(startTime)
	consumer.Statf("%s (%s) @ %s (%s total)",
		progress.FormatBytes(out.Size), out.Stats(),
		progress.FormatBPS(out.Size, duration),
		progress.FormatDuration(duration))

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
