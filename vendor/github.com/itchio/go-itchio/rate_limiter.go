package itchio

import "github.com/itchio/httpkit/rate"

var defaultRateLimiter rate.Limiter

// DefaultRateLimiter returns a rate.Limiter suitable
// for consuming the itch.io API. It is shared across all
// instances of Client, unless a custom limiter is set.
func DefaultRateLimiter() rate.Limiter {
	if defaultRateLimiter == nil {
		defaultRateLimiter = rate.NewLimiter(rate.LimiterOpts{
			RequestsPerSecond: 8,
			Burst:             20,
		})
	}
	return defaultRateLimiter
}
