package bah

import "github.com/itchio/butler/archive"

type Handler struct {
}

func NewHandler() archive.Handler {
	return &Handler{}
}

func (h *Handler) Name() string {
	return "bah"
}
