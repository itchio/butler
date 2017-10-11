package operate

import (
	"bufio"
	"encoding/json"
	"fmt"
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

		err := s.Err()
		if err != nil {
			comm.Warnf("JSON transport scanner got error: %s", err.Error())
		}

		close(jt.readQueue)
	}()
}

func (jt *JSONTransport) Read(messageType string) (GenericMessage, error) {
	for l := range jt.readQueue {
		if l["type"] == messageType {
			return l, nil
		}

		comm.Logf("Ignoring message of type %s (expected %s)", l["type"], messageType)
	}

	msg := fmt.Sprintf("EOF while trying to read a message of type %s", messageType)
	return nil, errors.New(msg)
}
