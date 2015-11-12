package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/itchio/brotli-go/dec"
	"github.com/itchio/brotli-go/enc"
	"github.com/dustin/go-humanize"
	"golang.org/x/crypto/ssh"
)

var version = "head"

type butlerError struct {
	Error string
}

type butlerDownloadStatus struct {
	Percent int
}

const bufferSize = 128 * 1024

func main() {
	if len(os.Args) < 2 {
		err("Missing command")
	}
	cmd := os.Args[1]

	switch cmd {
	case "version":
		fmt.Println(fmt.Sprintf("butler version %s", version))
	case "dl":
		dl()
	case "test-ssh":
		testSSH()
	case "test-brotli":
		testBrotli()
	default:
		err("Invalid command")
	}
}

func send(v interface{}) {
	j, _ := json.Marshal(v)
	fmt.Println(string(j))
}

func err(msg string) {
	e := &butlerError{Error: msg}
	send(e)
	os.Exit(1)
}

func dl() {
	if len(os.Args) < 4 {
		err("Missing url or dest for dl command")
	}
	url := os.Args[2]
	dest := os.Args[3]

	initialBytes := int64(0)
	stats, err := os.Lstat(dest)
	if err == nil {
		initialBytes = stats.Size()
	}

	bytesWritten := initialBytes

	out, _ := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	defer out.Close()

	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-", bytesWritten))
	resp, _ := client.Do(req)
	defer resp.Body.Close()

	for {
		n, _ := io.CopyN(out, resp.Body, bufferSize)
		bytesWritten += n

		totalBytes := (initialBytes + resp.ContentLength)
		status := &butlerDownloadStatus{
			Percent: int(bytesWritten * 100 / totalBytes)}
		send(status)

		if n == 0 {
			break
		}
	}
}

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

func testSSH() {
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

func testBrotli() {
	start := time.Now()

	src := os.Args[2]
	inputBuffer, err := ioutil.ReadFile(src)
	if err != nil {
		panic(err)
	}

	log.Println("Read file in", time.Since(start))
	log.Println("Uncompressed size is", humanize.Bytes(uint64(len(inputBuffer))))
	start = time.Now()

	var decoded []byte

	for q := 0; q <= 9; q++ {
		params := enc.NewBrotliParams()
		params.SetQuality(q)

		encoded, err := enc.CompressBuffer(params, inputBuffer, make([]byte, 1))
		if err != nil {
			panic(err)
		}

		log.Println("Compressed (q=", q, ") to", humanize.Bytes(uint64(len(encoded))), "in", time.Since(start))
		start = time.Now()

		decoded, err = dec.DecompressBuffer(encoded, make([]byte, 1))
		if err != nil {
			panic(err)
		}

		log.Println("Decompressed in", time.Since(start))
		start = time.Now()
	}

	if !bytes.Equal(inputBuffer, decoded) {
		log.Println("Decoded output does not match original input")
		return
	}

	log.Println("Compared in", time.Since(start))
	start = time.Now()

	log.Println("Round-trip through brotli successful!")
}
