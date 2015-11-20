package bio

import (
	"bytes"
	"encoding/gob"

	"gopkg.in/itchio/rsync-go.v0"
)

type Message struct {
	Sub interface{}
}

type UploadParams struct {
	GameId   int64
	Platform string
}

type SourceFile struct {
	Path   string
	Hashes []rsync.BlockHash
}

type FilePatched struct {
	Path    string
	ApplyTo string
}

type FileAdded struct {
	Path string
	Data []byte
}

type FileRemoved struct {
	Path string
}

func init() {
	gob.Register(Message{})
	gob.Register(UploadParams{})
	gob.Register(SourceFile{})
	gob.Register(FilePatched{})
	gob.Register(FileAdded{})
	gob.Register(FileRemoved{})
}

func (msg *Message) ToBytes() ([]byte, error) {
	buf := new(bytes.Buffer)
	genc := gob.NewEncoder(buf)

	err := genc.Encode(msg)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
