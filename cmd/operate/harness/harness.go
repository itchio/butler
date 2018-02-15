package harness

import (
	"github.com/itchio/butler/buse"
	itchio "github.com/itchio/go-itchio"
)

type Harness interface {
	ClientFromCredentials(credentials *buse.GameCredentials) (*itchio.Client, error)
}
