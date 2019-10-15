package fetch

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/itchio/boar"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	itchio "github.com/itchio/go-itchio"
	"github.com/pkg/errors"
)

var args = struct {
	target *string
	out    *string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("fetch", "Download and extract the latest build of a channel from itch.io")
	ctx.Register(cmd, do)

	args.target = cmd.Arg("target", "Which user/project:channel to fetch from, for example 'leafo/x-moon:win-64'. Targets are of the form project:channel where project is username/game or game_id.").Required().String()
	args.out = cmd.Arg("out", "Directory to fetch and extract build to").Required().String()
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(ctx, *args.target, *args.out))
}

func Do(ctx *mansion.Context, specStr string, outPath string) error {
	consumer := comm.NewStateConsumer()

	err := os.MkdirAll(outPath, os.FileMode(0o755))
	if err != nil {
		return errors.WithStack(err)
	}

	outFiles, err := ioutil.ReadDir(outPath)
	if err != nil {
		return errors.WithStack(err)
	}

	if len(outFiles) > 0 {
		return fmt.Errorf("Destination directory %s exists and is not empty", outPath)
	}

	spec, err := itchio.ParseSpec(specStr)
	if err != nil {
		return err
	}

	err = spec.EnsureChannel()
	if err != nil {
		return err
	}

	client, err := ctx.AuthenticateViaOauth()
	if err != nil {
		return err
	}

	comm.Opf("Getting last build of channel %s", spec.Channel)

	channelResponse, err := client.GetChannel(ctx.DefaultCtx(), spec.Target, spec.Channel)
	if err != nil {
		return err
	}

	if channelResponse.Channel.Head == nil {
		return fmt.Errorf("Channel %s doesn't have any builds yet", spec.Channel)
	}

	buildID := channelResponse.Channel.Head.ID

	buildFilesRes, err := client.ListBuildFiles(ctx.DefaultCtx(), buildID)
	if err != nil {
		return err
	}

	archiveFile := itchio.FindBuildFileEx(itchio.BuildFileTypeArchive, itchio.BuildFileSubTypeDefault, buildFilesRes.Files)
	if archiveFile == nil {
		return fmt.Errorf("Channel %s's latest build is still processing", spec.Channel)
	}

	url := client.MakeBuildFileDownloadURL(itchio.MakeBuildFileDownloadURLParams{
		BuildID: buildID,
		FileID:  archiveFile.ID,
	})

	comm.Opf("Extracting into %s", outPath)

	comm.StartProgress()
	extractRes, err := boar.SimpleExtract(&boar.SimpleExtractParams{
		ArchivePath:       url,
		Consumer:          consumer,
		DestinationFolder: outPath,
	})
	comm.EndProgress()
	if err != nil {
		return err
	}
	comm.Statf("Extracted %s", extractRes.Stats())

	return nil
}
