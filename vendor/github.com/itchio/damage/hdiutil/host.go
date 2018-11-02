package hdiutil

import (
	"os/exec"
	"strings"

	plist "github.com/DHowett/go-plist"
	"github.com/itchio/wharf/state"
	"github.com/pkg/errors"
)

// Any represent any data from a plist file
type Any map[string]interface{}

type DumpFunc func(p ...interface{})

// Host allows communicating with hdiutil, and
// handles logging, parsing, etc.
type Host interface {
	SetDump(dump DumpFunc)
	Command(name string) CommandBuilder
}

type host struct {
	consumer *state.Consumer
	dump     DumpFunc
}

// NewHost configures and returns a new hdiutil host
func NewHost(consumer *state.Consumer) Host {
	return &host{
		consumer: consumer,
	}
}

func (h *host) SetDump(dump DumpFunc) {
	h.dump = dump
}

type CommandBuilder interface {
	WithArgs(args ...string) CommandBuilder
	WithInput(input string) CommandBuilder
	RunAndDecode(dst interface{}) error
	Run() error
}

type commandBuilder struct {
	host  *host
	name  string
	args  []string
	input string
}

func (h *host) Command(name string) CommandBuilder {
	return &commandBuilder{
		host: h,
		name: name,
	}
}

func (cb *commandBuilder) WithArgs(args ...string) CommandBuilder {
	cb.args = args
	return cb
}

func (cb *commandBuilder) WithInput(input string) CommandBuilder {
	cb.input = input
	return cb
}

func (cb *commandBuilder) Run() error {
	h := cb.host

	output, err := h.run(cb.input, cb.name, cb.args...)
	if err != nil {
		return errors.WithStack(err)
	}

	if h.dump != nil {
		h.dump(output)
	}
	return nil
}

func (cb *commandBuilder) RunAndDecode(dst interface{}) error {
	h := cb.host

	output, err := h.run(cb.input, cb.name, cb.args...)
	if err != nil {
		return errors.WithStack(err)
	}

	if h.dump != nil {
		result := make(Any)
		_, err = plist.Unmarshal(output, &result)
		h.dump(result)
	}

	_, err = plist.Unmarshal(output, dst)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (h *host) run(input string, subcmd string, args ...string) ([]byte, error) {
	h.consumer.Debugf("hdiutil ::: %s ::: %s", subcmd, strings.Join(args, " ::: "))

	hdiArgs := []string{subcmd}
	hdiArgs = append(hdiArgs, args...)
	cmd := exec.Command("hdiutil", hdiArgs...)
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return output, nil
}
