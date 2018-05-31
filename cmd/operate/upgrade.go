package operate

import (
	"fmt"
	"path/filepath"

	"github.com/itchio/butler/installer/bfs"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/httpkit/progress"
	"github.com/itchio/savior/filesource"
	"github.com/itchio/wharf/eos/option"
	"github.com/itchio/wharf/pools/fspool"
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

	for i := istate.UpgradePathIndex; i < totalPatches; i++ {
		build := istate.UpgradePath.Builds[i]
		err := applyPatch(oc, meta, isub, receiptIn, i)
		if err != nil {
			return errors.WithMessage(err, fmt.Sprintf("while applying patch %d/%d (build %d)", i, totalPatches, build.ID))
		}
	}

	return nil
}

func applyPatch(oc *OperationContext, meta *MetaSubcontext, isub *InstallSubcontext, receiptIn *bfs.Receipt, upgradePathIndex int) error {
	rc := oc.rc
	consumer := oc.Consumer()
	params := meta.Data
	istate := isub.Data

	build := istate.UpgradePath.Builds[upgradePathIndex]

	LogBuild(consumer, params.Upload, build)

	client := rc.Client(params.Access.APIKey)
	subType := itchio.BuildFileSubTypeDefault
	if itchio.FindBuildFileEx(itchio.BuildFileTypePatch, itchio.BuildFileSubTypeOptimized, build.Files) != nil {
		subType = itchio.BuildFileSubTypeOptimized
	}

	patchURL := client.MakeBuildDownloadURL(&itchio.MakeBuildDownloadParams{
		Credentials: params.Access.Credentials,
		BuildID:     build.ID,
		Type:        itchio.BuildFileTypePatch,
		SubType:     subType,
		UUID:        istate.DownloadSessionId,
	})

	patchSource, err := filesource.Open(patchURL, option.WithConsumer(consumer))
	if err != nil {
		return errors.Wrap(err, "opening remote patch")
	}

	consumer.Infof("Patch is %s", progress.FormatBytes(patchSource.Size()))

	p, err := patcher.New(patchSource, consumer)
	if err != nil {
		return errors.Wrap(err, "creating patcher")
	}

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
	err = p.Resume(checkpoint, targetPool, bowl)
	if err != nil {
		return errors.WithMessage(err, "while applying patch")
	}

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
