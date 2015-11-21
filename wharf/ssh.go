package wharf

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"encoding/hex"
	"io/ioutil"
	"log"
	"net"

	"gopkg.in/kothar/brotli-go.v0/dec"
	"gopkg.in/kothar/brotli-go.v0/enc"

	"github.com/itchio/butler/bio"
	"golang.org/x/crypto/ssh"
)

type Channel struct {
	ch *ssh.Channel

	bw *enc.BrotliWriter
	br *dec.BrotliReader

	genc *gob.Encoder
	gdec *gob.Decoder
}

type Conn struct {
	Conn ssh.Conn

	Chans <-chan ssh.NewChannel
	Reqs  <-chan *ssh.Request

	Permissions *ssh.Permissions

	sessionID string
}

func Connect(address string, identityPath string) (*Conn, error) {
	bio.Logf("Trying to connect to %s", address)

	identity, err := readPrivateKey(identityPath)
	if err != nil {
		return nil, err
	}

	sshConfig := &ssh.ClientConfig{
		User: "butler",
		Auth: []ssh.AuthMethod{identity},
	}

	tcpConn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(tcpConn, "", sshConfig)
	if err != nil {
		return nil, err
	}
	bio.Logf("Connected to %s", sshConn.RemoteAddr())

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

func (c *Conn) OpenCompressedChannel(chType string, payload interface{}) (*Channel, error) {
	payloadBuf := new(bytes.Buffer)
	err := gob.NewEncoder(payloadBuf).Encode(&payload)
	if err != nil {
		return nil, err
	}

	ch, reqs, err := c.Conn.OpenChannel(chType, payloadBuf.Bytes())
	if err != nil {
		return nil, err
	}

	go func() {
		for req := range reqs {
			if req.WantReply {
				req.Reply(true, nil)
			}
		}
	}()

	params := enc.NewBrotliParams()
	params.SetQuality(1)
	bw := enc.NewBrotliWriter(params, ch)
	genc := gob.NewEncoder(bw)

	br := dec.NewBrotliReader(ch)
	gdec := gob.NewDecoder(br)

	cch := &Channel{
		br:   br,
		bw:   bw,
		genc: genc,
		gdec: gdec,
	}

	return cch, nil
}

func (c *Conn) SendRequest(name string, wantReply bool, payload interface{}) (bool, interface{}, error) {
	var payloadBytes []byte = nil
	if payload != nil {
		payloadBuf := new(bytes.Buffer)
		err := gob.NewEncoder(payloadBuf).Encode(&payload)
		if err != nil {
			return false, nil, err
		}
		payloadBytes = payloadBuf.Bytes()
	}

	status, replyBytes, err := c.Conn.SendRequest(name, wantReply, payloadBytes)
	if err != nil {
		log.Println("in sendrequest")
		return false, nil, err
	}

	var reply interface{} = nil
	if len(replyBytes) > 0 {
		err := gob.NewDecoder(bytes.NewReader(replyBytes)).Decode(&reply)
		if err != nil {
			log.Println("when parsing reply")
			return false, nil, err
		}
	}

	return status, reply, nil
}

func (ch *Channel) Close() error {
	err := ch.bw.Close()
	if err != nil {
		return err
	}

	err = ch.br.Close()
	if err != nil {
		return err
	}

	return nil
}

func (ch *Channel) Send(graal interface{}) error {
	return ch.genc.Encode(&graal)
}

func (ch *Channel) Receive() (interface{}, error) {
	var graal interface{}
	err := ch.gdec.Decode(&graal)
	if err != nil {
		return nil, err
	}

	return graal, nil
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

	bio.Logf("our public key is %s", base64.StdEncoding.EncodeToString(key.PublicKey().Marshal())[:25]+"...")
	return ssh.PublicKeys(key), nil
}
