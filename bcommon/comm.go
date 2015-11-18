package bcommon

import (
	"encoding/json"
	"fmt"
	"os"
)

type ButlerError struct {
	Error string
}

type ButlerDownloadStatus struct {
	Percent int
}

type ButlerMessage struct {
	Message string
}

// Send sends a JSON-encoded message to the client
func Send(v interface{}) {
	j, _ := json.Marshal(v)
	fmt.Println(string(j))
}

// Msg sends an informational message to the client
func Msg(msg string) {
	e := &ButlerMessage{Message: msg}
	Send(e)
}

// Die exits unsuccessfully after giving a reson to the client
func Die(msg string) {
	e := &ButlerError{Error: msg}
	Send(e)
	os.Exit(1)
}
