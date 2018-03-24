package service

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"
	"time"

	"github.com/itchio/butler/buse"
	"github.com/sourcegraph/jsonrpc2"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
)

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("service", "Start up the butler service").Hidden()
	ctx.Register(cmd, do)
}

const minSecretLength = 256

func do(ctx *mansion.Context) {
	comm.Result(map[string]interface{}{
		"type":      "secret-request",
		"minLength": minSecretLength,
	})

	secretChan := make(chan string)
	go func() {
		secret := ""
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			line := scanner.Bytes()
			m := make(map[string]interface{})
			ctx.Must(json.Unmarshal(line, &m))
			if s, ok := m["secret"].(string); ok {
				secret = s
				secretChan <- secret
			}
		}
	}()

	var secret string
	select {
	case secret = <-secretChan:
		// woo
	case <-time.After(1 * time.Second):
		comm.Dief("timed out while waiting for secret")
	}

	if len(secret) < minSecretLength {
		comm.Dief("secret too short (must be %d chars) or more", minSecretLength)
	}

	ctx.Must(Do(ctx, ctx.Context(), secret, func(addr string) {
		comm.Result(map[string]interface{}{
			"type":    "server-listening",
			"address": addr,
		})
	}))
}

type handler struct {
	ctx    *mansion.Context
	router *buse.Router
}

func (h *handler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	if req.Notif {
		return
	}

	h.router.Dispatch(ctx, conn, req)
}

type OnListenFunc func(addr string)

func Do(mansionContext *mansion.Context, ctx context.Context, secret string, onListen OnListenFunc) error {
	listenSpec := "127.0.0.1:"

	lis, err := net.Listen("tcp", listenSpec)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	onListen(lis.Addr().String())
	s := buse.NewServer(secret)

	h := &handler{
		ctx:    mansionContext,
		router: getRouter(mansionContext),
	}
	err = s.Serve(ctx, lis, h, comm.NewStateConsumer())
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
