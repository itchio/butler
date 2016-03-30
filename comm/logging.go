package comm

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

var settings = &struct {
	noProgress bool
	quiet      bool
	verbose    bool
	json       bool
	panic      bool
}{
	false,
	false,
	false,
	false,
	false,
}

// Configure sets all logging options in one go
func Configure(noProgress, quiet, verbose, json, panic bool) {
	settings.noProgress = noProgress
	settings.quiet = quiet
	settings.verbose = verbose
	settings.json = json
	settings.panic = panic
}

type jsonMessage map[string]interface{}

// Opf prints a formatted string informing the user on what operation we're doing
func Opf(format string, args ...interface{}) {
	Logf("%s %s", theme.OpSign, fmt.Sprintf(format, args...))
}

// Statf prints a formatted string informing the user how fast the operation went
func Statf(format string, args ...interface{}) {
	Logf("%s %s", theme.StatSign, fmt.Sprintf(format, args...))
}

// Log sends an informational message to the client
func Log(msg string) {
	Logl("info", msg)
}

// Logf sends a formatted informational message to the client
func Logf(format string, args ...interface{}) {
	Loglf("info", format, args...)
}

// Warn lets the user know about a problem that's non-critical
func Warn(msg string) {
	Logl("warn", msg)
}

// Warnf is a formatted variant of Warn
func Warnf(format string, args ...interface{}) {
	Loglf("warning", format, args...)
}

// Debug messages are like Info messages, but printed only when verbose
func Debug(msg string) {
	Logl("debug", msg)
}

// Debugf is a formatted variant of Debug
func Debugf(format string, args ...interface{}) {
	Loglf("debug", format, args...)
}

// Logl logs a message of a given level
func Logl(level string, msg string) {
	send("log", jsonMessage{
		"message": msg,
		"level":   level,
	})
}

// Loglf logs a formatted message of a given level
func Loglf(level string, format string, args ...interface{}) {
	Logl(level, fmt.Sprintf(format, args...))
}

// Die exits with a non-zero exit code after giving a reson to the client
func Die(msg string) {
	send("error", jsonMessage{
		"message": msg,
	})
}

// Dief is a formatted variant of Die
func Dief(format string, args ...interface{}) {
	Die(fmt.Sprintf(format, args...))
}

// sends a message to the client
func send(msgType string, obj jsonMessage) {
	if settings.json {
		obj["type"] = msgType
		sendJSON(obj)
		if msgType == "error" {
			os.Exit(1)
		}
	} else {
		switch msgType {
		case "log":
			if obj["level"] == "info" {
				if !settings.quiet {
					log.Println(obj["message"])
				}
			} else if obj["level"] == "debug" {
				if !settings.quiet && settings.verbose {
					log.Println(obj["message"])
				}
			} else {
				log.Printf("%s: %s\n", obj["level"], obj["message"])
			}
		case "error":
			EndProgress()
			if settings.panic {
				log.Panicln(obj["message"])
			} else {
				log.Println(obj["message"])
				os.Exit(1)
			}
		case "progress":
			setBarProgress(obj["percentage"].(float64) * 0.01)
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
