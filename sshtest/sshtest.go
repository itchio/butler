package main

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/itchio/butler/bio"

	"golang.org/x/crypto/ssh"
	"gopkg.in/kothar/brotli-go.v0/enc"
)

func publicKeyFile(file string) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil
	}

	log.Println("Our public key is", base64.StdEncoding.EncodeToString(key.PublicKey().Marshal()))
	return ssh.PublicKeys(key)
}

func main() {
	host := "localhost"
	port := 2222
	serverString := fmt.Sprintf("%s:%d", host, port)
	fmt.Printf("Trying to connect to %s\n", serverString)

	keyPath := fmt.Sprintf("%s/%s", os.Getenv("HOME"), ".ssh/id_rsa")
	key := publicKeyFile(keyPath)
	auth := []ssh.AuthMethod{key}

	sshConfig := &ssh.ClientConfig{
		User: "butler",
		Auth: auth,
	}

	serverConn, err := ssh.Dial("tcp", serverString, sshConfig)
	if err != nil {
		fmt.Printf("Server dial error: %s\n", err)
		return
	}
	defer serverConn.Close()
	fmt.Printf("Connected!\n")

	sendStuff(serverConn)
}

func sendStuff(serverConn *ssh.Client) {
	params := enc.NewBrotliParams()
	params.SetQuality(0)

	num := 10
	wait := make(chan bool)

	for i := 0; i < num; i++ {
		go func(i int) {
			defer func() { wait <- true }()

			payload := new(bytes.Buffer)
			gob.NewEncoder(payload).Encode(&i)

			ch, reqs, err := serverConn.OpenChannel("butler_send_file", payload.Bytes())
			if err != nil {
				panic(err)
			}
			log.Printf("channel %d's turn\n", i)

			go func() {
				for req := range reqs {
					req.Reply(true, nil)
				}
			}()

			bw := enc.NewBrotliWriter(params, ch)
			genc := gob.NewEncoder(bw)

			for j := 1; j <= (i * 3000); j++ {
				err = genc.Encode(bio.Message{
					Sub: bio.UploadParams{
						GameId:   12498,
						Platform: "osx",
					},
				})
				if err != nil {
					panic(err)
				}
			}

			time.Sleep(time.Duration(500) * time.Millisecond)
			err = bw.Close()
			if err != nil {
				panic(err)
			}
		}(i)
	}

	for i := 0; i < num; i++ {
		<-wait
	}
}
