package bio

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type jsonMessage map[string]interface{}

var (
	JsonOutput = false
	Quiet      = false
)

func Progress(perc float64) {
	if Quiet {
		return
	}
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

// sends a message to the client
func send(msgType string, obj jsonMessage) {
	if JsonOutput {
		obj["type"] = msgType
		sendJSON(obj)
	} else {
		switch msgType {
		case "log":
			log.Println(obj["message"])
		case "error":
			log.Fatalln(obj["message"])
		case "progress":
			log.Printf("progress: %.2f%%\n", obj["percentage"])
		default:
			log.Println(msgType, obj)
		}
	}
}

// sends a JSON-encoded message to the client
func sendJSON(obj jsonMessage) {
	json, _ := json.Marshal(obj)
	fmt.Println(string(json))
}
