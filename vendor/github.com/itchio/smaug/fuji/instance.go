package fuji

import "github.com/itchio/wharf/state"

type Settings struct {
	// CredentialsRegistryKey is the path of a key under HKEY_CURRENT_USER
	// itch uses `SOFTWARE\itch\Sandbox`.
	CredentialsRegistryKey string
}

type Credentials struct {
	Username string
	Password string
}

type Instance interface {
	Settings() *Settings
	Check(consumer *state.Consumer) error
	Setup(consumer *state.Consumer) error
	GetCredentials() (*Credentials, error)
}
