package mansion

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime"
	"strings"

	"github.com/blang/semver"
	"github.com/itchio/butler/comm"
	itchio "github.com/itchio/go-itchio"
)

func (ctx *Context) UpdateBaseURL() string {
	return fmt.Sprintf("https://dl.itch.ovh/butler/%s-%s", runtime.GOOS, runtime.GOARCH)
}

func (ctx *Context) DoVersionCheck() {
	currentVer, latestVer, err := ctx.QueryLatestVersion()
	if err != nil {
		comm.Logf("Version check failed: %s", err.Error())
	}

	if currentVer == nil || latestVer == nil {
		return
	}

	if latestVer.GT(*currentVer) {
		comm.Notice("New version available",
			[]string{
				fmt.Sprintf("Current version: %s", currentVer),
				fmt.Sprintf("Latest version:  %s", latestVer),
				"",
				"Run `butler upgrade` to get it.",
			})
	}
}

func parseSemver(s string) (semver.Version, error) {
	return semver.Make(strings.TrimLeft(s, "v"))
}

func (ctx *Context) QueryLatestVersion() (*semver.Version, *semver.Version, error) {
	if ctx.Quiet {
		return nil, nil, nil
	}

	if ctx.Version == "head" {
		return nil, nil, nil
	}

	currentVer, err := parseSemver(ctx.Version)
	if err != nil {
		return nil, nil, err
	}

	c := itchio.ClientWithKey("x")

	latestURL := fmt.Sprintf("%s/LATEST", ctx.UpdateBaseURL())
	req, err := http.NewRequest("GET", latestURL, nil)
	if err != nil {
		return nil, nil, err
	}

	res, err := c.Do(req)
	if err != nil {
		return nil, nil, err
	}

	if res.StatusCode != 200 {
		err = fmt.Errorf("HTTP %d: %s", res.StatusCode, latestURL)
		return nil, nil, err
	}

	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, nil, err
	}

	latestVersion := strings.Trim(string(buf), " \r\n")
	latestVer, err := parseSemver(latestVersion)
	if err != nil {
		return nil, nil, err
	}

	return &currentVer, &latestVer, nil
}
