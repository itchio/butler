package mansion

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/itchio/butler/buildinfo"
	"github.com/itchio/butler/comm"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/httpkit/timeout"
	"github.com/itchio/wharf/pwr"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var LogHttp = os.Getenv("BUTLER_HTTP_DEBUG") == "1"

type DoCommand func(ctx *Context)

type Context struct {
	App      *kingpin.Application
	Commands map[string]DoCommand

	// Identity is the path to the credentials file
	Identity string

	// String to include in our user-agent
	UserAgentAddition string

	// Quiet silences all output
	Quiet bool

	// Verbose enables chatty output
	Verbose bool

	// Verbose enables JSON output
	JSON bool

	// Path to the local sqlite database
	DBPath string

	// When non-zero (and DBPath is set), authenticate by reading the
	// stored apiKey for this butlerd profile id, instead of using
	// BUTLER_API_KEY / the keyfile / interactive OAuth.
	ButlerdProfileID int64

	CompressionAlgorithm string
	CompressionQuality   int

	ContextTimeout int64

	HTTPClient    *http.Client
	HTTPTransport *http.Transport

	// ClientLogger, when set, is passed to go-itchio clients for HTTP debug logging
	ClientLogger *slog.Logger

	ctx     context.Context
	ctxOnce sync.Once

	// url of the itch.io API server we're talking to
	apiAddress string
	// url of the itch.io web instance we're talking to
	webAddress string
}

func NewContext(app *kingpin.Application) *Context {
	client := timeout.NewDefaultClient()
	originalTransport := client.Transport.(*http.Transport)

	ctx := &Context{
		App:           app,
		Commands:      make(map[string]DoCommand),
		HTTPClient:    client,
		HTTPTransport: originalTransport,
	}

	client.Transport = &UserAgentSetter{
		OriginalTransport: originalTransport,
		Context:           ctx,
	}

	return ctx
}

func (ctx *Context) Register(clause *kingpin.CmdClause, do DoCommand) {
	ctx.Commands[clause.FullCommand()] = do
}

func (ctx *Context) Must(err error) {
	if err != nil {
		if ctx.Verbose || ctx.JSON {
			comm.Dief("%+v", err)
		} else {
			comm.Dief("%s", err)
		}
	}
}

func (ctx *Context) UserAgent() string {
	version := buildinfo.Version
	if version == "head" && buildinfo.Commit != "" {
		version = buildinfo.Commit
	}

	res := fmt.Sprintf("butler/%s", version)
	if ctx.UserAgentAddition != "" {
		res = fmt.Sprintf("%s %s", res, ctx.UserAgentAddition)
	}
	return res
}

func (ctx *Context) CompressionSettings() pwr.CompressionSettings {
	var algo pwr.CompressionAlgorithm

	switch ctx.CompressionAlgorithm {
	case "none":
		algo = pwr.CompressionAlgorithm_NONE
	case "brotli":
		algo = pwr.CompressionAlgorithm_BROTLI
	case "gzip":
		algo = pwr.CompressionAlgorithm_GZIP
	default:
		panic(fmt.Errorf("Unknown compression algorithm: %s", algo))
	}

	return pwr.CompressionSettings{
		Algorithm: algo,
		Quality:   int32(ctx.CompressionQuality),
	}
}

// Ctx is cancelled by the first SIGINT or SIGTERM, after which the
// default handling is restored so a second signal kills the process even
// if the command ignores cancellation. Signals are only caught once Ctx
// or DefaultCtx is first called, since daemon and launch install their
// own handlers.
func (ctx *Context) Ctx() context.Context {
	ctx.ctxOnce.Do(func() {
		var cancel context.CancelFunc
		ctx.ctx, cancel = context.WithCancel(context.Background())
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
		go func() {
			sig := <-sigs
			signal.Stop(sigs)
			comm.Warnf("Got %s, stopping (send again to force)", sig)
			cancel()
		}()
	})
	return ctx.ctx
}

func (ctx *Context) DefaultCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx.Ctx(), ctx.requestTimeout())
}

// DetachedCtx is for cleanup requests that must still go out after the
// command has been interrupted.
func (ctx *Context) DetachedCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), ctx.requestTimeout())
}

func (ctx *Context) requestTimeout() time.Duration {
	return time.Duration(ctx.ContextTimeout) * time.Second
}

func (ctx *Context) SetClientLogger(logger *slog.Logger) {
	ctx.ClientLogger = logger
}

func (ctx *Context) NewClient(key string) *itchio.Client {
	client := itchio.ClientWithKey(strings.TrimSpace(key))
	client.HTTPClient = ctx.HTTPClient
	client.SetServer(ctx.APIAddress())
	client.UserAgent = ctx.UserAgent()

	if ctx.ClientLogger != nil {
		client.Logger = ctx.ClientLogger
	} else if LogHttp {
		client.Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}

	return client
}

func (ctx *Context) WebAddress() string {
	return ctx.webAddress
}

func (ctx *Context) APIAddress() string {
	return ctx.apiAddress
}

func (ctx *Context) SetAddress(address string) {
	var err error
	ctx.webAddress, err = stripApiSubdomain(address)
	ctx.Must(err)
	ctx.apiAddress, err = addApiSubdomain(address)
	ctx.Must(err)
}

func (ctx *Context) EnsureDBPath() {
	if ctx.DBPath == "" {
		comm.Dief("butlerd: Missing database path: use --dbpath path/to/butler.db")
	}
}

//

type UserAgentSetter struct {
	OriginalTransport http.RoundTripper
	Context           *Context
}

var _ http.RoundTripper = (*UserAgentSetter)(nil)

func (uas *UserAgentSetter) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", uas.Context.UserAgent())
	return uas.OriginalTransport.RoundTrip(req)
}

// ThrottleBandwidth caps all network traffic to kilobitsPerSecond, the
// unit of --throttle and Network.SetBandwidthThrottle, and returns the
// cap in bytes per second. 0 lifts the cap.
func ThrottleBandwidth(kilobitsPerSecond int64) int64 {
	bytesPerSecond := kilobitsPerSecond * 1024 / 8
	timeout.SetBandwidth(bytesPerSecond)
	return bytesPerSecond
}
