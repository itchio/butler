package mansion

import (
	"github.com/itchio/smaug/fuji"
)

func GetFujiSettings() *fuji.Settings {
	return &fuji.Settings{
		CredentialsRegistryKey: `SOFTWARE\itch\Sandbox`,
	}
}
