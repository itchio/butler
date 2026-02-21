package models

import (
	"encoding/json"

	"github.com/itchio/dash"
	"github.com/pkg/errors"
)

type JSON string

func unmarshalJSON(in JSON, out interface{}, what string, allowEmpty bool) error {
	contents := []byte(in)
	if allowEmpty {
		if len(contents) == 0 || string(contents) == "null" {
			return nil
		}
	}

	err := json.Unmarshal(contents, out)
	if err != nil {
		return errors.Wrapf(err, "unmarshalling %s", what)
	}

	return nil
}

func marshalJSON(in interface{}, out *JSON, what string) error {
	contents, err := json.Marshal(in)
	if err != nil {
		return errors.Wrapf(err, "marshalling %s", what)
	}

	*out = JSON(contents)
	return nil
}

func UnmarshalJSON(in JSON, out interface{}, what string) error {
	return unmarshalJSON(in, out, what, false)
}

func UnmarshalJSONAllowEmpty(in JSON, out interface{}, what string) error {
	return unmarshalJSON(in, out, what, true)
}

func MarshalJSON(in interface{}, out *JSON, what string) error {
	return marshalJSON(in, out, what)
}

// Verdict

func UnmarshalVerdict(in JSON) (*dash.Verdict, error) {
	var out dash.Verdict
	err := UnmarshalJSON(in, &out, "verdict")
	if err != nil {
		return nil, err
	}

	return &out, nil
}

func MarshalVerdict(in *dash.Verdict, out *JSON) error {
	return MarshalJSON(in, out, "verdict")
}
