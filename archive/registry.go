package archive

var handlers = []Handler{}

func RegisterHandler(h Handler) {
	handlers = append(handlers, h)
}
