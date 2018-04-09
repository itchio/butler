package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"
	"time"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database"
	"github.com/jinzhu/gorm"
	"github.com/sourcegraph/jsonrpc2"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/pkg/errors"
)

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("daemon", "Start a butlerd instance").Hidden()
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

	comm.Object("butlerd/secret-request", map[string]interface{}{
		"minLength": minSecretLength,
	})

	secretChan := make(chan string)
	go func() {
		secret := ""
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := scanner.Bytes()
			m := make(map[string]interface{})
			err := json.Unmarshal(line, &m)
			if err != nil {
				comm.Warnf("could not unmarshal JSON input, ignoring: %s", err.Error())
				continue
			}

			if typ, ok := m["type"].(string); ok {
				switch typ {
				case "butlerd/secret-result":
					if s, ok := m["secret"].(string); ok {
						secret = s
						secretChan <- secret
						comm.Logf("Received secret")
						return
					}
				default:
					comm.Warnf("unrecognized json message type %s, ignoring", typ)
				}
			} else {
				comm.Warnf("json message missing 'type' field, ignoring")
			}
		}
	}()

	var secret string
	select {
	case secret = <-secretChan:
		// woo
	case <-time.After(15 * time.Second):
		comm.Dief("butlerd: Timed out while waiting for secret")
	}

	if len(secret) < minSecretLength {
		comm.Dief("butlerd: Secret too short (must be %d chars, received %d chars) or more", minSecretLength, len(secret))
	}

	db, err := database.Open(ctx.DBPath)
	if err != nil {
		ctx.Must(errors.WithMessage(err, "opening DB for the first time"))
	}

	err = database.Prepare(db)
	if err != nil {
		ctx.Must(errors.WithMessage(err, "preparing DB"))
	}

	ctx.Must(Do(ctx, ctx.Context(), db, secret, func(addr string) {
		comm.Object("butlerd/listen-notification", map[string]interface{}{
			"address": addr,
		})
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

type OnListenFunc func(addr string)

func Do(mansionContext *mansion.Context, ctx context.Context, db *gorm.DB, secret string, onListen OnListenFunc) error {
	listenSpec := "127.0.0.1:"

	lis, err := net.Listen("tcp", listenSpec)
	if err != nil {
		return errors.WithStack(err)
	}

	onListen(lis.Addr().String())
	s := butlerd.NewServer(secret)

	h := &handler{
		ctx:    mansionContext,
		router: getRouter(db, mansionContext),
	}
	err = s.Serve(ctx, lis, h, comm.NewStateConsumer())
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
