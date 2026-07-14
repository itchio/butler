package butlerd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_CavesSetSettingsParams_Validate_Type(t *testing.T) {
	require := require.New(t)

	invalidType := SandboxType("bogus")
	params := CavesSetSettingsParams{
		CaveID: "cave-1",
		Settings: &CaveSettings{
			SandboxType: &invalidType,
		},
	}

	err := params.Validate()
	require.Error(err)
	require.Contains(err.Error(), "settings.sandboxType")
}

func Test_CavesSetSettingsParams_Validate_Auto(t *testing.T) {
	require := require.New(t)

	autoType := SandboxTypeAuto
	params := CavesSetSettingsParams{
		CaveID: "cave-1",
		Settings: &CaveSettings{
			SandboxType: &autoType,
		},
	}

	err := params.Validate()
	require.NoError(err)
}

func Test_CavesSetSettingsParams_Validate_RequiresSettings(t *testing.T) {
	require := require.New(t)

	params := CavesSetSettingsParams{
		CaveID: "cave-1",
	}

	err := params.Validate()
	require.Error(err)
}

func Test_CavesSetSettingsParams_Validate_CommandTemplate(t *testing.T) {
	require := require.New(t)

	params := CavesSetSettingsParams{
		CaveID: "cave-1",
		Settings: &CaveSettings{
			CommandTemplate: "%command% %command%",
		},
	}

	err := params.Validate()
	require.Error(err)
	require.Contains(err.Error(), "settings.commandTemplate")
}

func Test_LaunchParams_Validate_CommandTemplate(t *testing.T) {
	require := require.New(t)

	params := LaunchParams{
		CaveID:          "cave-1",
		PrereqsDir:      "/tmp/prereqs",
		CommandTemplate: `%command% "unterminated`,
	}

	err := params.Validate()
	require.Error(err)
	require.Contains(err.Error(), "commandTemplate")
}
