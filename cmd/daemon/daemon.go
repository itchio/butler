package daemon

import (
	"context"
	"encoding/base64"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"

	"crawshaw.io/sqlite"
	"github.com/google/gops/agent"
	"github.com/google/uuid"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database"
	"github.com/sourcegraph/jsonrpc2"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/pkg/errors"
)

var args = struct {
	writeSecret string
	writeCert   string
	destinyPids []int64
	transport   string
	log         bool
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("daemon", "Start a butlerd instance").Hidden()
	cmd.Flag("write-secret", "Path to write the secret to").StringVar(&args.writeSecret)
	cmd.Flag("write-cert", "Path to write the certificate to").StringVar(&args.writeCert)
	cmd.Flag("destiny-pid", "The daemon will shutdown whenever any of its destiny PIDs shuts down").Int64ListVar(&args.destinyPids)
	cmd.Flag("transport", "Which transport to use").Default("http").EnumVar(&args.transport, "http", "tcp")
	cmd.Flag("log", "Log all requests to stderr").BoolVar(&args.log)
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	if !comm.JsonEnabled() {
		comm.Notice("Hello from butler daemon", []string{"We can't do anything interesting without --json, bailing out", "", "Learn more: https://docs.itch.ovh/butlerd/master/"})
		os.Exit(1)
	}

	if ctx.DBPath == "" {
		comm.Dief("butlerd: dbPath must be set")
	}

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

	dbPool, err := sqlite.Open(ctx.DBPath, 0, 100)
	if err != nil {
		ctx.Must(errors.WithMessage(err, "opening DB for the first time"))
	}
	defer dbPool.Close()

	func() {
		conn := dbPool.Get(context.Background().Done())
		defer dbPool.Put(conn)
		err = database.Prepare(conn)
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

func tryListen(port string) (net.Listener, error) {
	spec := "127.0.0.1:" + port
	lis, err := net.Listen("tcp", spec)
	if err != nil {
		spec = "127.0.0.1:"
		lis, err = net.Listen("tcp", spec)
	}
	return lis, err
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
			Handler:  h,
			Consumer: consumer,
			Listener: listener,
			Secret:   secret,
			Log:      args.log,
		})
		if err != nil {
			return err
		}
	case "http":
		ts, err := butlerd.MakeTLSState()
		if err != nil {
			return err
		}

		httpListener, err := tryListen("13141")
		if err != nil {
			return err
		}

		httpsListener, err := tryListen("13142")
		if err != nil {
			return err
		}

		ca := base64.StdEncoding.EncodeToString(ts.CertPEMBlock)

		comm.Object("butlerd/listen-notification", map[string]interface{}{
			"secret": secret,
			"http": map[string]interface{}{
				"address": httpListener.Addr().String(),
			},
			"https": map[string]interface{}{
				"address": httpsListener.Addr().String(),
				"ca":      ca,
			},
		})

		if args.writeCert != "" {
			err := ioutil.WriteFile(args.writeCert, ts.CertPEMBlock, os.FileMode(0644))
			if err != nil {
				comm.Warnf("%v", err)
			}
		}

		if args.writeSecret != "" {
			err := ioutil.WriteFile(args.writeSecret, []byte(secret), os.FileMode(0644))
			if err != nil {
				comm.Warnf("%v", err)
			}
		}
		err = s.ServeHTTP(ctx, butlerd.ServeHTTPParams{
			HTTPListener:  httpListener,
			HTTPSListener: httpsListener,
			ShutdownChan:  h.router.ShutdownChan,
			Handler:       h,
			TLSState:      ts,
			Consumer:      consumer,
			Log:           args.log,
		})
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}
