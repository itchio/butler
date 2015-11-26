package proto

import (
	"encoding/hex"
	"io/ioutil"
	"net"

	"golang.org/x/crypto/ssh"
)

func (c *Conn) SessionID() string {
	if c.sessionID == "" {
		c.sessionID = hex.EncodeToString(c.Conn.SessionID())
	}
	return c.sessionID
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.Conn.RemoteAddr()
}

func (c *Conn) Close() error {
	return c.Conn.Close()
}

func readPrivateKey(file string) (ssh.AuthMethod, error) {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil, err
	}

	return ssh.PublicKeys(key), nil
}
