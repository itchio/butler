package rate

import "time"

// A Limiter ensures we don't go over a certain number
// of requests per second.
type Limiter interface {
	// Wait must be called before performing any rate-limited action.
	// It doesn't return an error and never panics.
	Wait()
}

type limiter struct {
	c        chan struct{}
	interval time.Duration
}

// LimiterOpts specifies how many requests per
// second our limiter should allow, and how many "burst"
// requests are allowed.
type LimiterOpts struct {
	RequestsPerSecond int
	Burst             int
}

// NewLimiter returns a new Limiter configured
// to only allow rps requests per seconds.
func NewLimiter(opts LimiterOpts) Limiter {
	if opts.RequestsPerSecond <= 0 {
		panic("RequestsPerSecond must be > 0")
	}

	c := make(chan struct{}, opts.Burst)
	l := &limiter{
		c:        c,
		interval: time.Duration(1.0 / float64(opts.RequestsPerSecond) * float64(time.Second)),
	}
	go l.run()
	return l
}

func (l *limiter) run() {
	for {
		time.Sleep(l.interval)
		l.c <- struct{}{}
	}
}

func (l *limiter) Wait() {
	<-l.c
}
