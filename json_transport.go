package main

import (
	"bufio"
	"encoding/json"
	"os"
)

type JSONTransport struct {
	input chan []byte
}

func NewJSONTransport() *JSONTransport {
	return &JSONTransport{
		input: make(chan []byte),
	}
}

func isJSON(b []byte) bool {
	var js map[string]interface{}
	return json.Unmarshal(b, &js) == nil
}

func (jt *JSONTransport) Start() {
	go func() {
		s := bufio.NewScanner(os.Stdin)
		for s.Scan() {
			b := s.Bytes()
			if isJSON(b) {
				jt.input <- b
			}
		}
	}()
}

func (jt *JSONTransport) Read() ([]byte, error) {
	l := <-jt.input
	return l, nil
}
