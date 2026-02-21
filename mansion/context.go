package mansion

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/itchio/butler/buildinfo"
	"github.com/itchio/butler/comm"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/httpkit/timeout"
	"github.com/itchio/wharf/pwr"
	kingpin "github.com/alecthomas/kingpin/v2"
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

	CompressionAlgorithm string
	CompressionQuality   int

	ContextTimeout int64

	HTTPClient    *http.Client
	HTTPTransport *http.Transport

	// ClientLogger, when set, is passed to go-itchio clients for HTTP debug logging
	ClientLogger *slog.Logger

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

func (ctx *Context) DefaultCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Duration(ctx.ContextTimeout)*time.Second)
}

func (ctx *Context) SetClientLogger(logger *slog.Logger) {
	ctx.ClientLogger = logger
}

func (ctx *Context) NewClient(key string) *itchio.Client {
	client := itchio.ClientWithKey(key)
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
