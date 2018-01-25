package operate

import (
	"fmt"

	"github.com/itchio/butler/installer"
	"github.com/itchio/butler/installer/bfs"
	"github.com/itchio/savior/seeksource"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/tlc"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/pwr"
)

func heal(oc *OperationContext, meta *MetaSubcontext, receiptIn *bfs.Receipt) (*installer.InstallResult, error) {
	consumer := oc.Consumer()

	params := meta.data.InstallParams

	if params.Build == nil {
		return nil, errors.New("heal: missing build")
	}

	signatureURL := sourceURL(consumer, params, "signature")
	archiveURL := sourceURL(consumer, params, "archive")

	healSpec := fmt.Sprintf("archive,%s", archiveURL)

	vctx := &pwr.ValidatorContext{
		Consumer:   consumer,
		NumWorkers: 1,
		HealPath:   healSpec,
	}

	consumer.Infof("Fetching + parsing signature...")

	signatureFile, err := eos.Open(signatureURL)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}
	defer signatureFile.Close()

	signatureSource := seeksource.FromFile(signatureFile)

	_, err = signatureSource.Resume(nil)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	sigInfo, err := pwr.ReadSignature(signatureSource)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	consumer.Infof("Nursing wharf build back to health...")

	oc.StartProgress()
	err = vctx.Validate(params.InstallFolder, sigInfo)
	oc.EndProgress()

	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	consumer.Infof("✓ Everything healed")

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
