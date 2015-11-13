package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Send sends a JSON-encoded message to the client
func send(v interface{}) {
	j, _ := json.Marshal(v)
	fmt.Println(string(j))
}

// Msg sends an informational message to the client
func msg(msg string) {
	e := &butlerMessage{Message: msg}
	send(e)
}

// Die exits unsuccessfully after giving a reson to the client
func die(msg string) {
	e := &butlerError{Error: msg}
	send(e)
	os.Exit(1)
}
