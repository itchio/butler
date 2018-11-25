package mitch

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/handlers"
	"github.com/itchio/wharf/state"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

type Server interface {
	Address() net.Addr
	Store() *Store
}

type server struct {
	ctx      context.Context
	address  net.Addr
	listener net.Listener
	opts     serverOpts
	store    *Store
	consumer *state.Consumer
}

type serverOpts struct {
	port     int
	consumer *state.Consumer
}

type ServerOpt func(opts *serverOpts)

func WithPort(port int) ServerOpt {
	return func(opts *serverOpts) {
		opts.port = port
	}
}

func WithConsumer(consumer *state.Consumer) ServerOpt {
	return func(opts *serverOpts) {
		opts.consumer = consumer
	}
}

func NewServer(ctx context.Context, options ...ServerOpt) (Server, error) {
	var opts serverOpts
	for _, o := range options {
		o(&opts)
	}

	consumer := opts.consumer
	if consumer == nil {
		consumer = &state.Consumer{
			OnMessage: func(lvl string, message string) {
				log.Printf("[%s] %s", lvl, message)
			},
		}
	}

	s := &server{
		ctx:      ctx,
		opts:     opts,
		store:    newStore(),
		consumer: consumer,
	}

	err := s.start()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return s, nil
}

func (s *server) start() error {
	addr := fmt.Sprintf("127.0.0.1:%d", s.opts.port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return errors.WithStack(err)
	}
	s.listener = listener
	s.address = listener.Addr()

	go func() {
		<-s.ctx.Done()
		listener.Close()
	}()

	go s.serve()
	return nil
}

func (s *server) Address() net.Addr {
	return s.address
}

func (s *server) Store() *Store {
	return s.store
}

type coolHandler func(r *response)

func (s *server) serve() {
	m := mux.NewRouter()
	handler := func(ch coolHandler) http.HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request) {
			res := &response{
				s:     s,
				w:     w,
				req:   req,
				store: s.store,
			}
			err := func() (retErr error) {
				defer func() {
					if r := recover(); r != nil {
						if rErr, ok := r.(error); ok {
							cause := errors.Cause(rErr)
							if ae, ok := cause.(APIError); ok {
								res.WriteError(ae.status, ae.messages...)
								return
							}
							retErr = rErr
						} else {
							retErr = errors.Errorf("panic: %+v", r)
						}
					}
				}()
				ch(res)
				return nil
			}()
			if err != nil {
				res.WriteError(500, fmt.Sprintf("internal error: %+v", err))
			}
		}
	}
	route := func(route string, ch coolHandler) {
		m.HandleFunc(route, handler(ch))
	}
	routePrefix := func(prefix string, ch coolHandler) {
		m.PathPrefix(prefix).Handler(handler(ch))
	}

	route("/profile", func(r *response) {
		r.RespondTo(RespondToMap{
			"GET": func() {
				r.CheckAPIKey()
				r.WriteJSON(Any{
					"user": FormatUser(r.currentUser),
				})
			},
		})
	})

	route("/games/{id}", func(r *response) {
		r.RespondTo(RespondToMap{
			"GET": func() {
				r.CheckAPIKey()
				gameID := r.Int64Var("id")
				game := r.FindGame(gameID)
				r.AssertAuthorization(game.CanBeViewedBy(r.currentUser))
				r.WriteJSON(Any{
					"game": FormatGame(game),
				})
			},
		})
	})

	route("/games/{id}/uploads", func(r *response) {
		r.RespondTo(RespondToMap{
			"GET": func() {
				r.CheckAPIKey()
				gameID := r.Int64Var("id")
				game := r.FindGame(gameID)
				r.AssertAuthorization(game.CanBeViewedBy(r.currentUser))
				uploads := r.store.ListUploadsByGame(gameID)
				r.WriteJSON(Any{
					"uploads": FormatUploads(uploads),
				})
			},
		})
	})

	route("/uploads/{id}/builds", func(r *response) {
		r.RespondTo(RespondToMap{
			"GET": func() {
				r.CheckAPIKey()
				uploadID := r.Int64Var("id")
				upload := r.FindUpload(uploadID)
				r.AssertAuthorization(upload.CanBeViewedBy(r.currentUser))
				builds := r.store.ListBuildsByUpload(uploadID)
				r.WriteJSON(Any{
					"builds": FormatBuilds(builds),
				})
			},
		})
	})

	route("/games/{id}/download-sessions", func(r *response) {
		r.RespondTo(RespondToMap{
			"POST": func() {
				r.CheckAPIKey()
				gameID := r.Int64Var("id")
				game := r.store.FindGame(gameID)
				r.AssertAuthorization(game.CanBeViewedBy(r.currentUser))
				r.WriteJSON(Any{
					"uuid": uuid.New().String(),
				})
			},
		})
	})

	route("/uploads/{id}/download", func(r *response) {
		r.RespondTo(RespondToMap{
			"GET": func() {
				r.CheckAPIKey()
				uploadID := r.Int64Var("id")
				upload := r.FindUpload(uploadID)
				r.AssertAuthorization(upload.CanBeDownloadedBy(r.currentUser))
				switch upload.Storage {
				case "hosted":
					r.ServeCDNAsset(upload)
				case "build":
					build := r.FindBuild(upload.Head)
					archive := build.GetFile("archive", "default")
					if archive == nil {
						Throw(404, "no archive for build")
					}
					r.ServeCDNAsset(archive)
				default:
					Throw(500, "unsupported storage")
				}
			},
		})
	})

	route("/builds/{id}/download/{type}/{subtype}", func(r *response) {
		r.RespondTo(RespondToMap{
			"GET": func() {
				r.CheckAPIKey()

				buildID := r.Int64Var("id")
				build := r.FindBuild(buildID)
				upload := r.FindUpload(build.UploadID)
				r.AssertAuthorization(upload.CanBeDownloadedBy(r.currentUser))

				typ := r.Var("type")
				subtype := r.Var("subtype")
				bf := build.GetFile(typ, subtype)
				if bf == nil {
					log.Printf("no build file found for %s/%s for build %d", typ, subtype, build.ID)
					Throw(404, fmt.Sprintf("no %s/%s build file", typ, subtype))
				}
				r.ServeCDNAsset(bf)
			},
		})
	})

	route("/builds/{id}/upgrade-paths/{target_id}", func(r *response) {
		r.RespondTo(RespondToMap{
			"GET": func() {
				r.CheckAPIKey()

				id := r.Int64Var("id")
				targetID := r.Int64Var("target_id")
				targetBuild := r.FindBuild(targetID)

				curr := targetBuild
				builds := []*Build{curr}
				for {
					curr = r.FindBuild(curr.ParentBuildID)
					builds = append(builds, curr)
					if curr.ID == id {
						break
					}
					if curr.ID < id {
						// woops, we went back too far and didn't find it
						break
					}
				}

				if curr.ID != id {
					Throw(404, "upgrade path not found")
				}

				// see https://github.com/golang/go/wiki/SliceTricks
				for i := len(builds)/2 - 1; i >= 0; i-- {
					opp := len(builds) - 1 - i
					builds[i], builds[opp] = builds[opp], builds[i]
				}

				var formattedBuilds []Any
				for _, b := range builds {
					item := FormatBuild(b)
					patches := s.store.SelectBuildFiles(
						NoSort(),
						Eq{
							"BuildID": b.ID,
							"Type":    "patch",
						},
					)
					var files []Any
					for _, p := range patches {
						files = append(files, FormatBuildFile(p))
					}
					item["files"] = files
					formattedBuilds = append(formattedBuilds, item)
				}
				res := Any{
					"upgrade_path": Any{
						"builds": formattedBuilds,
					},
				}
				r.WriteJSON(res)
			},
		})
	})

	routePrefix("/@cdn", func(r *response) {
		r.RespondTo(RespondToMap{
			"GET": func() {
				path := r.req.URL.Path
				path = strings.TrimPrefix(path, "/@cdn")
				f := r.store.CDNFiles[path]
				if f == nil {
					Throw(404, "not found")
				}

				contentLength := f.Size
				rangeHeader := r.req.Header.Get("Range")
				data := f.Contents
				if rangeHeader == "" {
					r.status = 200
				} else {
					rangeTokens := strings.Split(rangeHeader, "=")
					byteTokens := strings.Split(rangeTokens[1], "-")

					start := int64(0)
					if startVal, err := strconv.ParseInt(byteTokens[0], 10, 64); err == nil {
						start = startVal
					}
					end := f.Size - 1
					if endVal, err := strconv.ParseInt(byteTokens[1], 10, 64); err == nil {
						end = endVal
					}

					// note that the server will return internal error if the range is invalid
					data = data[start : end+1]
					contentLength = end + 1 - start
					r.status = 206
					r.Header().Set("content-range", fmt.Sprintf("bytes %d-%d/%d", start, end, f.Size))
				}

				r.Header().Set("content-length", fmt.Sprintf("%d", contentLength))
				r.Header().Set("accept-range", "bytes")
				r.Header().Set("content-type", "application/octet-stream")
				r.Header().Set("content-disposition", fmt.Sprintf("attachment; filename=%q", f.Filename))
				r.Header().Set("connection", "close")
				r.WriteHeader()

				src := bytes.NewReader(data)
				r.s.Logf("Serving %s", f.Filename)
				io.Copy(r.w, src)
				r.s.Logf("Serving %s (done)", f.Filename)
			},
		})
	})

	routePrefix("/", func(r *response) {
		Throw(404, "invalid api endpoint")
	})

	pR, pW, err := os.Pipe()
	defer pW.Close()
	must(err)
	loggedM := handlers.LoggingHandler(pW, m)
	go func() {
		consumer := s.consumer
		s := bufio.NewScanner(pR)
		for s.Scan() {
			consumer.Infof(s.Text())
		}
	}()
	http.Serve(s.listener, loggedM)
}

func (s *server) Logf(format string, args ...interface{}) {
	s.consumer.Infof(format, args...)
}
