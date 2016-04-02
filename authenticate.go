package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"

	"github.com/itchio/go-itchio"
)

const (
	defaultPort = 22
	asciiArt    = "      ..........................\n" +
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
		"   .:ccc:ccccccccc:;;;;;;;;;;;;;;;;,.\n"
)

var callbackRe = regexp.MustCompile(`^\/oauth\/callback\/(.*)$`)

func authenticateViaOauth() (*itchio.Client, error) {
	var err error
	var identity = *appArgs.identity
	var key []byte

	makeClient := func(key string) *itchio.Client {
		client := itchio.ClientWithKey(key)
		client.BaseURL = fmt.Sprintf("%s/api/1", *appArgs.address)
		return client
	}

	key, err = ioutil.ReadFile(identity)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}

		done := make(chan string)
		errs := make(chan error)

		handler := func(w http.ResponseWriter, r *http.Request) {
			matches := callbackRe.FindStringSubmatch(r.RequestURI)
			if matches != nil {
				client := makeClient(matches[1])
				client.WharfStatus()

				w.Header().Set("Content-Type", "text/plain")
				fmt.Fprintf(w, asciiArt)
				done <- matches[1]
				return
			}

			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, `
        <!DOCTYPE html>
        <html>
        <head>
          <link href="https://fonts.googleapis.com/css?family=Lato:400,700" rel="stylesheet" type="text/css">
          <style>
            body {
              text-align: center;
              margin: 50px 0;
            }

            p {
              line-height: 1.6;
              font-size: 18px;
              font-family: Lato, sans-serif;
            }

            a, a:active, a:visited, a:hover {
              color: #FA5B5B;
            }

            /* A a pastel rainbow palette */
            @keyframes rainbow {
              from { color: #FFB3BA; }
              25%  { color: #FFDFBA; }
              50%  { color: #FFFFBA; }
              75%  { color: #BAFFC9; }
              to   { color: #BAE1FF; }
            }

            pre {
              animation: rainbow alternate 5s infinite linear;
              background: #1C1C1D;
              padding: 2em 0;
              line-height: 1.3;
              font-size: 16px;
              color: #FFB3BA;
              text-shadow: 0 0 20px;
              color: white;
            }
          </style>
        </head>
        <body>
          <pre id="art"></pre>
          <p id="message">
            Authenticating...
          </p>
          <script>
          'use strict'
          var key = location.hash.replace(/^#/, '')
          var xhr = new XMLHttpRequest()
          var $message = document.querySelector("#message")
          var $art = document.querySelector("#art")
          xhr.onload = function () {
            $art.innerHTML = xhr.responseText
            $message.innerHTML = "You're successfully authenticated! You can close this page and go back to your terminal."
          }
          xhr.onerror = function () {
            $message.innerHTML = "Copy the following code back in your terminal: " + key
          }
          xhr.open("POST", "/oauth/callback/" + key)
          xhr.send()
          </script>
        </body>
      </html>`)
		}

		http.HandleFunc("/", handler)

		var listener net.Listener
		listener, err = net.Listen("tcp", "localhost:0")
		if err != nil {
			return nil, err
		}

		go func() {
			err = http.Serve(listener, nil)
			if err != nil {
				errs <- err
			}
		}()

		log.Println("\n" + asciiArt)

		log.Println("\nWelcome to the itch.io command-line tools!\n")
		log.Println("Open the following link in your browser to authenticate:\n")

		form := url.Values{}
		form.Add("client_id", "butler")
		form.Add("scope", "wharf")
		form.Add("response_type", "token")
		form.Add("redirect_uri", fmt.Sprintf("http://%s/oauth/callback", listener.Addr().String()))
		query := form.Encode()

		uri := fmt.Sprintf("%s/user/oauth?%s", *appArgs.address, query)
		log.Println(uri)
		log.Println("\nI'll wait...")

		select {
		case err = <-errs:
			return nil, err
		case keyString := <-done:
			key = []byte(keyString)
			err = nil

			client := makeClient(keyString)
			_, err := client.WharfStatus()
			if err != nil {
				return nil, err
			}
			log.Printf("\nAuthenticated successfully! Saving key in %s...\n\n", identity)

			err = ioutil.WriteFile(identity, key, os.FileMode(0644))
			if err != nil {
				log.Printf("\nCould not save key: %s\n\n", err.Error())
			}
		}
	}

	return makeClient(string(key)), err
}
