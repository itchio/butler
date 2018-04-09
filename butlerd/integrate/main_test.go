package integrate

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/onsi/gocleanup"
	"github.com/pkg/errors"
)

var secret = strings.Repeat("dummy", 58)
var address string
var cancelButler context.CancelFunc

var (
	butlerPath = flag.String("butlerPath", "", "path to butler binary to test")
)

func TestMain(m *testing.M) {
	flag.Parse()

	onCi := os.Getenv("CI") != ""

	if !onCi {
		*butlerPath = "butler"
	}

	if *butlerPath == "" {
		if onCi {
			os.Exit(0)
		}
		gmust(errors.New("Not running (--butlerPath must be specified)"))
	}

	ctx := context.Background()
	ctx2, cancel := context.WithCancel(ctx)
	defer cancel()
	cancelButler = cancel

	bExec := exec.CommandContext(ctx2, *butlerPath, "daemon", "-j", "--dbpath", "file::memory:?cache=shared")
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

	go func() {
		s := bufio.NewScanner(stdout)
		for s.Scan() {
			line := s.Text()

			im := make(map[string]interface{})
			err := json.Unmarshal([]byte(line), &im)
			if err != nil {
				log.Printf("butler => %s", line)
				continue
			}

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

	address = <-addrChan
	gocleanup.Exit(m.Run())
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		cancelButler()
		t.Fatalf("%+v", err)
	}
}

func gmust(err error) {
	if err != nil {
		cancelButler()
		log.Printf("%+v", errors.WithStack(err))
		gocleanup.Exit(1)
	}
}
