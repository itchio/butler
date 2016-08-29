package uploader

import "fmt"

type netError struct {
	err    error
	status gcsStatus
}

func (ne *netError) Error() string {
	if ne.err != nil {
		return fmt.Sprintf("network error: %s", ne.err.Error())
	}
	return fmt.Sprintf("gcs status %s", ne.status)
}

type retryError struct {
	committedBytes int64
}

func (re *retryError) Error() string {
	return fmt.Sprintf("retrying, %d bytes committed", re.committedBytes)
}
