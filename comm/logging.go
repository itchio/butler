package comm

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/itchio/butler/art"
	"github.com/olekukonko/tablewriter"
	"github.com/skratchdot/open-golang/open"
)

var settings = &struct {
	noProgress     bool
	quiet          bool
	verbose        bool
	json           bool
	panic          bool
	assumeYes      bool
	adamLovesBeeps bool
}{
	false,
	false,
	false,
	false,
	false,
	false,
	false,
}

// Configure sets all logging options in one go
func Configure(noProgress, quiet, verbose, json, panic bool, assumeYes bool, adamLovesBeeps bool) {
	settings.noProgress = noProgress
	settings.quiet = quiet
	settings.verbose = verbose
	settings.json = json
	settings.panic = panic
	settings.assumeYes = assumeYes

	// NOW it's a feature, not a bug <3
	if adamLovesBeeps {
		themes["cp437"].OpSign = "•"
	}
}

type jsonMessage map[string]interface{}

type yesNoResponse struct {
	Response bool
}

// YesNo asks the user whether to proceed or not
// won't work in json mode for now.
func YesNo(question string) bool {
	if settings.json {
		if settings.assumeYes {
			return true
		}

		send("yesno", jsonMessage{"question": question})
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		input := scanner.Text()

		res := yesNoResponse{}
		err := json.Unmarshal([]byte(input), &res)
		if err != nil {
			Logf("Couldn't unmarshal response %s", input)
			Logf("...assuming no")
			return false
		}

		return res.Response
	}

	fmt.Printf(":: %s [y/N] ", question)

	if settings.assumeYes {
		fmt.Printf("y (--assume-yes)\n")
		return true
	}
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	answer := strings.ToLower(scanner.Text())

	return answer == "y"
}

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

// Notice prints a box with important info in it.
// UX style guide: don't abuse it or people will stop reading it.
func Notice(header string, lines []string) {
	if settings.json {
		Logf("notice: %s", header)
		for _, line := range lines {
			Logf("notice: %s", line)
		}
	} else {
		table := tablewriter.NewWriter(os.Stdout)
		table.SetAutoFormatHeaders(false)
		table.SetColWidth(60)
		table.SetHeader([]string{header})
		for _, line := range lines {
			table.Append([]string{line})
		}
		table.Render()
	}
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

// Result sends a result
func Result(value interface{}) {
	send("result", jsonMessage{
		"value": value,
	})
}

type printerFunc func()

func ResultOrPrint(value interface{}, p printerFunc) {
	if settings.json {
		Result(value)
	} else {
		p()
	}
}

func Request(operation string, request string, params interface{}) {
	send("request", jsonMessage{
		"operation": operation,
		"request":   request,
		"params":    params,
	})
}

func OperationError(operation string, code string, value interface{}) {
	send("operation-error", jsonMessage{
		"operation": operation,
		"code":      code,
	})
}

// Dief is a formatted variant of Die
func Dief(format string, args ...interface{}) {
	Die(fmt.Sprintf(format, args...))
}

func Login(uri string) {
	send("login", jsonMessage{
		"uri": uri,
	})
}

// sends a message to the client
func send(msgType string, obj jsonMessage) {
	if settings.json {
		obj["type"] = msgType
		obj["time"] = time.Now().UTC().Unix()
		if msgType == "log" {
			if obj["level"] == "debug" {
				if !settings.quiet && settings.verbose {
					// k, let it through
				} else {
					// no thanks!
					return
				}
			}
		}

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
		case "result":
			// don't show outside json mode
		case "login":
			uri, _ := (obj["uri"]).(string)
			showLogin(uri)
		case "progress":
			// already handled by pb
		default:
			log.Println(msgType, obj)
		}
	}
}

func showLogin(uri string) {
	log.Println("\n" + art.ItchLogo)
	log.Println("\nWelcome to the itch.io command-line tools!")
	open.Start(uri) // disregard error
	log.Println("If it hasn't already, open the following link in your browser to authenticate:")

	log.Println(uri)

	if runtime.GOOS == "windows" {
		log.Println("\n(To copy text in cmd.exe: Alt+Space, Edit->Mark, select text, press Enter)")
	}

	log.Println("\nIf you're running this on a remote server, the redirect will fail to load.")
	log.Println("In that case, copy the address you're redirected to, paste it below, and press enter.")
}

// sends a JSON-encoded message to the client
func sendJSON(obj jsonMessage) {
	json, _ := json.Marshal(obj)
	fmt.Println(string(json))
}
