package operate

import (
	"fmt"
	"net/url"
	"path/filepath"

	humanize "github.com/dustin/go-humanize"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/installer/bfs"

	"github.com/itchio/butler/installer"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/cmd/cp"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/manager"
	itchio "github.com/itchio/go-itchio"
)

func install(oc *OperationContext, meta *MetaSubcontext) (*installer.InstallResult, error) {
	consumer := oc.Consumer()

	params := meta.data.InstallParams

	if params == nil {
		return nil, errors.New("Missing install params")
	}

	if params.Game == nil {
		return nil, errors.New("Missing game in install")
	}

	if params.InstallFolder == "" {
		return nil, errors.New("Missing install folder in install")
	}

	consumer.Infof("Installing game %s", params.Game.Title)
	consumer.Infof("...into directory %s", params.InstallFolder)
	consumer.Infof("...using stage directory %s", oc.StageFolder())

	client, err := clientFromCredentials(params.Credentials)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	// TODO: cache that in context

	if params.Upload == nil {
		consumer.Infof("No upload specified, looking for compatible ones...")
		uploads, err := client.ListGameUploads(&itchio.ListGameUploadsParams{
			GameID:        params.Game.ID,
			DownloadKeyID: params.Credentials.DownloadKey,
		})
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		consumer.Infof("Filtering %d uploads", len(uploads.Uploads))

		uploadsFilterResult := manager.NarrowDownUploads(uploads.Uploads, params.Game, manager.CurrentRuntime())
		consumer.Infof("After filter, got %d uploads, they are: ", len(uploadsFilterResult.Uploads))
		for _, upload := range uploadsFilterResult.Uploads {
			consumer.Infof("- %#v", upload)
		}

		if len(uploadsFilterResult.Uploads) == 0 {
			consumer.Warnf("Didn't find a compatible upload. The initial uploads were:", len(uploads.Uploads))
			for _, upload := range uploads.Uploads {
				consumer.Infof("- %#v", upload)
			}

			return nil, (&OperationError{
				Code:      "noCompatibleUploads",
				Message:   "No compatible uploads",
				Operation: "install",
			}).Throw()
		}

		if len(uploadsFilterResult.Uploads) == 1 {
			params.Upload = uploadsFilterResult.Uploads[0]
		} else {
			var r buse.PickUploadResult
			err := oc.conn.Call(oc.ctx, "pick-upload", &buse.PickUploadParams{
				Uploads: uploadsFilterResult.Uploads,
			}, &r)
			if err != nil {
				return nil, errors.Wrap(err, 0)
			}

			params.Upload = uploadsFilterResult.Uploads[r.Index]
		}
	}

	// TODO: if upload is wharf-enabled, retrieve build & include it in context/receipt/result etc.

	var archiveUrlPath string
	if params.Build == nil {
		archiveUrlPath = fmt.Sprintf("/upload/%d/download", params.Upload.ID)
	} else {
		archiveUrlPath = fmt.Sprintf("/upload/%d/download/builds/%d/archive", params.Upload.ID, params.Build.ID)
	}
	values := make(url.Values)
	values.Set("api_key", params.Credentials.APIKey)
	if params.Credentials.DownloadKey != 0 {
		values.Set("download_key_id", fmt.Sprintf("%d", params.Credentials.DownloadKey))
	}
	var archiveUrl = fmt.Sprintf("itchfs://%s?%s", archiveUrlPath, values.Encode())

	// use natural file name for non-wharf downloads
	var archiveDownloadName = params.Upload.Filename // TODO: cache that in context
	if params.Build != nil {
		// make up a sensible .zip name for wharf downloads
		archiveDownloadName = fmt.Sprintf("%d-%d.zip", params.Upload.ID, params.Build.ID)
	}

	var archiveDownloadPath = filepath.Join(oc.StageFolder(), archiveDownloadName)
	copyParams := &cp.CopyParams{
		Consumer: consumer,
		OnStart: func(initialProgress float64, totalBytes int64) {
			// TODO: send requests to client letting it know we're downloading
			// something
			consumer.Infof("Download started, %s to fetch", humanize.IBytes(uint64(totalBytes)))
		},
		OnStop: func() {
			consumer.Infof("Download ended")
		},
	}
	err = cp.Do(oc.MansionContext(), copyParams, archiveUrl, archiveDownloadPath, true)
	// TODO: cache copy result in context
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	installerInfo, err := getInstallerInfo(archiveDownloadPath)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	// TODO: cache get installer info result in context
	consumer.Infof("Will use installer %s", installerInfo.Type)
	manager := installer.GetManager(string(installerInfo.Type))
	if manager == nil {
		msg := fmt.Sprintf("No manager for installer %s", installerInfo.Type)
		return nil, errors.New(msg)
	}

	receiptIn, err := bfs.ReadReceipt(params.InstallFolder)
	if err != nil {
		receiptIn = nil
		consumer.Warnf("Could not read existing receipt: %s", err.Error())
	}

	comm.StartProgress()
	res, err := manager.Install(&installer.InstallParams{
		Consumer:          oc.Consumer(),
		ArchiveListResult: installerInfo.ArchiveListResult,

		SourcePath:        archiveDownloadPath,
		InstallFolderPath: params.InstallFolder,

		ReceiptIn: receiptIn,
	})
	comm.EndProgress()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	consumer.Infof("Install successful, writing receipt")
	receipt := &bfs.Receipt{
		Files:         res.Files,
		InstallerName: string(installerInfo.Type),
		Game:          params.Game,
		Upload:        params.Upload,
		Build:         params.Build,
	}

	err = receipt.WriteReceipt(params.InstallFolder)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return res, nil
}
