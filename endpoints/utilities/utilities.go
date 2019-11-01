package utilities

import (
	"github.com/efarrer/iothrottler"
	"github.com/itchio/butler/buildinfo"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/httpkit/timeout"
)

func Register(router *butlerd.Router) {
	messages.VersionGet.Register(router, func(rc *butlerd.RequestContext, params butlerd.VersionGetParams) (*butlerd.VersionGetResult, error) {
		return &butlerd.VersionGetResult{
			Version:       buildinfo.Version,
			VersionString: buildinfo.VersionString,
		}, nil
	})

	messages.NetworkSetSimulateOffline.Register(router, func(rc *butlerd.RequestContext, params butlerd.NetworkSetSimulateOfflineParams) (*butlerd.NetworkSetSimulateOfflineResult, error) {
		rc.Consumer.Infof("Setting offline mode to: %v", params.Enabled)
		timeout.SetSimulateOffline(params.Enabled)

		if params.Enabled {
			// with http/2, we need to do this, otherwise it'll re-use existing connections
			rc.Consumer.Infof("Closing idle connections")
			rc.HTTPTransport.CloseIdleConnections()
		}

		res := &butlerd.NetworkSetSimulateOfflineResult{}
		return res, nil
	})

	messages.NetworkSetBandwidthThrottle.Register(router, func(rc *butlerd.RequestContext, params butlerd.NetworkSetBandwidthThrottleParams) (*butlerd.NetworkSetBandwidthThrottleResult, error) {
		if params.Enabled {
			timeout.ThrottlerPool.SetBandwidth(iothrottler.Bandwidth(params.Rate) * iothrottler.Kbps)
		} else {
			timeout.ThrottlerPool.SetBandwidth(iothrottler.Unlimited)
		}
		res := &butlerd.NetworkSetBandwidthThrottleResult{}
		return res, nil
	})
}
