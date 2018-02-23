package buse

import (
	itchio "github.com/itchio/go-itchio"
)

type Harness interface {
	ClientFromCredentials(credentials *GameCredentials) (*itchio.Client, error)
}
