//+build windows

package fuji

import "github.com/pkg/errors"

type instance struct {
	settings *Settings
}

var _ Instance = (*instance)(nil)

func NewInstance(settings *Settings) (Instance, error) {
	if settings.CredentialsRegistryKey == "" {
		return nil, errors.Errorf("CredentialsRegistryKey cannot be empty")
	}

	i := &instance{
		settings: settings,
	}
	return i, nil
}

func (i *instance) Settings() *Settings {
	return i.settings
}
