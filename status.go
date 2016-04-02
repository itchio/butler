package main

import (
	"fmt"
	"os"

	"github.com/dustin/go-humanize"
	"github.com/itchio/butler/comm"
	"github.com/itchio/go-itchio"
	"github.com/olekukonko/tablewriter"
)

func status(target string) {
	must(doStatus(target))
}

func doStatus(target string) error {
	client, err := authenticateViaWharf()
	if err != nil {
		return err
	}

	listChannelsResp, err := client.ListChannels(target)
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Channel", "Build", "Parent", "State"})

	for _, ch := range listChannelsResp.Channels {
		if ch.Head != nil {
			files := ch.Head.Files
			line := []string{ch.Name, buildState(ch.Head), buildParent(ch.Head), fileState(files.Patch, files.Archive)}
			table.Append(line)
		} else {
			line := []string{ch.Name, "No builds yet"}
			table.Append(line)
		}

		if ch.Pending != nil {
			files := ch.Pending.Files
			line := []string{"", buildState(ch.Pending), buildParent(ch.Pending), fileState(files.Patch, files.Archive)}
			table.Append(line)
		}
	}

	table.Render()

	return nil
}

func buildState(build *itchio.BuildInfo) string {
	theme := comm.GetTheme()

	switch build.State {
	case itchio.BuildState_COMPLETED:
		return fmt.Sprintf("%s #%d", theme.StatSign, build.ID)
	case itchio.BuildState_PROCESSING:
		return fmt.Sprintf("%s #%d", theme.OpSign, build.ID)
	default:
		return fmt.Sprintf("  #%d (%s)", build.ID, build.State)
	}
}

func buildParent(build *itchio.BuildInfo) string {
	if build.ParentBuildID == -1 {
		return ""
	} else {
		return fmt.Sprintf("#%d", build.ParentBuildID)
	}
}

func fileState(patch *itchio.BuildFileInfo, archive *itchio.BuildFileInfo) string {
	theme := comm.GetTheme()

	if archive.State == itchio.BuildFileState_UPLOADED {
		if patch.State == itchio.BuildFileState_UPLOADED {
			return fmt.Sprintf("%s %s archive, %s patch", theme.StatSign, humanize.Bytes(uint64(archive.Size)), humanize.Bytes(uint64(patch.Size)))
		} else {
			return fmt.Sprintf("%s %s archive, processing patch", theme.OpSign, humanize.Bytes(uint64(archive.Size)))
		}
	} else {
		if patch.State == itchio.BuildFileState_UPLOADED {
			return fmt.Sprintf("%s %s patch, processing archive", theme.OpSign, humanize.Bytes(uint64(patch.Size)))
		} else {
			return fmt.Sprintf("uploading...", theme.OpSign)
		}
	}
}
