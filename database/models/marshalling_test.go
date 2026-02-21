package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_UnmarshalJSONAllowEmpty_EmptyString(t *testing.T) {
	require := require.New(t)

	type payload struct {
		Value string `json:"value"`
	}

	var out payload
	err := UnmarshalJSONAllowEmpty(JSON(""), &out, "payload")
	require.NoError(err)
	require.Equal(payload{}, out)
}

func Test_UnmarshalJSONAllowEmpty_Null(t *testing.T) {
	require := require.New(t)

	type payload struct {
		Value string `json:"value"`
	}

	var out payload
	err := UnmarshalJSONAllowEmpty(JSON("null"), &out, "payload")
	require.NoError(err)
	require.Equal(payload{}, out)
}

func Test_UnmarshalJSONAllowEmpty_Invalid(t *testing.T) {
	require := require.New(t)

	type payload struct {
		Value string `json:"value"`
	}

	var out payload
	err := UnmarshalJSONAllowEmpty(JSON("{"), &out, "payload")
	require.Error(err)
}
