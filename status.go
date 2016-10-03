package main

import (
	"fmt"
	"os"

	"github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/go-itchio"
	"github.com/olekukonko/tablewriter"
)

func status(specStr string) {
	go versionCheck()
	must(doStatus(specStr))
}

func doStatus(specStr string) error {
	spec, err := itchio.ParseSpec(specStr)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	client, err := authenticateViaOauth()
	if err != nil {
		return errors.Wrap(err, 1)
	}

	listChannelsResp, err := client.ListChannels(spec.Target)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Channel", "Upload", "Build", "State"})

	found := false

	for _, ch := range listChannelsResp.Channels {
		if spec.Channel != "" && ch.Name != spec.Channel {
			continue
		}
		found = true

		if ch.Head != nil {
			files := ch.Head.Files
			line := []string{ch.Name, fmt.Sprintf("#%d", ch.Upload.ID), buildState(ch.Head), filesState(files)}
			table.Append(line)
		} else {
			line := []string{ch.Name, fmt.Sprintf("#%d", ch.Upload.ID), "No builds yet"}
			table.Append(line)
		}

		if ch.Pending != nil {
			files := ch.Pending.Files
			line := []string{"", "", buildState(ch.Pending), filesState(files)}
			table.Append(line)
		}
	}

	if found {
		table.Render()
	} else {
		comm.Logf("No channel %s found for %s", spec.Channel, spec.Target)
	}

	return nil
}

func buildState(build *itchio.BuildInfo) string {
	theme := comm.GetTheme()
	var s string

	switch build.State {
	case itchio.BuildState_COMPLETED:
		s = fmt.Sprintf("%s #%d", theme.StatSign, build.ID)
	case itchio.BuildState_PROCESSING:
		s = fmt.Sprintf("%s #%d", theme.OpSign, build.ID)
	default:
		s = fmt.Sprintf("  #%d (%s)", build.ID, build.State)
	}

	if build.ParentBuildID != -1 {
		s += fmt.Sprintf(" (from #%d)", build.ParentBuildID)
	}

	return s
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
	if file.SubType != itchio.BuildFileSubType_DEFAULT {
		fType += fmt.Sprintf(" (%s)", file.SubType)
	}

	sign := theme.StatSign
	if file.State != itchio.BuildFileState_UPLOADED {
		sign = theme.OpSign
	}

	fSize := humanize.IBytes(uint64(file.Size))

	return fmt.Sprintf("%s %s %s", sign, fSize, fType)
}
