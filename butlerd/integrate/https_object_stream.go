package integrate

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/sourcegraph/jsonrpc2"
)

type httpsObjectStream struct {
	address   string
	secret    string
	cid       string
	client    *http.Client
	transport *http.Transport

	id int64

	ctx    context.Context
	cancel context.CancelFunc

	feedChan chan []byte

	errors chan error
}

var _ jsonrpc2.ObjectStream = (*httpsObjectStream)(nil)

func (s *httpsObjectStream) Go() {
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.feedChan = make(chan []byte)
	s.cid = "testcid"
	s.errors = make(chan error)
	s.client = &http.Client{
		Transport: s.transport,
	}

	go func() {
		s.errors <- s.listen()
	}()
}

func (s *httpsObjectStream) onMsg(msg map[string]string) error {
	_, hasId := msg["id"]
	_, hasData := msg["data"]
	if hasId && hasData {
		s.feedChan <- []byte(msg["data"])
		return nil
	} else {
		// ignore
		return nil
	}
}

func (s *httpsObjectStream) listen() error {
	query := make(url.Values)
	query.Set("secret", s.secret)
	query.Set("cid", s.cid)
	feedURL := "https://" + s.address + "/feed?" + query.Encode()

	req, err := http.NewRequest("GET", feedURL, nil)
	if err != nil {
		return err
	}
	req = req.WithContext(s.ctx)

	res, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		body, _ := ioutil.ReadAll(res.Body)

		return errors.Errorf("Expected HTTP %d but got %d: %s", 200, res.StatusCode, string(body))
	}

	if res.Header.Get("content-type") != "text/event-stream" {
		return errors.Errorf("Expected content-type (%s) but got (%s)", "text/event-stream", res.Header.Get("content-type"))
	}

	scan := bufio.NewScanner(res.Body)
	msg := make(map[string]string)
	// cf. https://html.spec.whatwg.org/multipage/server-sent-events.html#event-stream-interpretation
	for scan.Scan() {
		line := scan.Text()

		if line == "" {
			// If the line is empty (a blank line)
			// Dispatch the event
			err := s.onMsg(msg)
			if err != nil {
				return err
			}

			for k := range msg {
				delete(msg, k)
			}
		} else if strings.HasPrefix(line, ":") {
			// If the line starts with a U+003A COLON character (:)
			// Ignore the line.
		} else if strings.ContainsAny(line, ":") {
			// If the line contains a U+003A COLON character (:)
			tokens := strings.SplitN(line, ":", 2)

			// Collect the characters on the line before the first U+003A COLON character (:), and let field be that string.
			field := tokens[0]

			// Collect the characters on the line after the first U+003A COLON character (:), and let value be that string.
			value := tokens[1]
			// If value starts with a U+0020 SPACE character, remove it from value.
			value = strings.TrimPrefix(value, " ")

			// Process the field using the steps described below, using field as the field name and value as the field value.
			msg[field] = value
		}
	}

	err = scan.Err()
	if err != nil {
		return err
	}

	return nil
}

func (s *httpsObjectStream) WriteObject(obj interface{}) error {
	marshalled, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	go func() {
		err := s.writeObject(marshalled)
		if err != nil {
			log.Printf("While writing object %s: %+v", marshalled, err)
			s.cancel()
		}
	}()
	return nil
}

func (s *httpsObjectStream) writeObject(marshalled []byte) error {
	var url string

	intermediate := make(map[string]interface{})
	err := json.Unmarshal(marshalled, &intermediate)
	if err != nil {
		return err
	}

	_, hasMethod := intermediate["method"]
	expectedStatus := 200
	if hasMethod {
		// it's a call!
		url = "https://" + s.address + "/call/" + intermediate["method"].(string)

		marshalled, err = json.Marshal(intermediate["params"])
		if err != nil {
			return err
		}
	} else {
		// it's a reply
		expectedStatus = 204
		url = "https://" + s.address + "/reply"
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(marshalled))
	if err != nil {
		return err
	}
	req = req.WithContext(s.ctx)
	req.Header.Set("x-secret", s.secret)
	req.Header.Set("x-cid", s.cid)

	if hasMethod {
		req.Header.Set("x-id", strconv.FormatInt(s.id, 10))
		s.id++
	}

	res, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != expectedStatus {
		body, _ := ioutil.ReadAll(res.Body)

		return errors.Errorf("Expected HTTP %d, but got %d: %s", expectedStatus, res.StatusCode, string(body))
	}

	if expectedStatus == 200 {
		response, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		s.feedChan <- response
	}

	return nil
}

func (s *httpsObjectStream) ReadObject(obj interface{}) error {
	select {
	case msg := <-s.feedChan:
		return json.Unmarshal(msg, obj)
	case <-s.ctx.Done():
		return io.EOF
	}
}

func (s *httpsObjectStream) Close() error {
	s.cancel()
	return nil
}
