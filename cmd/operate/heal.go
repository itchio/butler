package operate

import (
	"fmt"
	"time"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/installer"
	"github.com/itchio/butler/installer/bfs"
	"github.com/itchio/httpkit/progress"
	"github.com/itchio/savior/seeksource"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/eos/option"
	"github.com/itchio/wharf/tlc"

	"github.com/itchio/wharf/pwr"
	"github.com/pkg/errors"
)

func heal(oc *OperationContext, meta *MetaSubcontext, isub *InstallSubcontext, receiptIn *bfs.Receipt) error {
	consumer := oc.Consumer()
	istate := isub.Data
	params := meta.Data

	if params.Build == nil {
		return errors.New("heal: missing build")
	}

	messages.TaskStarted.Notify(oc.rc, &butlerd.TaskStartedNotification{
		Reason: butlerd.TaskReasonInstall,
		Type:   butlerd.TaskTypeHeal,
		Game:   params.Game,
		Upload: params.Upload,
		Build:  params.Build,
	})

	signatureURL := sourceURL(consumer, istate, params, "signature")
	archiveURL := sourceURL(consumer, istate, params, "archive")

	healSpec := fmt.Sprintf("archive,%s", archiveURL)

	vc := &pwr.ValidatorContext{
		Consumer:   consumer,
		NumWorkers: 1,
		HealPath:   healSpec,
	}

	signatureFile, err := eos.Open(signatureURL, option.WithConsumer(consumer))
	if err != nil {
		return errors.WithStack(err)
	}
	defer signatureFile.Close()

	stat, err := signatureFile.Stat()
	if err != nil {
		return errors.WithStack(err)
	}

	consumer.Infof("Fetching + parsing %s signature...",
		progress.FormatBytes(stat.Size()),
	)

	signatureSource := seeksource.FromFile(signatureFile)

	timeBeforeSig := time.Now()

	_, err = signatureSource.Resume(nil)
	if err != nil {
		return errors.WithStack(err)
	}

	sigInfo, err := pwr.ReadSignature(oc.ctx, signatureSource)
	if err != nil {
		return errors.WithStack(err)
	}

	consumer.Infof("✓ Fetched signature in %s, dealing with %s container",
		time.Since(timeBeforeSig),
		progress.FormatBytes(sigInfo.Container.Size),
	)

	consumer.Infof("Healing container...")

	timeBeforeHeal := time.Now()

	oc.rc.StartProgress()
	err = vc.Validate(oc.ctx, params.InstallFolder, sigInfo)
	oc.rc.EndProgress()
	if err != nil {
		return errors.WithStack(err)
	}

	healDuration := time.Since(timeBeforeHeal)
	containerSize := sigInfo.Container.Size

	if vc.WoundsConsumer.HasWounds() {
		if healer, ok := vc.WoundsConsumer.(pwr.Healer); ok {
			totalHealed := healer.TotalHealed()
			perSec := progress.FormatBPS(totalHealed, healDuration)

			consumer.Infof("✓ %s corrupted data found (of %s total), %s healed @ %s/s, %s total",
				progress.FormatBytes(vc.WoundsConsumer.TotalCorrupted()),
				progress.FormatBytes(sigInfo.Container.Size),
				progress.FormatBytes(totalHealed),
				perSec,
				healDuration,
			)
		} else {
			consumer.Warnf("%s corrupted data found (of %s total)",
				progress.FormatBytes(vc.WoundsConsumer.TotalCorrupted()),
				progress.FormatBytes(sigInfo.Container.Size),
			)
		}
	} else {
		perSec := progress.FormatBPS(containerSize, healDuration)

		consumer.Infof("✓ All %s were healthy (checked @ %s/s, %s total)",
			progress.FormatBytes(containerSize),
			perSec,
			healDuration,
		)
	}

	res := resultForContainer(sigInfo.Container)

	consumer.Infof("Busting ghosts...")

	err = bfs.BustGhosts(&bfs.BustGhostsParams{
		Folder:   params.InstallFolder,
		NewFiles: res.Files,
		Receipt:  receiptIn,

		Consumer: consumer,
	})
	if err != nil {
		return errors.WithStack(err)
	}

	return commitInstall(oc, &CommitInstallParams{
		InstallFolder: params.InstallFolder,

		// if we're healing, it's a wharf-enabled upload,
		// and if it's a wharf-enabled upload, our installer is "archive".
		InstallerName: "archive",
		Game:          params.Game,
		Upload:        params.Upload,
		Build:         params.Build,

		InstallResult: res,
	})
}

func resultForContainer(c *tlc.Container) *installer.InstallResult {
	res := &installer.InstallResult{
		Files: nil,
	}

	for _, f := range c.Files {
		res.Files = append(res.Files, f.Path)
	}
	for _, f := range c.Symlinks {
		res.Files = append(res.Files, f.Path)
	}

	return res
}
