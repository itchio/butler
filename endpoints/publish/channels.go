package publish

import (
	"github.com/itchio/butler/butlerd"
	itchio "github.com/itchio/go-itchio"
	"github.com/pkg/errors"
)

func ListChannels(rc *butlerd.RequestContext, params butlerd.PublishListChannelsParams) (*butlerd.PublishListChannelsResult, error) {
	_, client := rc.ProfileClient(params.ProfileID)

	res, err := client.ListChannels(rc.Ctx, params.Target)
	if err != nil {
		return nil, errors.Wrap(err, "listing channels")
	}

	out := make(map[string]*butlerd.PublishChannel, len(res.Channels))
	for k, ch := range res.Channels {
		out[k] = toPublishChannel(ch)
	}
	return &butlerd.PublishListChannelsResult{
		Channels: out,
	}, nil
}

func GetChannel(rc *butlerd.RequestContext, params butlerd.PublishGetChannelParams) (*butlerd.PublishGetChannelResult, error) {
	_, client := rc.ProfileClient(params.ProfileID)

	res, err := client.GetChannel(rc.Ctx, params.Target, params.Channel)
	if err != nil {
		return nil, errors.Wrap(err, "getting channel")
	}

	return &butlerd.PublishGetChannelResult{
		Channel: toPublishChannel(res.Channel),
	}, nil
}

func toPublishChannel(c *itchio.Channel) *butlerd.PublishChannel {
	if c == nil {
		return nil
	}
	return &butlerd.PublishChannel{
		Name:    c.Name,
		Tags:    c.Tags,
		Upload:  c.Upload,
		Head:    c.Head,
		Pending: c.Pending,
	}
}
