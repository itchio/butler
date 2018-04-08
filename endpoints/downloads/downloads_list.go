package downloads

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
)

func DownloadsList(rc *butlerd.RequestContext, params *butlerd.DownloadsListParams) (*butlerd.DownloadsListResult, error) {
	downloads := models.AllDownloads(rc.DB())
	models.PreloadDownloads(rc.DB(), downloads)

	var fdls []*butlerd.Download
	for _, d := range downloads {
		fdls = append(fdls, formatDownload(d))
	}

	res := &butlerd.DownloadsListResult{
		Downloads: fdls,
	}
	return res, nil
}

func formatDownload(download *models.Download) *butlerd.Download {
	return &butlerd.Download{
		ID:            download.ID,
		Error:         download.Error,
		ErrorMessage:  download.ErrorMessage,
		ErrorCode:     download.ErrorCode,
		Position:      download.Position,
		CaveID:        download.CaveID,
		Game:          download.Game,
		Upload:        download.Upload,
		Build:         download.Build,
		StartedAt:     download.StartedAt,
		FinishedAt:    download.FinishedAt,
		StagingFolder: download.StagingFolder,
		Reason:        butlerd.DownloadReason(download.Reason),
	}
}
