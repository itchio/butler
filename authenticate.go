package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/itchio/go-itchio"
	"github.com/olekukonko/tablewriter"
)

const (
	asciiArt = "      ..........................\n" +
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

func login() {
	must(doLogin())
}

func doLogin() error {
	var identity = *appArgs.identity
	_, err := os.Lstat(identity)
	hasSavedCredentials := !os.IsNotExist(err)

	if hasSavedCredentials {
		client, err := authenticateViaOauth()
		if err != nil {
			return err
		}

		_, err = client.WharfStatus()
		if err != nil {
			return err
		}

		fmt.Println("Your local credentials are valid!\n")
		fmt.Println("If you want to log in as another account, use the `butler logout` command first,")
		fmt.Println("or specify a different credentials path with the `-i` flag.")
	} else {
		// this does the full login flow + saves
		_, err := authenticateViaOauth()
		if err != nil {
			return err
		}
		return nil
	}

	return nil
}

func logout() {
	must(doLogout())
}

func doLogout() error {
	var identity = *appArgs.identity

	_, err := os.Lstat(identity)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No saved credentials at", identity)
			fmt.Println("Nothing to do.")
			return nil
		}
	}

	fmt.Printf(":: Do you want to erase your saved API key? [y/N] ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	answer := strings.ToLower(scanner.Text())
	if answer != "y" {
		fmt.Println("Okay, not erasing credentials. Bye!")
		return nil
	}

	err = os.Remove(identity)
	if err != nil {
		return err
	}

	fmt.Println("You've successfully erased the API key that was saved on your computer.\n")

	table := tablewriter.NewWriter(os.Stdout)
	table.SetColWidth(50)
	table.SetHeader([]string{"Important note"})
	table.Append([]string{"Note: this command does not invalidate the API key itself. If you wish to revoke it (for example, because it's been compromised), you should do so in your user settings:\n"})
	table.Append([]string{""})
	table.Append([]string{fmt.Sprintf("  %s/user/settings\n\n", *appArgs.address)})
	table.Render()

	return nil
}

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
			_, err = client.WharfStatus()
			if err != nil {
				return nil, err
			}
			log.Printf("\nAuthenticated successfully! Saving key in %s...\n", identity)

			err = os.MkdirAll(path.Dir(identity), os.FileMode(0777))
			if err != nil {
				log.Printf("\nCould not create directory for storing API key: %s\n\n", err.Error())
				err = nil
			} else {
				err = ioutil.WriteFile(identity, key, os.FileMode(0644))
				if err != nil {
					log.Printf("\nCould not save API key: %s\n\n", err.Error())
					err = nil
				}
			}
		}
	}

	return makeClient(string(key)), err
}
