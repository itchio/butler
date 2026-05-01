package push

import (
	"context"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/filtering"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/butler/walkutil"

	itchio "github.com/itchio/go-itchio"

	"github.com/itchio/lake/tlc"

	"github.com/pkg/errors"
)

var previewArgs = struct {
	src         string
	target      string
	changesOnly bool
	dereference bool
	fixPerms    bool
	autoWrap    bool
	autoUnzip   bool
}{}

// RegisterPreview wires up `butler push-preview`, a no-side-effects companion
// to `butler push` that reports per-file status (NEW/MODIFIED/SAME/DELETED)
// vs the previous build on the channel. No build is created, nothing is
// uploaded.
func RegisterPreview(ctx *mansion.Context) {
	cmd := ctx.App.Command("push-preview", "Compare a directory against a channel's previous build and report per-file status. Authenticated; doesn't modify anything on itch.io.")
	cmd.Arg("src", "Directory to compare. May also be a zip archive (slower).").Required().StringVar(&previewArgs.src)
	cmd.Arg("target", "Push target to compare against, e.g. 'leafo/x-moon:win-64'.").Required().StringVar(&previewArgs.target)
	cmd.Flag("changes-only", "Hide unchanged entries from the listing. Counts in the summary still cover every entry.").Default("false").BoolVar(&previewArgs.changesOnly)
	cmd.Flag("dereference", "Dereference symlinks").Default("false").BoolVar(&previewArgs.dereference)
	cmd.Flag("fix-permissions", "Detect Mac & Linux executables and adjust their permissions automatically").Default("true").BoolVar(&previewArgs.fixPerms)
	cmd.Flag("auto-wrap", "Apply workaround for https://github.com/itchio/itch/issues/2147").Default("true").BoolVar(&previewArgs.autoWrap)
	cmd.Flag("auto-unzip", "If src is a directory containing a single .zip file, compare the zip's contents instead of the zip-as-a-blob").Default("true").BoolVar(&previewArgs.autoUnzip)
	ctx.Register(cmd, doPreview)
}

func doPreview(ctx *mansion.Context) {
	ctx.Must(DoPreview(ctx, previewArgs.src, previewArgs.target, previewArgs.changesOnly, previewArgs.fixPerms, previewArgs.dereference, previewArgs.autoWrap, previewArgs.autoUnzip))
}

// DoPreview runs the comparison flow: walk the source, fetch the channel's
// previous-build signature, hash the source, classify per file, and print
// the result. Mirrors what cmd/push.Do does for a real push but without
// creating a build or uploading anything.
func DoPreview(ctx *mansion.Context, buildPath string, specStr string, changesOnly bool, fixPerms bool, dereference bool, wrap bool, autoUnzip bool) error {
	consumer := comm.NewStateConsumer()

	if autoUnzip {
		buildPath = walkutil.ResolveSingleZipDir(buildPath, filtering.FilterPaths)
	}

	sourceContainerChan := make(chan walkResult)
	walkErrs := make(chan error)
	walkOpts := tlc.WalkOpts{
		Filter:      filtering.FilterPaths,
		Dereference: dereference,
	}
	if wrap {
		walkOpts.AutoWrap(&buildPath, consumer)
	}

	go doWalk(buildPath, sourceContainerChan, walkErrs, fixPerms, walkOpts)

	spec, err := itchio.ParseSpec(specStr)
	if err != nil {
		return errors.Wrapf(err, "parsing target '%s'", specStr)
	}
	if err := spec.EnsureChannel(); err != nil {
		return err
	}

	client, err := ctx.AuthenticateViaOauth()
	if err != nil {
		return errors.Wrap(err, "authenticating")
	}

	requestCtx, cancel := ctx.DefaultCtx()
	chanInfo, err := client.GetChannel(requestCtx, spec.Target, spec.Channel)
	cancel()
	if err != nil {
		apiErr, ok := itchio.AsAPIError(err)
		if !ok || apiErr.StatusCode != 404 {
			return errors.Wrap(err, "getting channel")
		}

		// GetChannel returns 404 both for an absent channel and for some
		// invalid/inaccessible targets. ListChannels verifies the target is
		// reachable before we report a brand-new channel as "all entries new".
		requestCtx, cancel = ctx.DefaultCtx()
		channels, listErr := client.ListChannels(requestCtx, spec.Target)
		cancel()
		if listErr != nil {
			return errors.Wrap(err, "getting channel")
		}
		if channels != nil && channels.Channels != nil {
			if channel := channels.Channels[spec.Channel]; channel != nil {
				chanInfo = &itchio.GetChannelResponse{Channel: channel}
			}
		}
	}

	var walkies walkResult
	select {
	case walkErr := <-walkErrs:
		return errors.Wrap(walkErr, "walking directory to compare")
	case walkies = <-sourceContainerChan:
	}

	hasParent := chanInfo != nil && chanInfo.Channel != nil && chanInfo.Channel.Head != nil
	var parentID int64
	var result *comparisonResult

	if !hasParent {
		comm.Opf("No previous build on channel `%s`, all entries are new.", spec.Channel)
		result = allNewFromContainer(walkies.container)
	} else {
		parentID = chanInfo.Channel.Head.ID
		comm.Opf("Comparing against build %d on channel `%s`...", parentID, spec.Channel)
		targetSig, err := getSignature(ctx, client, consumer, parentID)
		if err != nil {
			return errors.Wrap(err, "getting previous build signature")
		}

		result, err = compareContainers(context.Background(), walkies.container, walkies.pool, targetSig, consumer)
		if err != nil {
			return errors.Wrap(err, "comparing containers")
		}
	}

	printComparison(result, changesOnly)

	if hasParent {
		comm.Statf("Comparison vs build %d: %d new, %d modified, %d deleted, %d unchanged",
			parentID, result.Counts.New, result.Counts.Modified, result.Counts.Deleted, result.Counts.Same)
	} else {
		comm.Statf("All %d entries are new (no previous build)", result.Counts.New)
	}

	previewResult(spec.Channel, hasParent, parentID, &result.Counts)
	return nil
}

func previewResult(channel string, hasParent bool, parentBuildID int64, comparison *pushComparisonCounts) {
	out := map[string]interface{}{
		"channel":    channel,
		"hasParent":  hasParent,
		"comparison": comparison,
	}
	if hasParent {
		out["parentBuildId"] = parentBuildID
	}
	comm.Result(out)
}
