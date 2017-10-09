package xad

import "github.com/itchio/butler/archive"

type Handler struct {
}

var _ archive.Handler = (*Handler)(nil)

func (h *Handler) Name() string {
	return "xad"
}

func Register() {
	archive.RegisterHandler(&Handler{})
}
