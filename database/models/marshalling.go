package models

import (
	"encoding/json"

	"github.com/itchio/dash"
	"github.com/pkg/errors"
)

type JSON string

// Verdict

func UnmarshalVerdict(in JSON) (*dash.Verdict, error) {
	var out dash.Verdict
	err := json.Unmarshal([]byte(in), &out)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshalling verdict")
	}

	return &out, nil
}

func MarshalVerdict(in *dash.Verdict, out *JSON) error {
	contents, err := json.Marshal(in)
	if err != nil {
		return errors.Wrap(err, "marshalling verdict")
	}
	*out = JSON(contents)
	return nil
}
