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
	client, err := authenticateViaOauth()
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
			line := []string{ch.Name, buildState(ch.Head), buildParent(ch.Head), filesState(files)}
			table.Append(line)
		} else {
			line := []string{ch.Name, "No builds yet"}
			table.Append(line)
		}

		if ch.Pending != nil {
			files := ch.Pending.Files
			line := []string{"", buildState(ch.Pending), buildParent(ch.Pending), filesState(files)}
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
	}
	return fmt.Sprintf("#%d", build.ParentBuildID)
}

func filesState(files []*itchio.BuildFileInfo) string {
	if len(files) == 0 {
		return "(no files)"
	}

	s := ""
	for _, file := range files {
		if s != "" {
			s += ", "
		}
		s += fileState(file)
	}

	return s
}

func fileState(file *itchio.BuildFileInfo) string {
	theme := comm.GetTheme()

	fType := string(file.Type)
	if file.SubType != itchio.BuildFileType_ARCHIVE {
		fType += fmt.Sprintf(" (%s)", file.SubType)
	}

	sign := theme.StatSign
	if file.State != itchio.BuildFileState_UPLOADED {
		sign = theme.OpSign
	}

	fSize := humanize.Bytes(uint64(file.Size))

	return fmt.Sprintf("%s %s %s", sign, fSize, fType)
}
