package downloads

import (
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/database/models"
)

func DownloadsList(rc *buse.RequestContext, params *buse.DownloadsListParams) (*buse.DownloadsListResult, error) {
	downloads := models.AllDownloads(rc.DB())
	models.PreloadDownloads(rc.DB(), downloads)

	var fdls []*buse.Download
	for _, d := range downloads {
		fdls = append(fdls, formatDownload(d))
	}

	res := &buse.DownloadsListResult{
		Downloads: fdls,
	}
	return res, nil
}

func formatDownload(download *models.Download) *buse.Download {
	return &buse.Download{
		ID:            download.ID,
		Position:      download.Position,
		CaveID:        download.CaveID,
		Game:          download.Game,
		Upload:        download.Upload,
		Build:         download.Build,
		StartedAt:     download.StartedAt,
		FinishedAt:    download.FinishedAt,
		StagingFolder: download.StagingFolder,
	}
}
