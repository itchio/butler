package bio

import (
	"bytes"
	"encoding/gob"

	"gopkg.in/itchio/rsync-go.v0"
)

type LogEntry struct {
	Message string
}

type Target struct {
	RepoSpec string
}

type SourceFile struct {
	Path string
	Size uint64
}

type MD5Hash struct {
	Hash []byte
}

type EndOfSources struct{}

type FilePatched struct {
	Path    string
	ApplyTo string
}

type FileAdded struct {
	Path string
}

type FileRemoved struct {
	Path string
}

func init() {
	Register()
}

func Register() {
	gob.Register(LogEntry{})

	gob.Register(Target{})

	gob.Register(SourceFile{})
	gob.Register(MD5Hash{})
	gob.Register(EndOfSources{})

	gob.Register(rsync.BlockHash{})
	gob.Register(rsync.Operation{})

	gob.Register(FilePatched{})
	gob.Register(FileAdded{})
	gob.Register(FileRemoved{})
	gob.Register(rsync.BlockHash{})
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
