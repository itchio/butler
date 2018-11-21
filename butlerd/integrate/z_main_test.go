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
	"github.com/sourcegraph/jsonrpc2"
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
		gmust(errors.New("Refusing to run integrate tests without --butlerPath"))
	}

	conf.PidString = strconv.FormatInt(int64(os.Getpid()), 10)
	conf.PpidString = strconv.FormatInt(int64(os.Getppid()), 10)

	status := m.Run()
	os.Exit(status)
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
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
		panic(fmt.Sprintf("%+v", errors.WithStack(err)))
	}
}
