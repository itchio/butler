package operate

import (
	"fmt"
	"time"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/installer"
	"github.com/itchio/butler/installer/bfs"

	"github.com/itchio/savior/seeksource"

	"github.com/itchio/httpkit/eos"
	"github.com/itchio/httpkit/eos/option"

	"github.com/itchio/headway/united"

	"github.com/itchio/lake/tlc"

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

	messages.TaskStarted.Notify(oc.rc, butlerd.TaskStartedNotification{
		Reason: butlerd.TaskReasonInstall,
		Type:   butlerd.TaskTypeHeal,
		Game:   params.Game,
		Upload: params.Upload,
		Build:  params.Build,
	})

	client := oc.rc.Client(params.Access.APIKey)

	signatureURL := MakeSourceURL(client, consumer, istate.DownloadSessionID, params, "signature")
	archiveURL := MakeSourceURL(client, consumer, istate.DownloadSessionID, params, "archive")

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
		united.FormatBytes(stat.Size()),
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
		united.FormatBytes(sigInfo.Container.Size),
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
			perSec := united.FormatBPS(totalHealed, healDuration)

			consumer.Infof("✓ %s corrupted data found (of %s total), %s healed @ %s/s, %s total",
				united.FormatBytes(vc.WoundsConsumer.TotalCorrupted()),
				united.FormatBytes(sigInfo.Container.Size),
				united.FormatBytes(totalHealed),
				perSec,
				united.FormatDuration(healDuration),
			)
		} else {
			consumer.Warnf("%s corrupted data found (of %s total)",
				united.FormatBytes(vc.WoundsConsumer.TotalCorrupted()),
				united.FormatBytes(sigInfo.Container.Size),
			)
		}
	} else {
		perSec := united.FormatBPS(containerSize, healDuration)

		consumer.Infof("✓ All %s were healthy (checked @ %s/s, %s total)",
			united.FormatBytes(containerSize),
			perSec,
			healDuration,
		)
	}

	err = isub.EventSink(oc).PostEvent(butlerd.InstallEvent{
		Heal: &butlerd.HealInstallEvent{
			TotalCorrupted:   vc.WoundsConsumer.TotalCorrupted(),
			AppliedCaseFixes: vc.CaseFixStats != nil && len(vc.CaseFixStats.Fixes) > 0,
		},
	})
	if err != nil {
		return err
	}

	res := resultForContainer(sigInfo.Container)

	consumer.Infof("Busting ghosts...")

	var bustGhostStats bfs.BustGhostStats
	err = bfs.BustGhosts(bfs.BustGhostsParams{
		Folder:   params.InstallFolder,
		NewFiles: res.Files,
		Receipt:  receiptIn,

		Consumer: consumer,
		Stats:    &bustGhostStats,
	})
	if err != nil {
		return errors.WithStack(err)
	}
	err = isub.EventSink(oc).PostGhostBusting("heal", bustGhostStats)
	if err != nil {
		return err
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
