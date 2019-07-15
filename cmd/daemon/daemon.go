package daemon

import (
	"context"
	"net"
	"os"
	"path/filepath"

	"github.com/itchio/butler/butlerd/horror"

	"crawshaw.io/sqlite"
	"github.com/google/gops/agent"
	"github.com/google/uuid"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database"
	"github.com/itchio/headway/state"
	"github.com/sourcegraph/jsonrpc2"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/pkg/errors"
)

var args = struct {
	destinyPids []int64
	transport   string
	keepAlive   bool
	log         bool
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("daemon", "Start a butlerd instance").Hidden()
	cmd.Flag("destiny-pid", "The daemon will shutdown whenever any of its destiny PIDs shuts down").Int64ListVar(&args.destinyPids)
	cmd.Flag("transport", "Which transport to use").Default("tcp").EnumVar(&args.transport, "http", "tcp")
	cmd.Flag("keep-alive", "Accept multiple TCP connections, stay up until killed or a destiny PID shuts down").BoolVar(&args.keepAlive)
	cmd.Flag("log", "Log all requests to stderr").BoolVar(&args.log)
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	if !comm.JsonEnabled() {
		comm.Notice("Hello from butler daemon", []string{"We can't do anything interesting without --json, bailing out", "", "Learn more: https://docs.itch.ovh/butlerd/master/"})
		os.Exit(1)
	}

	ctx.EnsureDBPath()

	err := agent.Listen(agent.Options{
		Addr:            "localhost:0",
		ShutdownCleanup: true,
	})
	if err != nil {
		comm.Warnf("butlerd: Could not start gops agent: %+v", err)
	}

	for _, destinyPid := range args.destinyPids {
		go tieDestiny(destinyPid)
	}

	generateSecret := func() string {
		var res string
		for rounds := 4; rounds > 0; rounds-- {
			res += uuid.New().String()
		}
		return res
	}
	secret := generateSecret()

	err = os.MkdirAll(filepath.Dir(ctx.DBPath), 0755)
	if err != nil {
		ctx.Must(errors.WithMessage(err, "creating DB directory if necessary"))
	}

	justCreated := false
	_, statErr := os.Stat(ctx.DBPath)
	if statErr != nil {
		comm.Logf("butlerd: creating new DB at %s", ctx.DBPath)
		justCreated = true
	}

	dbPool, err := sqlite.Open(ctx.DBPath, 0, 100)
	if err != nil {
		ctx.Must(errors.WithMessage(err, "opening DB for the first time"))
	}
	defer dbPool.Close()

	err = func() (retErr error) {
		defer horror.RecoverInto(&retErr)

		conn := dbPool.Get(context.Background().Done())
		defer dbPool.Put(conn)
		return database.Prepare(&state.Consumer{
			OnMessage: func(lvl string, msg string) {
				comm.Logf("[db prepare] [%s] %s", lvl, msg)
			},
		}, conn, justCreated)
	}()
	if err != nil {
		ctx.Must(errors.WithMessage(err, "preparing DB"))
	}

	ctx.Must(Do(ctx, context.Background(), dbPool, secret))
}

type handler struct {
	ctx    *mansion.Context
	router *butlerd.Router
}

func (h *handler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	if req.Notif {
		return
	}

	h.router.Dispatch(ctx, conn, req)
}

func Do(mansionContext *mansion.Context, ctx context.Context, dbPool *sqlite.Pool, secret string) error {
	s := butlerd.NewServer(secret)
	h := &handler{
		ctx:    mansionContext,
		router: getRouter(dbPool, mansionContext),
	}
	consumer := comm.NewStateConsumer()

	switch args.transport {
	case "tcp":
		listener, err := net.Listen("tcp", "127.0.0.1:")
		if err != nil {
			return err
		}

		comm.Object("butlerd/listen-notification", map[string]interface{}{
			"secret": secret,
			"tcp": map[string]interface{}{
				"address": listener.Addr().String(),
			},
		})

		err = s.ServeTCP(ctx, butlerd.ServeTCPParams{
			Handler:   h,
			Consumer:  consumer,
			Listener:  listener,
			Secret:    secret,
			Log:       args.log,
			KeepAlive: args.keepAlive,

			ShutdownChan: h.router.ShutdownChan,
		})
		if err != nil {
			return err
		}
	case "http":
		comm.Dief("The HTTP transport is deprecated. Use TCP instead.")
	}

	return nil
}
