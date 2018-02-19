package operate

import (
	"fmt"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/itchio/butler/installer"
	"github.com/itchio/butler/installer/bfs"
	"github.com/itchio/savior/seeksource"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/tlc"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/pwr"
)

func heal(oc *OperationContext, meta *MetaSubcontext, isub *InstallSubcontext, receiptIn *bfs.Receipt) (*installer.InstallResult, error) {
	consumer := oc.Consumer()
	istate := isub.data

	params := meta.data.InstallParams

	if params.Build == nil {
		return nil, errors.New("heal: missing build")
	}

	signatureURL := sourceURL(consumer, istate, params, "signature")
	archiveURL := sourceURL(consumer, istate, params, "archive")

	healSpec := fmt.Sprintf("archive,%s", archiveURL)

	vc := &pwr.ValidatorContext{
		Consumer:   consumer,
		NumWorkers: 1,
		HealPath:   healSpec,
	}

	signatureFile, err := eos.Open(signatureURL)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}
	defer signatureFile.Close()

	stat, err := signatureFile.Stat()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	consumer.Infof("Fetching + parsing %s signature...",
		humanize.IBytes(uint64(stat.Size())),
	)

	signatureSource := seeksource.FromFile(signatureFile)

	timeBeforeSig := time.Now()

	_, err = signatureSource.Resume(nil)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	sigInfo, err := pwr.ReadSignature(signatureSource)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	consumer.Infof("✓ Fetched signature in %s, dealing with %s container",
		time.Since(timeBeforeSig),
		humanize.IBytes(uint64(sigInfo.Container.Size)),
	)

	consumer.Infof("Healing container...")

	timeBeforeHeal := time.Now()

	oc.StartProgress()
	err = vc.Validate(params.InstallFolder, sigInfo)
	oc.EndProgress()

	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	healDuration := time.Since(timeBeforeHeal)
	containerSize := sigInfo.Container.Size

	if vc.WoundsConsumer.HasWounds() {
		if healer, ok := vc.WoundsConsumer.(pwr.Healer); ok {
			totalHealed := healer.TotalHealed()
			perSec := humanize.IBytes(uint64(float64(totalHealed) / healDuration.Seconds()))

			consumer.Infof("✓ %s corrupted data found (of %s total), %s healed @ %s/s, %s total",
				humanize.IBytes(uint64(vc.WoundsConsumer.TotalCorrupted())),
				humanize.IBytes(uint64(sigInfo.Container.Size)),
				humanize.IBytes(uint64(totalHealed)),
				perSec,
				healDuration,
			)
		} else {
			consumer.Warnf("%s corrupted data found (of %s total)",
				humanize.IBytes(uint64(vc.WoundsConsumer.TotalCorrupted())),
				humanize.IBytes(uint64(sigInfo.Container.Size)),
			)
		}
	} else {
		perSec := humanize.IBytes(uint64(float64(containerSize) / healDuration.Seconds()))

		consumer.Infof("✓ All %s were healthy (checked @ %s/s, %s total)",
			humanize.IBytes(uint64(containerSize)),
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
		return nil, errors.Wrap(err, 0)
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
		Files: make([]string, len(c.Files)+len(c.Symlinks)),
	}

	i := 0

	for _, f := range c.Files {
		res.Files[i] = f.Path
		i++
	}
	for _, f := range c.Symlinks {
		res.Files[i] = f.Path
		i++
	}

	return res
}
