package wharf

import (
	"github.com/itchio/butler/butlerd"
	itchio "github.com/itchio/go-itchio"
	"github.com/pkg/errors"
)

func ListChannels(rc *butlerd.RequestContext, params butlerd.WharfListChannelsParams) (*butlerd.WharfListChannelsResult, error) {
	_, client := rc.ProfileClient(params.ProfileID)

	res, err := client.ListChannels(rc.Ctx, params.Target)
	if err != nil {
		return nil, errors.Wrap(err, "listing channels")
	}

	out := make(map[string]*butlerd.WharfChannel, len(res.Channels))
	for k, ch := range res.Channels {
		out[k] = toWharfChannel(ch)
	}
	return &butlerd.WharfListChannelsResult{
		Channels: out,
	}, nil
}

func GetChannel(rc *butlerd.RequestContext, params butlerd.WharfGetChannelParams) (*butlerd.WharfGetChannelResult, error) {
	_, client := rc.ProfileClient(params.ProfileID)

	res, err := client.GetChannel(rc.Ctx, params.Target, params.Channel)
	if err != nil {
		return nil, errors.Wrap(err, "getting channel")
	}

	return &butlerd.WharfGetChannelResult{
		Channel: toWharfChannel(res.Channel),
	}, nil
}

func toWharfChannel(c *itchio.Channel) *butlerd.WharfChannel {
	if c == nil {
		return nil
	}
	return &butlerd.WharfChannel{
		Name:    c.Name,
		Tags:    c.Tags,
		Upload:  c.Upload,
		Head:    c.Head,
		Pending: c.Pending,
	}
}
