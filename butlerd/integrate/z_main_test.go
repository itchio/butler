package integrate

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"testing"

	"github.com/pkg/errors"
	"github.com/itchio/butler/butlerd/jsonrpc2"
)

type IntegrateConfig struct {
	OnCI       bool
	ButlerPath string
	PidString  string
	PpidString string
}

var conf IntegrateConfig

var (
	butlerPath = flag.String("butlerPath", "", "path to butler binary to test")
)

func TestMain(m *testing.M) {
	flag.Parse()

	conf.ButlerPath = *butlerPath
	conf.OnCI = os.Getenv("CI") != ""

	if conf.ButlerPath == "" && !conf.OnCI {
		conf.ButlerPath = "butler"
	}

	if conf.ButlerPath == "" {
		if conf.OnCI {
			log.Printf("Skipping integrate tests (on CI, no butler path specified)")
			os.Exit(0)
		}
		must(errors.New("Refusing to run integrate tests without --butlerPath"))
	}

	conf.PidString = strconv.FormatInt(int64(os.Getpid()), 10)
	conf.PpidString = strconv.FormatInt(int64(os.Getppid()), 10)

	status := m.Run()
	os.Exit(status)
}

func must(err error) {
	if err != nil {
		if je, ok := errors.Cause(err).(*jsonrpc2.Error); ok {
			if je.Data != nil {
				bs := []byte(*je.Data)
				intermediate := make(map[string]interface{})
				jErr := json.Unmarshal(bs, &intermediate)
				if jErr == nil {
					panic(fmt.Sprintf("%v: JSON-RPC stack trace:\n%+v", err, intermediate["stack"]))
				} else {
					log.Printf("could not Unmarshal json-rpc2 error data: %v", jErr)
					log.Printf("data was: %s", string(bs))
				}
			}
		}
		panic(fmt.Sprintf("%+v", err))
	}
}
