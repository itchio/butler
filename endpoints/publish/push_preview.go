package publish

import (
	"fmt"
	"strconv"

	"github.com/itchio/butler/butlerd"
)

// PushPreview spawns a `butler push --dry-run --compare` worker subprocess
// to report what would change if Src were pushed to the channel. No build
// is created and no data is uploaded.
func PushPreview(rc *butlerd.RequestContext, params butlerd.PublishPushPreviewParams) (*butlerd.PublishPushPreviewResult, error) {
	args := buildPushPreviewArgs(params)
	result, err := runPushWorker(rc, params.ProfileID, args, "")
	if err != nil {
		return nil, err
	}

	channel := result.Channel
	if channel == "" {
		channel = params.Channel
	}

	res := &butlerd.PublishPushPreviewResult{
		Channel:         channel,
		HasParent:       result.HasParent,
		ParentBuildID:   result.ParentBuildID,
		SourceSize:      result.SourceSize,
		TopChangedFiles: result.TopChangedFiles,
	}
	// Promise the client non-nil arrays for every category — matches the
	// JSON-side contract from cmd/push/preview.go and lets the renderer
	// treat an empty list as "no changes of this kind" without nil checks.
	if res.TopChangedFiles.New == nil {
		res.TopChangedFiles.New = []butlerd.PublishPushPreviewEntry{}
	}
	if res.TopChangedFiles.Modified == nil {
		res.TopChangedFiles.Modified = []butlerd.PublishPushPreviewEntry{}
	}
	if res.TopChangedFiles.Deleted == nil {
		res.TopChangedFiles.Deleted = []butlerd.PublishPushPreviewEntry{}
	}
	if result.Comparison != nil {
		res.Comparison = *result.Comparison
	}
	return res, nil
}

func buildPushPreviewArgs(p butlerd.PublishPushPreviewParams) []string {
	specStr := fmt.Sprintf("%s:%s", p.Target, p.Channel)
	args := []string{"push-preview", p.Src, specStr, "--json", "--changes-only"}

	if p.Dereference {
		args = append(args, "--dereference")
	}
	if p.FixPermissions != nil {
		args = append(args, "--fix-permissions="+strconv.FormatBool(*p.FixPermissions))
	}
	if p.AutoWrap != nil {
		args = append(args, "--auto-wrap="+strconv.FormatBool(*p.AutoWrap))
	}
	return args
}
