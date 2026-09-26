package operate

import (
	"encoding/json"
	"testing"

	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/headway/state"
	"github.com/stretchr/testify/require"
)

func TestLoadSubcontextWithLaunchTargets(t *testing.T) {
	var warnings []string
	oc := &OperationContext{
		consumer: &state.Consumer{
			OnMessage: func(level string, msg string) {
				warnings = append(warnings, msg)
			},
		},
		loaded: make(map[string]struct{}),
	}

	targets := json.RawMessage(`[{"path":"game.love","flavor":"love","engine":{"engine":"love","version":"11.5"}}]`)
	saved := &MetaSubcontext{Data: &InstallParams{
		Upload: &itchio.Upload{ID: 12, LaunchTargets: targets},
		Build:  &itchio.Build{ID: 34},
	}}

	encoded, err := json.Marshal(map[string]interface{}{saved.Key(): saved.GetData()})
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(encoded, &oc.root))

	loaded := NewMetaSubcontext()
	oc.Load(loaded)
	require.Empty(t, warnings)
	require.NotNil(t, loaded.Data.Upload)
	require.EqualValues(t, 12, loaded.Data.Upload.ID)
	require.JSONEq(t, string(targets), string(loaded.Data.Upload.LaunchTargets))
	require.EqualValues(t, 34, loaded.Data.Build.ID)
}
