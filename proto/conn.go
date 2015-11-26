package proto

import (
	"fmt"
	"net"

	"golang.org/x/crypto/ssh"
)

type Conn struct {
	Conn ssh.Conn

	Chans <-chan ssh.NewChannel
	Reqs  <-chan *ssh.Request

	Permissions *ssh.Permissions

	sessionID string
}

// Connect tries to connect to a butler-proto server
func Connect(address string, identityPath string, version string) (*Conn, error) {
	tcpConn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	identity, err := readPrivateKey(identityPath)
	if err != nil {
		return nil, err
	}

	sshConfig := &ssh.ClientConfig{
		User:          "butler",
		Auth:          []ssh.AuthMethod{identity},
		ClientVersion: fmt.Sprintf("SSH-2.0-butler_%s", version),
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(tcpConn, "", sshConfig)
	if err != nil {
		return nil, err
	}

	return &Conn{
		Conn:  sshConn,
		Chans: chans,
		Reqs:  reqs,
	}, nil
}

func Accept(listener net.Listener, config *ssh.ServerConfig) (*Conn, error) {
	tcpConn, err := listener.Accept()
	if err != nil {
		return nil, err
	}

	sshConn, chans, reqs, err := ssh.NewServerConn(tcpConn, config)
	if err != nil {
		return nil, err
	}

	return &Conn{
		Conn:        sshConn.Conn,
		Permissions: sshConn.Permissions,
		Chans:       chans,
		Reqs:        reqs,
	}, nil
}
