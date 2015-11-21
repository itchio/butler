package bio

import (
	"bytes"
	"encoding/gob"

	"gopkg.in/itchio/rsync-go.v0"
)

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
	Register()
}

func Register() {
	gob.Register(UploadParams{})
	gob.Register(SourceFile{})
	gob.Register(FilePatched{})
	gob.Register(FileAdded{})
	gob.Register(FileRemoved{})
}

func Marshal(value interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	genc := gob.NewEncoder(buf)

	err := genc.Encode(&value)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func Unmarshal(buf []byte) (interface{}, error) {
	gdec := gob.NewDecoder(bytes.NewReader(buf))

	var value interface{}
	err := gdec.Decode(&value)
	if err != nil {
		return nil, err
	}

	return value, nil
}
