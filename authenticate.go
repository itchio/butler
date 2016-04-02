package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/itchio/go-itchio"
)

const (
	defaultPort = 22
)

func authenticateViaWharf() (*itchio.Client, error) {
	var identity = *appArgs.identity

	key, err := ioutil.ReadFile(identity)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}

		log.Println("\n" +
			"      ..........................\n" +
			"    ':cccclooooolcccccloooooocccc:,.\n" +
			"  ':cccccooooooolccccccooooooolccccc,.\n" +
			" ,;;;;;;cllllllc:;;;;;;clllllll:;;;;;;.\n" +
			" ,,,,,,,;cccccc;,,,,,,,,cccccc:,,,,,,,.\n" +
			" .',,,'..':cc:,...,,,'...;cc:,...',,'.\n" +
			"   .,;:dxl;,;;cxdc,,,;okl;,,,:odc,,,.\n" +
			"   ,kkkkkx:'..'okkkkkkxxo'...;oxxxxx,\n" +
			"   ,kkkk:       ...''...       ,dxxx,\n" +
			"   ,kkk:          .:c'          'xxx;\n" +
			"   ,kko         .,ccc:;.         :xx;\n" +
			"   ,kx.         .,;;,,'..         cl'\n" +
			"   ,kc           .''''.           'l'\n" +
			"   ,x.       ..............       .l'\n" +
			"   ,x'      ,oddddddddoolcc,      .l'\n" +
			"   'xo,...;ldxxxxxxxdollllllc;...'cl'\n" +
			"   .:ccc:ccccccccc:;;;;;;;;;;;;;;;;,.")

		log.Println("\nWelcome to the itch.io command-line tools!\n")
		log.Println("Open the following link in your browser to authenticate:\n")

		form := url.Values{}
		form.Add("client_id", "butler")
		form.Add("scope", "wharf")
		form.Add("response_type", "token")
		form.Add("redirect_uri", "http://localhost:8120/oauth/callback")
		query := form.Encode()

		uri := fmt.Sprintf("%s/user/oauth?%s", *appArgs.address, query)
		log.Println(uri)
		log.Println("\nI'll wait...")

		handler := func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Woo!")
		}

		http.HandleFunc("/", handler)
		http.ListenAndServe("localhost:", nil)

		time.Sleep(time.Second * 60)
		key = []byte("nope")
	}

	client := itchio.ClientWithKey(string(key))
	client.BaseURL = fmt.Sprintf("%s/api/1", *appArgs.address)

	return client, err
}
