package szah

import "github.com/itchio/butler/archive"

type Handler struct {
}

func (h *Handler) Name() string {
	return "szah"
}

func Register() {
	archive.RegisterHandler(&Handler{})
}
