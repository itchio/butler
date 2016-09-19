package netpool

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/itchio/butler/comm"
)

type SimpleBlockServer struct {
	// essential settings
	BasePath string
	Port     int

	// optional settings
	Latency time.Duration
}

func NewSimpleBlockServer(basePath string, port int) (*SimpleBlockServer, error) {
	sbs := &SimpleBlockServer{
		BasePath: basePath,
		Port:     port,

		Latency: 0,
	}

	return sbs, nil
}

func (sbs *SimpleBlockServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	tokens := strings.Split(strings.TrimPrefix(r.URL.String(), "/blocks/"), "/")
	last := tokens[len(tokens)-1]
	size, pErr := strconv.ParseInt(last, 10, 64)
	if pErr != nil {
		comm.Warnf("Invalid URL requested", r.URL.String())
		http.Error(w, "Invalid size", 400)
		return
	}

	time.Sleep(sbs.Latency)

	path := filepath.FromSlash(strings.TrimPrefix(r.URL.String(), "/"))

	comm.ProgressLabel(r.URL.String())

	f, pErr := os.Open(path)
	if pErr != nil {
		http.Error(w, "Block not found", 404)
		return
	}

	defer f.Close()

	bytesWritten, pErr := io.Copy(w, f)
	if pErr != nil {
		comm.Logf("Error when writing block to http: %s", pErr.Error())
		return
	}

	if bytesWritten != size {
		comm.Logf("Expected block to be %d, but got %d", size, bytesWritten)
		return
	}
}

func (sbs *SimpleBlockServer) ListenAndServe() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/blocks/", sbs.handleRequest)
	return http.ListenAndServe(fmt.Sprintf(":%d", sbs.Port), mux)
}

func (sbs *SimpleBlockServer) Source() Source {
	return &HttpSource{
		BaseURL: fmt.Sprintf("http://localhost:%d/blocks", sbs.Port),
	}
}
