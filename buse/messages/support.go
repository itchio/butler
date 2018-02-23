package messages

type RequestMessage interface {
	Method() string
}

type NotificationMessage interface {
	Method() string
}
