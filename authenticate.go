package main

import (
	"fmt"
	"strings"

	"github.com/itchio/butler/comm"
	"github.com/itchio/go-itchio"
	"github.com/itchio/wharf"
	"golang.org/x/crypto/ssh"
)

const (
	defaultPort = 22
)

func authenticateViaWharf() (*itchio.Client, error) {
	var identity = *appArgs.identity

	var address = *appArgs.address
	if !strings.Contains(address, ":") {
		address = fmt.Sprintf("%s:%d", address, defaultPort)
	}

	comm.Debugf("Authenticating via %s", address)
	conn, err := wharf.Connect(address, identity, "butler", version)
	if err != nil {
		return nil, err
	}
	comm.Debugf("Connected to %s", conn.Conn.RemoteAddr())

	go ssh.DiscardRequests(conn.Reqs)

	req := &wharf.AuthenticationRequest{}
	res := &wharf.AuthenticationResponse{}

	err = conn.SendRequest("authenticate", req, res)
	if err != nil {
		return nil, fmt.Errorf("Authentication error; %s", err.Error())
	}

	err = conn.Close()
	if err != nil {
		return nil, err
	}

	// TODO: if buildPath is an archive, warn and unpack it

	client := itchio.ClientWithKey(res.Key)
	client.BaseURL = res.ItchioBaseUrl

	return client, err
}
