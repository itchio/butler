package httpfile

import "fmt"

type NeedsRenewalError struct {
	url string
}

func (nre *NeedsRenewalError) Error() string {
	return "url has expired and needs renewal"
}

type ServerErrorCode int64

const (
	ServerErrorCodeUnknown ServerErrorCode = iota
	ServerErrorCodeNoRangeSupport
)

type ServerError struct {
	Host       string
	Message    string
	Code       ServerErrorCode
	StatusCode int
}

func (se *ServerError) Error() string {
	return fmt.Sprintf("%s: %s", se.Host, se.Message)
}
