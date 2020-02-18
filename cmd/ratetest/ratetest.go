package ratetest

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"crawshaw.io/sqlite/sqlitex"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"github.com/pkg/errors"
	"golang.org/x/time/rate"
	"xorm.io/builder"
)

var args = struct {
	limit    float64
	burst    int
	workers  int
	requests int
}{
	limit: -1,
	burst: -1,
}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("ratetest", "(Internal) Perform a rate limiting test").Hidden()
	cmd.Flag("limit", "Max numbers of requests per second").Float64Var(&args.limit)
	cmd.Flag("burst", "Number of burst requests allowed").IntVar(&args.burst)
	cmd.Flag("workers", "Number of workers").Default("4").IntVar(&args.workers)
	cmd.Flag("requests", "Number of requests").Default("8").IntVar(&args.requests)
	ctx.Register(cmd, func(ctx *mansion.Context) {
		ctx.Must(do(ctx))
	})
}

func do(mc *mansion.Context) error {
	consumer := comm.NewStateConsumer()
	if mc.DBPath == "" {
		consumer.Debugf("DB path not specified (--dbpath), guessing...")
		mc.DBPath = butlerd.GuessDBPath("")
	}
	consumer.Debugf("Using database (%s)", mc.DBPath)

	dbPool, err := sqlitex.Open(mc.DBPath, 0, 1)
	if err != nil {
		mc.Must(errors.WithMessage(err, "opening DB for the first time"))
	}
	defer dbPool.Close()

	ctx := context.Background()

	conn := dbPool.Get(ctx)
	if conn == nil {
		panic("database busy")
	}

	var profiles []*models.Profile
	models.MustSelect(conn, &profiles, builder.NewCond(), hades.Search{})
	models.MustPreload(conn, profiles, hades.Assoc("User"))

	if len(profiles) == 0 {
		panic("no profiles remembered, can't run rate test")
	}

	profile := profiles[0]

	defaultLimiter := itchio.DefaultRateLimiter()
	limit := defaultLimiter.Limit()
	burst := defaultLimiter.Burst()
	if args.limit != -1 {
		limit = rate.Limit(args.limit)
	}
	if args.burst != -1 {
		burst = args.burst
	}

	minInterval := time.Duration(float64(time.Second) * 1.0 / float64(limit))
	consumer.Infof("================ Test settings ================")
	consumer.Infof("User: %v (ID %d)", profile.User.Username, profile.User.ID)
	consumer.Infof("Workers: %v, reqs per worker: %v (%v total reqs)", args.workers, args.requests, args.workers*args.requests)
	consumer.Infof("Limit: %v reqs/s - Burst: %v - (Minimum interval: %v)", limit, burst, minInterval)
	consumer.Infof("===============================================")

	var lastRequestMu sync.Mutex
	var lastRequest time.Time
	var zeroTime time.Time
	var violations []time.Duration

	rateLimited := 0
	allowedViolations := burst
	client := mc.NewClient(profile.APIKey)
	client.Limiter = rate.NewLimiter(limit, burst)
	client.OnRateLimited(func(req *http.Request, res *http.Response) {
		fmt.Fprintf(os.Stderr, "!")
		rateLimited++
	})

	client.OnOutgoingRequest(func(req *http.Request) {
		reqTime := time.Now()

		lastRequestMu.Lock()
		defer lastRequestMu.Unlock()

		if lastRequest == zeroTime {
			lastRequest = reqTime
			return
		}
		interval := reqTime.Sub(lastRequest)
		if interval < minInterval {
			if allowedViolations > 0 {
				allowedViolations--
			} else {
				deviation := minInterval - interval
				violations = append(violations, deviation)
			}
		}
		lastRequest = reqTime
	})

	startTime := time.Now()

	var mu sync.Mutex
	var durations []time.Duration

	var wg sync.WaitGroup
	wg.Add(args.workers * args.requests)
	for i := 0; i < args.workers; i++ {
		go func() {
			for j := 0; j < args.requests; j++ {
				beforeReq := time.Now()
				_, err := client.ListGameUploads(ctx, itchio.ListGameUploadsParams{
					GameID: 323326,
				})
				if err != nil {
					panic(errors.WithMessage(err, "While doing test API call"))
				}
				duration := time.Since(beforeReq)
				mu.Lock()
				durations = append(durations, duration)
				mu.Unlock()

				fmt.Fprintf(os.Stderr, ".")
				wg.Done()
			}
		}()
	}

	wg.Wait()
	fmt.Fprintf(os.Stderr, "\n")
	consumer.Infof("===================  Results   ================")
	consumer.Statf("All done in %v!", time.Since(startTime))

	{
		var min int64 = math.MaxInt64
		var max int64 = 0
		var mean float64 = 0

		for _, dt := range durations {
			d := int64(dt)
			if d < min {
				min = d
			}
			if d > max {
				max = d
			}
			mean += float64(d) / float64(len(durations))
		}

		consumer.Statf("Min:    %v", time.Duration(min))
		consumer.Statf("Mean:   %v", time.Duration(mean))
		consumer.Statf("Max:    %v", time.Duration(max))

		timePerReq := time.Duration(mean / float64(args.requests*args.workers))
		consumer.Statf("%.02f r/s", 1.0/timePerReq.Seconds())
	}

	if len(violations) > 0 {
		consumer.Debugf("%d violations", len(violations))
		var violationStrings []string
		for _, v := range violations {
			violationStrings = append(violationStrings, fmt.Sprintf("%v", v))
		}
		s := strings.Join(violationStrings, ", ")
		consumer.Debugf("%s", s)
	}
	if rateLimited > 0 {
		consumer.Infof("%d rate limiting events total", rateLimited)
		consumer.Infof("[ERROR] was rate limited, exiting with non-zero code")
		os.Exit(1)
	} else {
		consumer.Statf("No rate limiting events!")
	}

	return nil
}
