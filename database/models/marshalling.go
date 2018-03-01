package models

import (
	"encoding/json"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/configurator"
)

type JSON string

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
