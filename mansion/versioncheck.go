package mansion

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime"
	"strings"

	"github.com/itchio/butler/comm"
	itchio "github.com/itchio/go-itchio"
)

func (ctx *Context) UpdateBaseURL(variant VersionVariant) string {
	channel := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	if variant == VersionVariantHead {
		channel += "-head"
	}
	return fmt.Sprintf("https://broth.itch.ovh/butler/%s", channel)
}

func (ctx *Context) DoVersionCheck() {
	vinfo, err := ctx.QueryLatestVersion(ctx.CurrentVariant())
	if err != nil {
		comm.Logf("Version check failed: %s", err.Error())
		return
	}

	if vinfo.Current == nil || vinfo.Latest == nil {
		return
	}

	if vinfo.Current.Equal(vinfo.Latest) {
		return
	}

	if vinfo.Current.Name == "" {
		// home-built, don't check
		return
	}

	if vinfo.Current != vinfo.Latest {
		comm.Notice("New version available",
			[]string{
				fmt.Sprintf("Current version: %s", vinfo.Current),
				fmt.Sprintf("Latest version:  %s", vinfo.Latest),
				"",
				"Run `butler upgrade` to get it.",
			})
	}
}

type Version struct {
	Name    string
	Variant VersionVariant
}

func (v *Version) String() string {
	return fmt.Sprintf("%s (%s)", v.Name, v.Variant)
}

func (v *Version) Equal(v2 *Version) bool {
	if v == nil {
		return false
	}
	if v2 == nil {
		return false
	}
	if v.Variant != v2.Variant {
		return false
	}
	if v.Name != v2.Name {
		return false
	}
	return true
}

func normalizeVersionName(name string) string {
	return strings.TrimLeft(name, "v")
}

type VersionCheckResult struct {
	Current *Version
	Latest  *Version
}

type VersionVariant string

const (
	VersionVariantStable = "stable"
	VersionVariantHead   = "head"
)

func (ctx *Context) CurrentVersion() *Version {
	variant := ctx.CurrentVariant()
	switch variant {
	case VersionVariantHead:
		return &Version{
			Name:    ctx.Commit,
			Variant: variant,
		}
	default:
		return &Version{
			Name:    normalizeVersionName(ctx.Version),
			Variant: variant,
		}
	}
}

func (ctx *Context) CurrentVariant() VersionVariant {
	switch ctx.Version {
	case "head":
		return VersionVariantHead
	default:
		return VersionVariantStable
	}
}

func (ctx *Context) QueryLatestVersion(variant VersionVariant) (*VersionCheckResult, error) {
	if ctx.Quiet {
		return nil, nil
	}

	res := &VersionCheckResult{
		Current: ctx.CurrentVersion(),
	}

	c := itchio.ClientWithKey("x")

	latestURL := fmt.Sprintf("%s/LATEST", ctx.UpdateBaseURL(variant))
	req, err := http.NewRequest("GET", latestURL, nil)
	if err != nil {
		return nil, err
	}

	latestRes, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	if latestRes.StatusCode != 200 {
		err = fmt.Errorf("HTTP %d: %s", latestRes.StatusCode, latestURL)
		return nil, err
	}

	buf, err := ioutil.ReadAll(latestRes.Body)
	if err != nil {
		return nil, err
	}

	latestVersion := strings.Trim(string(buf), " \r\n")
	res.Latest = &Version{
		Name:    normalizeVersionName(latestVersion),
		Variant: variant,
	}

	return res, nil
}
