package xad

import "github.com/itchio/butler/archive"

type Handler struct {
}

var _ archive.Handler = (*Handler)(nil)

func NewHandler() archive.Handler {
	return &Handler{}
}

func (h *Handler) Name() string {
	return "xad"
}
