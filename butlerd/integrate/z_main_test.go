package integrate

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/itchio/butler/butlerd/jsonrpc2"
	"github.com/pkg/errors"
)

type IntegrateConfig struct {
	OnCI       bool
	ButlerPath string
	PidString  string
	PpidString string
}

var conf IntegrateConfig

var (
	butlerPath = flag.String("butlerPath", "", "path to butler binary to test; built from the working tree when unset")
)

func TestMain(m *testing.M) {
	flag.Parse()

	conf.ButlerPath = *butlerPath
	conf.OnCI = os.Getenv("CI") != ""

	if conf.ButlerPath == "" {
		if conf.OnCI {
			log.Printf("Skipping integrate tests (on CI, no butler path specified)")
			os.Exit(0)
		}
		// The tests drive the daemon as a separate process, so a binary is
		// needed; one from PATH would silently be an older build.
		conf.ButlerPath = buildButler()
	}

	conf.PidString = strconv.FormatInt(int64(os.Getpid()), 10)
	conf.PpidString = strconv.FormatInt(int64(os.Getppid()), 10)

	status := m.Run()
	os.Exit(status)
}

// buildButler compiles the working tree to where `make build` puts it, the
// repository root, and returns the binary's path. The build cache makes a
// repeat build quick.
func buildButler() string {
	// tests run with the package directory as working directory
	out, err := filepath.Abs(filepath.Join("..", "..", "butler"))
	must(err)
	if runtime.GOOS == "windows" {
		out += ".exe"
	}
	log.Printf("Building butler from the working tree into %s", out)
	cmd := exec.Command("go", "build", "-o", out, "github.com/itchio/butler")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	must(errors.Wrap(cmd.Run(), "building butler for the integrate tests"))
	return out
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
