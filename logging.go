package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type jsonMessage map[string]interface{}

// Log sends an informational message to the client
func Log(msg string) {
	send("log", jsonMessage{
		"message": msg,
		"level":   "info",
	})
}

func Logf(format string, args ...interface{}) {
	Log(fmt.Sprintf(format, args...))
}

// Warn sends a warning message to the client
func Warn(msg string) {
	send("log", jsonMessage{
		"message": msg,
		"level":   "warning",
	})
}

func Warnf(format string, args ...interface{}) {
	Warn(fmt.Sprintf(format, args...))
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
	if *appArgs.json {
		obj["type"] = msgType
		sendJSON(obj)
	} else {
		switch msgType {
		case "log":
			if obj["level"] == "warning" {
				log.Printf("Warning: %s\n", obj["message"])
			} else {
				log.Println(obj["message"])
			}
		case "error":
			log.Fatalln(obj["message"])
		case "progress":
			setBarProgress(obj["percentage"].(float64))
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
