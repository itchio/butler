package mitch

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type response struct {
	s      *server
	w      http.ResponseWriter
	req    *http.Request
	status int
	store  *Store

	currentUser *User
}

type Any map[string]interface{}

type APIError struct {
	status   int
	messages []string
}

func Error(status int, messages ...string) APIError {
	return APIError{
		status:   status,
		messages: messages,
	}
}

func Throw(status int, messages ...string) APIError {
	panic(Error(status, messages...))
}

func (ae APIError) Error() string {
	return fmt.Sprintf("api error (%d): %v", ae.status, ae.messages)
}

func (r *response) WriteError(status int, errors ...string) {
	r.status = status
	payload := map[string]interface{}{
		"errors": errors,
	}
	r.WriteJSON(payload)
}

func (r *response) WriteJSON(payload interface{}) {
	r.Header().Set("content-type", "application/json")
	r.WriteHeader()

	bs, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		panic(err)
	}

	r.Write(bs)
}

func (r *response) Header() http.Header {
	return r.w.Header()
}

func (r *response) WriteHeader() {
	status := r.status
	if r.status == 0 {
		status = 200
	}
	r.w.WriteHeader(status)
}

type RespondToMap map[string]func()

var (
	validRespondToMethods = map[string]bool{
		"GET":  true,
		"POST": true,
	}
)

func (r *response) RespondTo(m RespondToMap) {
	for k := range m {
		if !validRespondToMethods[k] {
			Throw(500, fmt.Sprintf("handler is trying to handle invalid method %s", k))
		}
	}

	if h, ok := m[r.req.Method]; ok {
		h()
	} else {
		Throw(400, "invalid method")
	}
}

func (r *response) Write(p []byte) {
	r.w.Write(p)
}

func (r *response) Int64Var(name string) int64 {
	res, err := strconv.ParseInt(r.Var(name), 10, 64)
	must(err)
	return res
}

func (r *response) Var(name string) string {
	return mux.Vars(r.req)[name]
}

func (r *response) CheckAPIKey() {
	keyString := r.req.Header.Get("Authorization")
	if keyString == "" {
		keyString = r.req.URL.Query().Get("api_key")
	}
	if keyString == "" {
		Throw(401, "authentication required")
	}

	apiKey := r.s.store.FindAPIKeysByKey(keyString)
	if apiKey == nil {
		Throw(403, "unauthorized")
	}

	r.currentUser = r.s.store.FindUser(apiKey.UserID)
	if r.currentUser == nil {
		Throw(500, "api key has no user")
	}
}

func (r *response) AssertAuthorization(authorized bool) {
	if !authorized {
		Throw(403, "forbidden")
	}
}

func (r *response) RedirectTo(url string) {
	r.Header().Set("Location", url)
	r.status = 302
	r.WriteHeader()
}

func (r *response) FindGame(gameID int64) *Game {
	game := r.store.FindGame(gameID)
	if game == nil {
		Throw(404, "game not found")
	}
	return game
}

func (r *response) FindUpload(uploadID int64) *Upload {
	upload := r.store.FindUpload(uploadID)
	if upload == nil {
		Throw(404, "upload not found")
	}
	return upload
}

func (r *response) FindBuild(buildID int64) *Build {
	build := r.store.FindBuild(buildID)
	if build == nil {
		Throw(404, "build not found")
	}
	return build
}

func (r *response) makeURL(format string, args ...interface{}) string {
	path := fmt.Sprintf(format, args...)
	url := fmt.Sprintf("http://%s%s", r.s.Address().String(), path)
	return url
}

type cdnAsset interface {
	CDNPath() string
}

func (r *response) ServeCDNAsset(ass cdnAsset) {
	r.RedirectTo(r.makeURL("/@cdn%s", ass.CDNPath()))
}
