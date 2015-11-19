package bio

import (
	"encoding/json"
	"fmt"
	"os"
)

type jsonMessage map[string]interface{}

func Progress(perc float64) {
	send("progress", jsonMessage{
		"percentage": perc,
	})
}

// Msg sends an informational message to the client
func Log(msg string) {
	send("log", jsonMessage{
		"message": msg,
	})
}

func Logf(format string, args ...interface{}) {
	Log(fmt.Sprintf(format, args...))
}

// Die exits unsuccessfully after giving a reson to the client
func Die(msg string) {
	send("error", jsonMessage{
		"message": msg,
	})
	os.Exit(1)
}

func Dief(format string, args ...interface{}) {
	Die(fmt.Sprintf(format, args...))
}

// sends a JSON-encoded message to the client
func send(msgType string, obj jsonMessage) {
	obj["type"] = msgType
	json, _ := json.Marshal(obj)
	fmt.Println(string(json))
}
