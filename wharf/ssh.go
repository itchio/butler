package wharf

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"io/ioutil"

	"gopkg.in/kothar/brotli-go.v0/dec"
	"gopkg.in/kothar/brotli-go.v0/enc"

	"github.com/itchio/butler/bio"
	"golang.org/x/crypto/ssh"
)

type Conn ssh.Client

func Connect(endpoint string, identityPath string) (*Conn, error) {
	bio.Logf("Trying to connect to %s", endpoint)

	identity, err := readPrivateKey(identityPath)
	if err != nil {
		return nil, err
	}

	sshConfig := &ssh.ClientConfig{
		User: "butler",
		Auth: []ssh.AuthMethod{identity},
	}

	client, err := ssh.Dial("tcp", endpoint, sshConfig)
	if err != nil {
		return nil, err
	}
	bio.Log("Connected!")

	return (*Conn)(client), nil
}

type Channel struct {
	ch *ssh.Channel

	bw *enc.BrotliWriter
	br *dec.BrotliReader

	genc *gob.Encoder
	gdec *gob.Decoder
}

func (conn *Conn) OpenCompressedChannel(chType string, payload interface{}) (*Channel, error) {
	payloadBuf := new(bytes.Buffer)
	err := gob.NewEncoder(payloadBuf).Encode(&payload)
	if err != nil {
		return nil, err
	}

	ch, reqs, err := conn.OpenChannel(chType, payloadBuf.Bytes())
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
		ch:   &ch,
		br:   br,
		bw:   bw,
		genc: genc,
		gdec: gdec,
	}

	return cch, nil
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

	bio.Logf("our public key is %s", base64.StdEncoding.EncodeToString(key.PublicKey().Marshal()))
	return ssh.PublicKeys(key), nil
}
