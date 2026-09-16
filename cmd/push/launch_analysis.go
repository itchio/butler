package push

import (
	"encoding/json"

	"github.com/itchio/butler/buildinfo"
	"github.com/itchio/dash"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/headway/state"
	"github.com/itchio/lake"
	"github.com/itchio/lake/pools"
	"github.com/itchio/lake/tlc"
)

// scanBuildLaunchAnalysis builds the launch report push sends with
// CreateBuild. It opens its own pool so the scan's zip spool is released
// before the upload starts.
func scanBuildLaunchAnalysis(buildPath string, container *tlc.Container, consumer *state.Consumer) *itchio.BuildLaunchAnalysis {
	pool, err := pools.New(container, buildPath)
	if err != nil {
		consumer.Warnf("Could not open launch scan pool; continuing upload without launch analysis: %v", err)
		return nil
	}
	defer func() {
		if err := pool.Close(); err != nil {
			consumer.Warnf("Could not close launch scan pool: %v", err)
		}
	}()
	return scanLaunchAnalysis(container, pool, consumer)
}

// scanLaunchAnalysis takes the upload's container so ignored files and
// wrapping match the build.
func scanLaunchAnalysis(container *tlc.Container, pool lake.Pool, consumer *state.Consumer) *itchio.BuildLaunchAnalysis {
	targets, err := dash.ScanLaunchTargetsContainer(container, pool, dash.ConfigureParams{Consumer: consumer})
	if err != nil {
		consumer.Warnf("Could not scan launch targets; continuing upload without launch analysis: %v", err)
		return nil
	}
	data, err := json.Marshal(targets)
	if err != nil {
		consumer.Warnf("Could not encode launch targets; continuing upload without launch analysis: %v", err)
		return nil
	}
	version := buildinfo.Version
	if version == "head" && buildinfo.Commit != "" {
		version = buildinfo.Commit
	}
	return &itchio.BuildLaunchAnalysis{
		SchemaVersion:  dash.LaunchTargetsSchemaVersion,
		ScannerVersion: "butler/" + version,
		LaunchTargets:  data,
	}
}
