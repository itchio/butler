package progress

import (
	"time"

	"github.com/itchio/butler/pb"
	"github.com/itchio/httpkit/timeout"
)

var maxBucketDuration = 1 * time.Second

type Counter struct {
	lastBandwidthUpdate time.Time
	lastBandwidthAlpha  float64
	bps                 float64
	bar                 *pb.ProgressBar
	alpha               float64
}

func NewCounter() *Counter {
	bar := pb.New64(100 * 100)
	bar.AlwaysUpdate = true
	bar.RefreshRate = 125 * time.Millisecond

	return &Counter{
		// show to the 1/100ths of a percent (1/10000th of an alpha)
		bar: bar,
	}
}

func (c *Counter) SetTotalBytes(totalBytes int64) {
	c.bar.TotalBytes = totalBytes
}

func (c *Counter) SetSilent(silent bool) {
	c.bar.NotPrint = silent
}

func (c *Counter) Start() {
	c.bar.Start()
}

func (c *Counter) Finish() {
	c.bar.Postfix("")
	c.bar.Finish()
}

func (c *Counter) Pause() {
	c.bar.AlwaysUpdate = false
}

func (c *Counter) Resume() {
	c.bar.AlwaysUpdate = true
}

func (c *Counter) SetProgress(alpha float64) {
	if c.bar.TotalBytes != 0 {
		if c.lastBandwidthUpdate.IsZero() {
			c.lastBandwidthUpdate = time.Now()
			c.lastBandwidthAlpha = alpha
		}
		bucketDuration := time.Since(c.lastBandwidthUpdate)

		if bucketDuration > maxBucketDuration {
			bytesSinceLastUpdate := float64(c.bar.TotalBytes) * (alpha - c.lastBandwidthAlpha)
			c.bps = bytesSinceLastUpdate / bucketDuration.Seconds()
			c.lastBandwidthUpdate = time.Now()
			c.lastBandwidthAlpha = alpha
		}
		// otherwise, keep current bps value
	} else {
		c.bps = 0
	}

	c.alpha = alpha
	c.bar.Set64(alphaToValue(alpha))
}

func (c *Counter) Progress() float64 {
	return c.alpha
}

func (c *Counter) ETA() time.Duration {
	return c.bar.TimeLeft
}

func (c *Counter) BPS() float64 {
	return timeout.GetBPS()
}

func (c *Counter) WorkBPS() float64 {
	return c.bps
}

func (c *Counter) Bar() *pb.ProgressBar {
	return c.bar
}

func alphaToValue(alpha float64) int64 {
	return int64(alpha * 100.0 * 100.0)
}
