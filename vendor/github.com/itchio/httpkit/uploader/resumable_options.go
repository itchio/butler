package uploader

type settings struct {
	MaxChunkGroup int
}

func defaultSettings() *settings {
	return &settings{
		// 64 * 256KiB = 16MiB
		MaxChunkGroup: 64,
	}
}

type Option interface {
	Apply(s *settings)
}

// ---------

type maxChunkGroupOption struct {
	maxChunkGroup int
}

// WithMaxChunkGroup specifies how many 256KiB chunks can be
// uploaded at a time. Raising this value may increase RAM
// usage, and cost more if we have to retry blocks.
//
// The default value is 64 (16MiB groups)
func WithMaxChunkGroup(maxChunkGroup int) *maxChunkGroupOption {
	return &maxChunkGroupOption{
		maxChunkGroup: maxChunkGroup,
	}
}

func (o *maxChunkGroupOption) Apply(s *settings) {
	s.MaxChunkGroup = o.maxChunkGroup
}
