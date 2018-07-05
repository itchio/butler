package integrate

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/exec"
	"strconv"
	"testing"

	"github.com/pkg/errors"
	"github.com/sourcegraph/jsonrpc2"
)

var secret string
var address string
var ca []byte
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cancelButler = cancel

	pidString := strconv.FormatInt(int64(os.Getpid()), 10)
	ppidString := strconv.FormatInt(int64(os.Getppid()), 10)

	args := []string{
		"daemon",
		"--json",
		"--dbpath", "file::memory:?cache=shared",
		"--destiny-pid", pidString,
		"--destiny-pid", ppidString,
	}
	bExec := exec.CommandContext(ctx, *butlerPath, args...)

	stdout, err := bExec.StdoutPipe()
	gmust(err)

	bExec.Stderr = os.Stderr
	gmust(bExec.Start())

	go func() {
		gmust(bExec.Wait())
	}()

	s := bufio.NewScanner(stdout)
	addrChan := make(chan string)

	go func() {
		defer cancel()

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
			case "butlerd/listen-notification":
				secret = im["secret"].(string)
				httpsBlock := im["https"].(map[string]interface{})
				ca, err = base64.StdEncoding.DecodeString(httpsBlock["ca"].(string))
				gmust(err)
				addrChan <- httpsBlock["address"].(string)
			case "log":
				log.Printf("[butler] %s", im["message"].(string))
			default:
				gmust(errors.Errorf("unknown butlerd request: %s", typ))
			}
		}
	}()

	select {
	case address = <-addrChan:
		// cool!
	case <-ctx.Done():
		gmust(errors.Errorf("cancelled"))
	}

	status := m.Run()
	cancel()
	os.Exit(status)
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		cancelButler()
		if je, ok := errors.Cause(err).(*jsonrpc2.Error); ok {
			if je.Data != nil {
				bs := []byte(*je.Data)
				intermediate := make(map[string]interface{})
				jErr := json.Unmarshal(bs, &intermediate)
				if jErr != nil {
					t.Errorf("could not Unmarshal json-rpc2 error data: %v", jErr)
					t.Errorf("data was: %s", string(bs))
				} else {
					t.Errorf("json-rpc2 full stack:\n%s", intermediate["stack"])
				}
			}
		}
		t.Fatalf("%+v", err)
	}
}

func gmust(err error) {
	if err != nil {
		cancelButler()
		log.Printf("%+v", errors.WithStack(err))
	}
}
