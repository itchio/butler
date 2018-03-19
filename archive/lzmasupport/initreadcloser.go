package lzmasupport

import "io"

type initReadCloser struct {
	init func() (io.ReadCloser, error)
	rc   io.ReadCloser
}

var _ io.ReadCloser = (*initReadCloser)(nil)

func (irc *initReadCloser) Read(buf []byte) (int, error) {
	if irc.rc == nil {
		rc, err := irc.init()
		if err != nil {
			return 0, err
		}
		irc.rc = rc
	}

	return irc.rc.Read(buf)
}

func (irc *initReadCloser) Close() error {
	if irc.rc == nil {
		return nil
	}

	return irc.rc.Close()
}
