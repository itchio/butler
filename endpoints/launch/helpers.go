package launch

import (
	"crawshaw.io/sqlite"
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/configure"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/endpoints/launch/manifest"
	"github.com/itchio/butler/installer/bfs"
	"github.com/itchio/butler/manager"
	"github.com/itchio/dash"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"github.com/pkg/errors"
)

func getUploadAndBuild(rc *butlerd.RequestContext, info withInstallFolderInfo) (upload *itchio.Upload, build *itchio.Build, err error) {
	consumer := rc.Consumer

	upload = info.cave.Upload
	build = info.cave.Build

	// attempt to refresh upload
	{
		client := rc.Client(info.access.APIKey)
		uploadRes, err := client.GetUpload(rc.Ctx, itchio.GetUploadParams{
			Credentials: info.access.Credentials,
			UploadID:    upload.ID,
		})
		if err != nil {
			consumer.Warnf("Could not refresh upload: %v", err)
		} else {
			upload = uploadRes.Upload
			rc.WithConn(func(conn *sqlite.Conn) {
				models.MustSave(conn, upload, hades.Assoc("Build"))
			})
			consumer.Debugf("Refreshed upload (last updated %s)", upload.UpdatedAt)
		}
	}

	consumer.Infof("Passed:")
	operate.LogUpload(consumer, upload, build)

	receiptIn, err := bfs.ReadReceipt(info.installFolder)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	receiptSaidOtherwise := false

	if receiptIn != nil {
		if receiptIn.Upload != nil {
			if upload == nil || upload.ID != receiptIn.Upload.ID {
				receiptSaidOtherwise = true
				upload = receiptIn.Upload
			}

			if receiptIn.Build != nil {
				if build == nil || build.ID != receiptIn.Build.ID {
					receiptSaidOtherwise = true
					build = receiptIn.Build
				}
			}
		}
	}

	if receiptSaidOtherwise {
		consumer.Warnf("Receipt had different data, switching to:")
		operate.LogUpload(consumer, upload, build)
	}

	return
}

type getTargetsParams struct {
	info  withInstallFolderInfo
	hosts []manager.Host
}

type getTargetsResult struct {
	appManifest *manifest.Manifest
	targets     []*butlerd.LaunchTarget
}

func getTargets(rc *butlerd.RequestContext, params getTargetsParams) (*getTargetsResult, error) {
	err := validation.ValidateStruct(&params,
		validation.Field(&params.hosts, validation.Required),
	)
	if err != nil {
		return nil, err
	}

	consumer := rc.Consumer
	info := params.info

	installFolder := info.installFolder

	upload, _, err := getUploadAndBuild(rc, info)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	appManifest, err := manifest.Read(installFolder)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	verdict, err := configure.Do(configure.Params{
		Path:     installFolder,
		NoFilter: true,
		Consumer: consumer,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var targets []*butlerd.LaunchTarget

	shouldBrowse := false
	if upload != nil {
		switch upload.Type {
		case "soundtrack", "book", "video", "documentation", "mod", "audio_assets", "graphical_assets", "sourcecode":
			consumer.Infof("Upload is of type (%s), forcing shell strategy", upload.Type)
			shouldBrowse = true
		}
	}

	if !shouldBrowse {
		for _, host := range params.hosts {
			hostTargets, err := getTargetsForHost(rc, upload, appManifest, verdict, info, host)
			if err != nil {
				return nil, err
			}
			targets = append(targets, hostTargets...)
		}
	}

	if len(targets) == 0 {
		consumer.Warnf("Falling back to shell strategy")
		targets = append(targets, &butlerd.LaunchTarget{
			Action: &manifest.Action{
				Name: info.cave.Game.Title,
				Path: ".",
			},
			Strategy: &butlerd.StrategyResult{
				FullTargetPath: installFolder,
				Strategy:       butlerd.LaunchStrategyShell,
			},
		})
	}

	var uniqueTargets []*butlerd.LaunchTarget
	fullPathsDone := make(map[string]struct{})
	for _, target := range targets {
		if _, ok := fullPathsDone[target.Strategy.FullTargetPath]; ok {
			consumer.Debugf("Removing duplicate target:\n%s", target.Strategy.String())
			continue
		}

		fullPathsDone[target.Strategy.FullTargetPath] = struct{}{}
		uniqueTargets = append(uniqueTargets, target)
	}
	targets = uniqueTargets

	return &getTargetsResult{
		appManifest,
		targets,
	}, nil
}

func getTargetsForHost(rc *butlerd.RequestContext,
	upload *itchio.Upload,
	appManifest *manifest.Manifest,
	verdict *dash.Verdict,
	info withInstallFolderInfo,
	host manager.Host,
) ([]*butlerd.LaunchTarget, error) {
	consumer := rc.Consumer
	consumer.Opf("Seeking launch targets for host (%s)", host)

	var targets []*butlerd.LaunchTarget

	if appManifest == nil {
		consumer.Infof("No app manifest.")
	} else {
		actions := appManifest.ListActions(host.Runtime.Platform)
		consumer.Statf("%d/%d manifest actions are relevant on (%s)", len(actions), len(appManifest.Actions), host)

		for _, action := range actions {
			target, err := ActionToLaunchTarget(consumer, host, info.installFolder, action)
			if err != nil {
				return nil, err
			}
			targets = append(targets, target)
			consumer.Logf(target.Strategy.String())
		}
	}

	if len(targets) > 0 {
		return targets, nil
	}

	consumer.Infof("Filtering verdict for host %v", host)
	filterParams := dash.FilterParams{
		OS: host.Runtime.OS(),
	}
	if info.runtime.Platform == host.Runtime.Platform {
		// if the platform we're getting targets for is
		// our currently running platform, we know the architecture,
		// so use it to filter.
		filterParams.Arch = info.runtime.Arch()
	}

	v2 := verdict.Filter(consumer, filterParams)
	verdict = &v2

	for _, candidate := range verdict.Candidates {
		target, err := CandidateToLaunchTarget(consumer, info.installFolder, host, candidate)
		if err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}

	return targets, nil
}
