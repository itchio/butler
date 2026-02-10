package integrate

import (
	"bufio"
	"context"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/jsonrpc2"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/headway/state"
)

type cmdPipeRWC struct {
	in  io.ReadCloser
	out io.WriteCloser
}

var _ jsonrpc2.ReadWriteClose = (*cmdPipeRWC)(nil)

func (c *cmdPipeRWC) Read(p []byte) (int, error) {
	return c.in.Read(p)
}

func (c *cmdPipeRWC) Write(p []byte) (int, error) {
	return c.out.Write(p)
}

func (c *cmdPipeRWC) Close() error {
	outErr := c.out.Close()
	inErr := c.in.Close()
	if outErr != nil {
		return outErr
	}
	return inErr
}

func Test_StdioTransport(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	args := []string{
		"daemon",
		"--json",
		"--transport", "stdio",
		"--dbpath", "file::memory:?cache=shared",
		"--destiny-pid", conf.PidString,
		"--destiny-pid", conf.PpidString,
	}
	bExec := exec.CommandContext(ctx, conf.ButlerPath, args...)

	stdin, err := bExec.StdinPipe()
	must(err)

	stdout, err := bExec.StdoutPipe()
	must(err)

	stderr, err := bExec.StderrPipe()
	must(err)
	go func() {
		s := bufio.NewScanner(stderr)
		for s.Scan() {
			if os.Getenv("QUIET_TESTS") != "1" {
				t.Logf("[butler stderr] %s", s.Text())
			}
		}
	}()

	must(bExec.Start())

	waitErr := make(chan error, 1)
	go func() {
		waitErr <- bExec.Wait()
	}()

	select {
	case err := <-waitErr:
		t.Fatalf("butler daemon exited before first RPC: %+v", err)
	default:
	}

	consumer := &state.Consumer{
		OnMessage: func(lvl string, msg string) {
			if os.Getenv("QUIET_TESTS") != "1" {
				t.Logf("[client][%s] %s", lvl, msg)
			}
		},
	}

	h := newHandler(consumer)
	connCtx, connCancel := context.WithCancel(ctx)
	defer connCancel()

	conn := jsonrpc2.NewConn(connCtx, jsonrpc2.NewRwcTransport(&cmdPipeRWC{
		in:  stdout,
		out: stdin,
	}), h)
	defer conn.Close()

	rc := &butlerd.RequestContext{
		Ctx:      connCtx,
		Conn:     conn,
		Consumer: consumer,
	}

	vgr, err := messages.VersionGet.TestCall(rc, butlerd.VersionGetParams{})
	must(err)
	if vgr.Version == "" {
		t.Fatalf("Version.Get returned empty version")
	}

	_, err = messages.MetaShutdown.TestCall(rc, butlerd.MetaShutdownParams{})
	must(err)

	select {
	case err := <-waitErr:
		if err != nil {
			t.Fatalf("butler daemon did not exit cleanly after Meta.Shutdown: %+v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for stdio daemon to exit after Meta.Shutdown")
	}
}
