package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type jsonMessage map[string]interface{}

func Progress(perc float64) {
	if *appArgs.quiet {
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

var csvCols []string

func CsvCol(cols ...interface{}) {
	if csvCols == nil {
		csvCols = make([]string, 0)
	}

	for _, col := range cols {
		csvCols = append(csvCols, fmt.Sprint(col))
	}
}

func CsvFinish() {
	csvWriter := csv.NewWriter(os.Stdout)
	csvWriter.Write(csvCols)
	csvWriter.Flush()
}

// sends a message to the client
func send(msgType string, obj jsonMessage) {
	if *appArgs.csv {
		// don't send that
	} else if *appArgs.json {
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
