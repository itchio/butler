package bah

import "github.com/itchio/butler/archive"

type Handler struct {
}

func (h *Handler) Name() string {
	return "bah"
}

func Register() {
	archive.RegisterHandler(&Handler{})
}
