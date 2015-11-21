package main

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
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
	if len(os.Args) < 3 {
		bio.Dief("Usage: butler game_id platform")
	}
	gameId, _ := strconv.ParseInt(os.Args[1], 10, 64)
	platform := os.Args[2]

	err := run(gameId, platform)
	if err != nil {
		bio.Dief("sshtest failed with: %s", err.Error())
	}
	return
}

func run(gameId int64, platform string) error {
	host := "localhost"
	port := 2222
	serverString := fmt.Sprintf("%s:%d", host, port)
	bio.Logf("Trying to connect to %s\n", serverString)

	keyPath := fmt.Sprintf("%s/%s", os.Getenv("HOME"), ".ssh/id_rsa")
	key := publicKeyFile(keyPath)
	auth := []ssh.AuthMethod{key}

	sshConfig := &ssh.ClientConfig{
		User: "butler",
		Auth: auth,
	}

	serverConn, err := ssh.Dial("tcp", serverString, sshConfig)
	if err != nil {
		return err
	}
	defer serverConn.Close()
	bio.Log("Connected!")

	rum, err := bio.Marshal(bio.UploadParams{
		GameId:   gameId,
		Platform: platform,
	})

	if err != nil {
		return err
	}

	ok, _, err := serverConn.SendRequest("butler/upload-params", true, rum)
	if err != nil {
		return err
	}

	if !ok {
		bio.Dief("server couldn't find upload to replace :(")
	}

	return sendStuff(serverConn)
}

func sendStuff(serverConn *ssh.Client) error {
	params := enc.NewBrotliParams()
	params.SetQuality(0)

	num := 10
	wait := make(chan bool)

	for i := 0; i < num; i++ {
		go func(i int) {
			defer func() { wait <- true }()

			payload := new(bytes.Buffer)
			gob.NewEncoder(payload).Encode(&i)

			ch, reqs, err := serverConn.OpenChannel("butler/send-file", payload.Bytes())
			if err != nil {
				panic(err)
			}
			log.Printf("channel %d's turn\n", i)

			go func() {
				for req := range reqs {
					log.Println("got request", req)
					if req.WantReply {
						req.Reply(true, nil)
					}
				}
			}()

			bw := enc.NewBrotliWriter(params, ch)
			genc := gob.NewEncoder(bw)

			for j := 1; j <= (i * 3000); j++ {
				var fa interface{} = bio.FileAdded{
					Path: "Hello",
				}
				err = genc.Encode(&fa)
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
	return nil
}
