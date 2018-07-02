package daemon

import (
	"context"
	"log"
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
	http bool
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("daemon", "Start a butlerd instance").Hidden()
	cmd.Arg("http", "HTTP mode").BoolVar(&args.http)
	ctx.Register(cmd, do)
}

const minSecretLength = 256

func do(ctx *mansion.Context) {
	if !comm.JsonEnabled() {
		comm.Notice("Hello from butler daemon", []string{"We can't do anything interesting without --json, bailing out", "", "Learn more: https://docs.itch.ovh/butlerd/master/"})
		os.Exit(1)
	}

	if ctx.DBPath == "" {
		comm.Dief("butlerd: dbPath must be set")
	}

	generateSecret := func() (string, error) {
		res := ""
		for i := 0; i < 16; i++ {
			u, err := uuid.NewV4()
			if err != nil {
				return "", errors.WithStack(err)
			}
			res += u.String()
		}
		return res, nil
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

	ctx.Must(Do(ctx, context.Background(), dbPool, secret, func(addr string) {
		comm.Object("butlerd/listen-notification", map[string]interface{}{
			"secret":  secret,
			"address": addr,
		})
	}))
}

type handler struct {
	ctx    *mansion.Context
	router *butlerd.Router
}

func (h *handler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	log.Printf("Handle called: %#v", *req)
	if req.Notif {
		return
	}

	h.router.Dispatch(ctx, conn, req)
}

type OnListenFunc func(addr string)

func Do(mansionContext *mansion.Context, ctx context.Context, dbPool *sqlite.Pool, secret string, onListen OnListenFunc) error {
	listenSpec := "127.0.0.1:13140"

	lis, err := net.Listen("tcp", listenSpec)
	if err != nil {
		listenSpec = "127.0.0.1:"
		lis, err = net.Listen("tcp", listenSpec)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	onListen(lis.Addr().String())
	s := butlerd.NewServer(secret)

	h := &handler{
		ctx:    mansionContext,
		router: getRouter(dbPool, mansionContext),
	}

	err = s.Serve(ctx, butlerd.ServeParams{
		Listener: lis,
		Handler:  h,
		Consumer: comm.NewStateConsumer(),
	})
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
