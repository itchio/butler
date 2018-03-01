package models

import (
	"encoding/json"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/configurator"
	itchio "github.com/itchio/go-itchio"
)

type JSON string

// Game

func UnmarshalGame(in JSON) (*itchio.Game, error) {
	var out itchio.Game
	err := json.Unmarshal([]byte(in), &out)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return &out, nil
}

func MarshalGame(in *itchio.Game, out *JSON) error {
	contents, err := json.Marshal(in)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	*out = JSON(contents)
	return nil
}

// Upload

func UnmarshalUpload(in JSON) (*itchio.Upload, error) {
	var out itchio.Upload
	err := json.Unmarshal([]byte(in), &out)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return &out, nil
}

func MarshalUpload(in *itchio.Upload, out *JSON) error {
	contents, err := json.Marshal(in)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	*out = JSON(contents)
	return nil
}

// Build

func UnmarshalBuild(in JSON) (*itchio.Build, error) {
	var out itchio.Build
	err := json.Unmarshal([]byte(in), &out)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return &out, nil
}

func MarshalBuild(in *itchio.Build, out *JSON) error {
	contents, err := json.Marshal(in)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	*out = JSON(contents)
	return nil
}

// User

func UnmarshalUser(in JSON) (*itchio.User, error) {
	var out itchio.User
	err := json.Unmarshal([]byte(in), &out)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return &out, nil
}

func MarshalUser(in *itchio.User, out *JSON) error {
	contents, err := json.Marshal(in)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	*out = JSON(contents)
	return nil
}

// Verdict

func UnmarshalVerdict(in JSON) (*configurator.Verdict, error) {
	var out configurator.Verdict
	err := json.Unmarshal([]byte(in), &out)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return &out, nil
}

func MarshalVerdict(in *configurator.Verdict, out *JSON) error {
	contents, err := json.Marshal(in)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	*out = JSON(contents)
	return nil
}
