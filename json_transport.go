package main

import (
	"bufio"
	"encoding/json"
	"os"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
)

// GenericMessage represents any message that can be JSON-encoded
type GenericMessage map[string]interface{}

// JSONTransport handles reading and writing json-lines from stdin/stdout
type JSONTransport struct {
	readQueue chan GenericMessage
}

// NewJSONTransport creates a new JSON transport (wee)
func NewJSONTransport() *JSONTransport {
	return &JSONTransport{
		readQueue: make(chan GenericMessage),
	}
}

// Start fires up goroutines to handle reading JSON-lines messages
func (jt *JSONTransport) Start() {
	go func() {
		s := bufio.NewScanner(os.Stdin)
		for s.Scan() {
			b := s.Bytes()
			var m map[string]interface{}
			err := json.Unmarshal(b, &m)
			if err != nil {
				// not json, igonre
				continue
			}

			jt.readQueue <- m
		}
	}()
}

func (jt *JSONTransport) Read(messageType string) (GenericMessage, error) {
	for l := range jt.readQueue {
		if l["type"] == messageType {
			return l, nil
		}

		comm.Logf("Ignoring unexpected message type %s", messageType)
	}

	return nil, errors.New("empty read queue")
}
