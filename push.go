package main

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"

	"github.com/itchio/butler/comm"
	"github.com/itchio/go-itchio"
	"github.com/itchio/wharf"
)

func push(buildPath string, spec string) {
	must(doPush(buildPath, spec))
}

const (
	defaultPort = 22
)

func doPush(buildPath string, spec string) error {
	address := *pushArgs.address

	if !strings.Contains(address, ":") {
		address = fmt.Sprintf("%s:%d", address, defaultPort)
	}

	comm.Logf("Connecting to %s", address)
	conn, err := wharf.Connect(address, *pushArgs.identity, "butler", version)
	if err != nil {
		return err
	}
	defer conn.Close()
	comm.Logf("Connected to %s", conn.Conn.RemoteAddr())

	go ssh.DiscardRequests(conn.Reqs)

	req := &wharf.AuthenticationRequest{}
	res := &wharf.AuthenticationResponse{}

	err = conn.SendRequest("authenticate", req, res)
	if err != nil {
		return fmt.Errorf("Authentication error; %s", err.Error())
	}

	// TODO: if buildPath is an archive, warn and unpack it

	client := itchio.ClientWithKey(res.Key)

	target, channel, err := parseSpec(spec)
	if err != nil {
		return err
	}

	newBuildRes, err := client.CreateBuild(target, channel)
	if err != nil {
		return err
	}

	buildID := newBuildRes.ID
	buildID = buildID

	return nil
}

func parseSpec(spec string) (string, string, error) {
	tokens := strings.Split(spec, ":")

	if len(tokens) == 1 {
		return "", "", fmt.Errorf("invalid spec: %s, missing channel (examples: %s:windows-32-beta, %s:linux-64)", spec, spec, spec)
	} else if len(tokens) != 2 {
		return "", "", fmt.Errorf("invalid spec: %s, expected something of the form user/page:channel", spec)
	}

	return tokens[0], tokens[1], nil
}

func parseAddress(address string) string {
	if strings.Contains(address, ":") {
		return address
	} else {
		return fmt.Sprintf("%s:%d", address, defaultPort)
	}
}
