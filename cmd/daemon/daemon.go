package daemon

import (
	"context"
	"encoding/base64"
	"io/ioutil"
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
	writeSecret string
	writeCert   string
	destinyPids []int64
	transport   string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("daemon", "Start a butlerd instance").Hidden()
	cmd.Flag("write-secret", "Path to write the secret to").StringVar(&args.writeSecret)
	cmd.Flag("write-cert", "Path to write the certificate to").StringVar(&args.writeCert)
	cmd.Flag("destiny-pid", "The daemon will shutdown whenever any of its destiny PIDs shuts down").Int64ListVar(&args.destinyPids)
	cmd.Flag("transport", "Which transport to use").Default("http").EnumVar(&args.transport, "http", "tcp")
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

	for _, destinyPid := range args.destinyPids {
		go func(destinyPid int64) {
			proc, err := os.FindProcess(int(destinyPid))
			if err != nil {
				log.Printf("While looking for destiny PID %d: %+v", destinyPid, err)
				os.Exit(1)
			}

			if proc == nil {
				log.Printf("Desinty PID %d exited, exiting too", destinyPid)
				os.Exit(1)
			}

			_, err = proc.Wait()
			if err != nil {
				log.Printf("While waiting on destiny PID %d: %+v, exiting", destinyPid, err)
				os.Exit(1)
			}

			log.Printf("Destiny PID %d exited, exiting too", destinyPid)
			os.Exit(0)
		}(destinyPid)
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
			"tcp": map[string]interface{}{
				"address": listener.Addr().String(),
			},
		})

		err = s.ServeTCP(ctx, butlerd.ServeTCPParams{
			Handler:  h,
			Consumer: consumer,
			Listener: listener,
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
		err = s.Serve(ctx, butlerd.ServeParams{
			HTTPListener:  httpListener,
			HTTPSListener: httpsListener,
			Handler:       h,
			TLSState:      ts,
			Consumer:      consumer,
		})
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}
