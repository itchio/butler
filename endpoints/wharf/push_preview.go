package wharf

import (
	"fmt"
	"strconv"

	"github.com/itchio/butler/butlerd"
)

// PushPreview spawns a `butler push --dry-run --compare` worker subprocess
// to report what would change if Src were pushed to the channel. No build
// is created and no data is uploaded.
func PushPreview(rc *butlerd.RequestContext, params butlerd.WharfPushPreviewParams) (*butlerd.WharfPushPreviewResult, error) {
	args := buildPushPreviewArgs(params)
	result, err := runPushWorker(rc, params.ProfileID, args, "")
	if err != nil {
		return nil, err
	}

	channel := result.Channel
	if channel == "" {
		channel = params.Channel
	}

	res := &butlerd.WharfPushPreviewResult{
		Channel:         channel,
		HasParent:       result.HasParent,
		ParentBuildID:   result.ParentBuildID,
		SourceSize:      result.SourceSize,
		TopChangedFiles: result.TopChangedFiles,
	}
	if res.TopChangedFiles == nil {
		// Promise the client a non-nil array even when the worker emits
		// nothing — matches the JSON-side contract from cmd/push/preview.go.
		res.TopChangedFiles = []butlerd.WharfPushPreviewEntry{}
	}
	if result.Comparison != nil {
		res.Comparison = *result.Comparison
	}
	return res, nil
}

func buildPushPreviewArgs(p butlerd.WharfPushPreviewParams) []string {
	specStr := fmt.Sprintf("%s:%s", p.Target, p.Channel)
	args := []string{"push-preview", p.Src, specStr, "--json"}

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
