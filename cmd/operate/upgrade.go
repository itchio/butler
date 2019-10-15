package operate

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dchest/safefile"

	"github.com/itchio/butler/installer/bfs"
	itchio "github.com/itchio/go-itchio"

	"github.com/itchio/headway/united"

	"github.com/itchio/savior/filesource"

	"github.com/itchio/httpkit/eos/option"

	"github.com/itchio/lake/pools/fspool"

	"github.com/itchio/wharf/pwr/bowl"
	"github.com/itchio/wharf/pwr/patcher"

	"github.com/pkg/errors"
)

func upgrade(oc *OperationContext, meta *MetaSubcontext, isub *InstallSubcontext, receiptIn *bfs.Receipt) error {
	consumer := oc.Consumer()
	istate := isub.Data

	totalPatches := len(istate.UpgradePath.Builds)
	donePatches := istate.UpgradePathIndex
	remainingPatches := totalPatches - donePatches

	consumer.Infof("Applying %d patches (%d already done)", remainingPatches, donePatches)

	var roughPatchCosts []float64
	var totalPatchCost float64
	for _, b := range istate.UpgradePath.Builds {
		bf := FindBuildFile(b.Files, itchio.BuildFileTypePatch, itchio.BuildFileSubTypeDefault)
		var cost float64
		if bf != nil {
			cost = float64(bf.Size)
		}
		roughPatchCosts = append(roughPatchCosts, cost)
		totalPatchCost += cost
	}
	if totalPatchCost == 0 {
		totalPatchCost += 0.00001
	}

	var donePatchCost float64
	for i := 0; i < istate.UpgradePathIndex; i++ {
		donePatchCost += roughPatchCosts[i]
	}

	{
		formatCost := func(cost float64) string {
			return fmt.Sprintf("%.1f", cost/1024.0)
		}
		var costStrings []string
		for _, cost := range roughPatchCosts {
			costStrings = append(costStrings, formatCost(cost))
		}
		consumer.Debugf("Cost done: %s / %s, repartition: %s",
			formatCost(donePatchCost),
			formatCost(totalPatchCost),
			strings.Join(costStrings, " :: "),
		)
	}

	oc.rc.StartProgress()
	for i := istate.UpgradePathIndex; i < totalPatches; i++ {
		build := istate.UpgradePath.Builds[i]
		cost := roughPatchCosts[i]
		sp := SlicedProgress{
			consumer: consumer,
			start:    donePatchCost / totalPatchCost,
			end:      (donePatchCost + cost) / totalPatchCost,
		}
		err := applyPatch(oc, meta, isub, receiptIn, i, sp)
		if err != nil {
			return errors.WithMessage(err, fmt.Sprintf("while applying patch %d/%d (build %d)", i, totalPatches, build.ID))
		}
		sp.Progress(1)

		donePatchCost += cost
	}
	oc.rc.EndProgress()

	return nil
}

func applyPatch(oc *OperationContext, meta *MetaSubcontext, isub *InstallSubcontext, receiptIn *bfs.Receipt, upgradePathIndex int, progressTarget SlicedProgress) error {
	rc := oc.rc
	consumer := oc.Consumer()
	params := meta.Data
	istate := isub.Data

	build := istate.UpgradePath.Builds[upgradePathIndex]

	LogBuild(consumer, params.Upload, build)

	client := rc.Client(params.Access.APIKey)
	subType := itchio.BuildFileSubTypeDefault
	if FindBuildFile(build.Files, itchio.BuildFileTypePatch, itchio.BuildFileSubTypeOptimized) != nil {
		subType = itchio.BuildFileSubTypeOptimized
	}

	patchURL := client.MakeBuildDownloadURL(itchio.MakeBuildDownloadURLParams{
		Credentials: params.Access.Credentials,
		BuildID:     build.ID,
		Type:        itchio.BuildFileTypePatch,
		SubType:     subType,
		UUID:        istate.DownloadSessionID,
	})

	patchSource, err := filesource.Open(patchURL, option.WithConsumer(consumer))
	if err != nil {
		return errors.Wrap(err, "opening remote patch")
	}

	consumer.Infof("Patch is %s", united.FormatBytes(patchSource.Size()))

	checkpointPath := filepath.Join(oc.StageFolder(), fmt.Sprintf("patch-%d-%s-checkpoint", build.ID, subType))
	consumer.Debugf("Using checkpoint (%s)", checkpointPath)

	p, err := patcher.New(patchSource, consumer)
	if err != nil {
		return errors.Wrap(err, "creating patcher")
	}

	lastSaveTime := time.Now()
	saveInterval := 4 * time.Second
	consumer.Debugf("Save interval: %s", saveInterval)
	p.SetSaveConsumer(&patcherSaveConsumer{
		shouldSave: func() bool {
			progressTarget.Progress(p.Progress())

			select {
			case <-oc.Ctx().Done():
				return true
			default:
				return time.Since(lastSaveTime) > saveInterval
			}
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

			select {
			case <-oc.Ctx().Done():
				return patcher.AfterSaveStop, nil
			default:
				return patcher.AfterSaveContinue, nil
			}
		},
	})

	targetPool := fspool.New(p.GetTargetContainer(), params.InstallFolder)

	stageFolder := filepath.Join(params.StagingFolder, "patch-overlay")

	bowl, err := bowl.NewOverlayBowl(&bowl.OverlayBowlParams{
		TargetContainer: p.GetTargetContainer(),
		SourceContainer: p.GetSourceContainer(),

		OutputFolder: params.InstallFolder,
		StageFolder:  stageFolder,
	})
	if err != nil {
		return errors.WithMessage(err, "while creating bowl for patch")
	}

	var checkpoint *patcher.Checkpoint
	readCheckpoint := func() error {
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
			consumer.Infof("Using checkpoint")
		}
		return nil
	}

	err = readCheckpoint()
	if err != nil {
		return err
	}

	err = p.Resume(checkpoint, targetPool, bowl)
	if err != nil {
		return errors.WithMessage(err, "while applying patch")
	}

	os.RemoveAll(checkpointPath)

	err = bowl.Commit()
	if err != nil {
		return errors.WithMessage(err, "while committing patch")
	}

	consumer.Infof("Patching done, getting signature info...")

	res := resultForContainer(p.GetSourceContainer())

	err = commitInstall(oc, &CommitInstallParams{
		InstallFolder: params.InstallFolder,

		// if we're applying patches, it's a wharf-enabled upload,
		// and if it's a wharf-enabled upload, our installer is "archive".
		InstallerName: "archive",
		Game:          params.Game,
		Upload:        params.Upload,
		Build:         build,

		InstallResult: res,
	})
	if err != nil {
		return errors.WithMessage(err, "while committing install")
	}

	istate.UpgradePathIndex = upgradePathIndex + 1
	err = oc.Save(isub)
	if err != nil {
		return errors.WithMessage(err, "while saving install subcontext")
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
