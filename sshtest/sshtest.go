package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"golang.org/x/crypto/ssh"
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
	host := "butler.itch.zone"
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
	fmt.Printf("Connected!\n")

	ch, _, err := serverConn.OpenChannel("butler", []byte{})
	if err != nil {
		panic(err)
	}

	ch.Write([]byte("Hi"))
	ch.Close()

	serverConn.Close()
}
