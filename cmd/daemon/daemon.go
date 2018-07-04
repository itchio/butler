package daemon

import (
	"context"
	"encoding/base64"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"

	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database"
	uuid "github.com/satori/go.uuid"
	"github.com/sourcegraph/jsonrpc2"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/pkg/errors"
)

var args = struct {
	writeSecret string
	writeCert   string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("daemon", "Start a butlerd instance").Hidden()
	cmd.Flag("write-secret", "Path to write the secret to").StringVar(&args.writeSecret)
	cmd.Flag("write-cert", "Path to write the certificate to").StringVar(&args.writeCert)
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

	generateSecret := func() (string, error) {
		u, err := uuid.NewV4()
		if err != nil {
			return "", errors.WithStack(err)
		}
		return u.String(), nil
	}
	secret, err := generateSecret()
	if err != nil {
		ctx.Must(err)
	}

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

	ts, err := butlerd.MakeTLSState()
	if err != nil {
		ctx.Must(err)
	}

	ctx.Must(Do(ctx, context.Background(), dbPool, ts, secret, func(httpAddress string, httpsAddress string) {
		ca := base64.StdEncoding.EncodeToString(ts.CertPEMBlock)

		comm.Object("butlerd/listen-notification", map[string]interface{}{
			"secret": secret,
			"http": map[string]interface{}{
				"address": httpAddress,
			},
			"https": map[string]interface{}{
				"address": httpsAddress,
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
	}))
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

type OnListenFunc func(httpAddress string, httpsAddress string)

func tryListen(port string) (net.Listener, error) {
	spec := "127.0.0.1:" + port
	lis, err := net.Listen("tcp", spec)
	if err != nil {
		spec = "127.0.0.1:"
		lis, err = net.Listen("tcp", spec)
	}
	return lis, err
}

func Do(mansionContext *mansion.Context, ctx context.Context, dbPool *sqlite.Pool, ts *butlerd.TLSState, secret string, onListen OnListenFunc) error {
	httpListener, err := tryListen("13141")
	if err != nil {
		return err
	}

	httpsListener, err := tryListen("13142")
	if err != nil {
		return err
	}

	onListen(httpListener.Addr().String(), httpsListener.Addr().String())
	s := butlerd.NewServer(secret)

	h := &handler{
		ctx:    mansionContext,
		router: getRouter(dbPool, mansionContext),
	}

	err = s.Serve(ctx, butlerd.ServeParams{
		HTTPListener:  httpListener,
		HTTPSListener: httpsListener,
		Handler:       h,
		TLSState:      ts,
		Consumer:      comm.NewStateConsumer(),
	})
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
