package butlerd_test

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/itchio/butler/butlerd"
	"github.com/pkg/errors"
	"github.com/sourcegraph/jsonrpc2"
	"github.com/stretchr/testify/assert"
)

var jc *jsonrpc2.Conn

var cancelButler context.CancelFunc

var (
	butlerPath = flag.String("butlerPath", "", "path to butler binary to test")
)

func TestMain(m *testing.M) {
	flag.Parse()
	if *butlerPath == "" {
		log.Fatal("--butlerPath must be specified")
	}

	ctx := context.Background()
	ctx2, cancel := context.WithCancel(ctx)
	defer cancel()
	cancelButler = cancel

	bExec := exec.CommandContext(ctx2, *butlerPath, "daemon", "-j")
	stdin, err := bExec.StdinPipe()
	gmust(err)

	stdout, err := bExec.StdoutPipe()
	gmust(err)

	bExec.Stderr = os.Stderr
	gmust(bExec.Start())

	go func() {
		gmust(bExec.Wait())
	}()

	addrChan := make(chan string)

	secret := strings.Repeat("dummy", 58)

	go func() {
		s := bufio.NewScanner(stdout)
		for s.Scan() {
			line := s.Text()
			log.Printf("butler => %s", line)

			im := make(map[string]interface{})
			gmust(json.Unmarshal([]byte(line), &im))

			typ := im["type"].(string)
			switch typ {
			case "butlerd/secret-request":
				log.Printf("Sending secret")
				_, err = stdin.Write([]byte(fmt.Sprintf(`{"type": "butlerd/secret-result", "secret": %#v}%s`, secret, "\n")))
				gmust(err)
			case "butlerd/listen-notification":
				addrChan <- im["address"].(string)
			case "log":
				log.Printf("[butler] %s", im["message"].(string))
			default:
				gmust(errors.Errorf("unknown butlerd request: %s", typ))
			}
		}
	}()

	address := <-addrChan
	conn, err := net.DialTimeout("tcp", address, time.Second)
	gmust(err)

	h := &handler{
		secret: secret,
	}
	jc = jsonrpc2.NewConn(ctx, jsonrpc2.NewBufferedStream(conn, butlerd.LFObjectCodec{}), h)

	os.Exit(m.Run())
}

func Test_Version(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	vgr := &butlerd.VersionGetResult{}
	must(t, jc.Call(ctx, "Version.Get", &butlerd.VersionGetParams{}, vgr))

	assert.EqualValues(t, vgr.Version, "head")
}

type handler struct {
	secret string
}

var _ jsonrpc2.Handler = (*handler)(nil)

func (h handler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	switch req.Method {
	case "Handshake":
		im := make(map[string]interface{})
		json.Unmarshal(*req.Params, &im)
		msg := im["message"].(string)
		signature := fmt.Sprintf("%x", sha256.Sum256([]byte(h.secret+msg)))
		conn.Reply(ctx, req.ID, map[string]interface{}{
			"signature": signature,
		})
		return
	}

	conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
		Code:    jsonrpc2.CodeInternalError,
		Message: "Not implemented yet",
	})
}

func must(t *testing.T, err error) {
	if err != nil {
		cancelButler()
		t.Fatalf("%+v", err)
	}
}

func gmust(err error) {
	if err != nil {
		cancelButler()
		log.Fatalf("%+v", err)
	}
}
