package status

import (
	"fmt"
	"os"
	"sort"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/wharf/state"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
)

var args = struct {
	target       *string
	showAllFiles *bool
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("status", "Show a list of channels and the status of their latest and pending builds.")
	ctx.Register(cmd, do)

	args.target = cmd.Arg("target", "Which user/project to show the status of, for example 'leafo/x-moon'").Required().String()
	args.showAllFiles = cmd.Flag("show-all-files", "Show status of all files, not just archive").Bool()
}

func do(ctx *mansion.Context) {
	go ctx.DoVersionCheck()
	ctx.Must(Do(ctx, *args.target, *args.showAllFiles))
}

func Do(ctx *mansion.Context, specStr string, showAllFiles bool) error {
	spec, err := itchio.ParseSpec(specStr)
	if err != nil {
		return errors.Wrapf(err, "parsing spec %s", spec)
	}

	client, err := ctx.AuthenticateViaOauth()
	if err != nil {
		return errors.Wrap(err, "authenticating")
	}

	listChannelsResp, err := client.ListChannels(spec.Target)
	if err != nil {
		return errors.Wrap(err, "listing channels")
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Channel", "Upload", "Build", "Version"})

	found := false

	sortedChannelNames := []string{}
	for name := range listChannelsResp.Channels {
		sortedChannelNames = append(sortedChannelNames, name)
	}
	sort.Strings(sortedChannelNames)

	for _, channelName := range sortedChannelNames {
		ch := listChannelsResp.Channels[channelName]
		if spec.Channel != "" && ch.Name != spec.Channel {
			continue
		}
		found = true

		if ch.Head != nil {
			line := []string{ch.Name, fmt.Sprintf("#%d", ch.Upload.ID), buildState(ch.Head), versionState(ch.Head)}
			table.Append(line)
		} else {
			line := []string{ch.Name, fmt.Sprintf("#%d", ch.Upload.ID), "No builds yet"}
			table.Append(line)
		}

		if ch.Pending != nil {
			line := []string{"", "", buildState(ch.Pending), versionState(ch.Pending)}
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

func buildState(build *itchio.Build) string {
	theme := state.GetTheme()
	var s string

	switch build.State {
	case itchio.BuildStateCompleted:
		s = fmt.Sprintf("%s #%d", theme.StatSign, build.ID)
	case itchio.BuildStateProcessing:
		s = fmt.Sprintf("%s #%d", theme.OpSign, build.ID)
	default:
		s = fmt.Sprintf("  #%d (%s)", build.ID, build.State)
	}

	if build.ParentBuildID != -1 {
		s += fmt.Sprintf(" (from #%d)", build.ParentBuildID)
	}

	return s
}

func versionState(build *itchio.Build) string {
	switch build.State {
	case itchio.BuildStateCompleted:
		if build.UserVersion != "" {
			return build.UserVersion
		}

		return fmt.Sprintf("%d", build.Version)
	default:
		return ""
	}
}
