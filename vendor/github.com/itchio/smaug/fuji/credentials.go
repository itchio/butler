//+build windows

package fuji

import (
	"github.com/pkg/errors"
	"golang.org/x/sys/windows/registry"
)

func (i *instance) GetCredentials() (*Credentials, error) {
	username, err := getRegistryString(i.settings, "username")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	password, err := getRegistryString(i.settings, "password")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	creds := &Credentials{
		Username: username,
		Password: password,
	}
	return creds, nil
}

func (i *instance) saveCredentials(creds *Credentials) error {
	err := setRegistryString(i.settings, "username", creds.Username)
	if err != nil {
		return errors.WithStack(err)
	}

	err = setRegistryString(i.settings, "password", creds.Password)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// registry utilities

func getRegistryString(s *Settings, name string) (string, error) {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, s.CredentialsRegistryKey, registry.READ)
	if err != nil {
		return "", errors.WithStack(err)
	}

	defer key.Close()

	ret, _, err := key.GetStringValue(name)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return ret, nil
}

func setRegistryString(s *Settings, name string, value string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, s.CredentialsRegistryKey, registry.WRITE)
	if err != nil {
		return errors.WithStack(err)
	}

	defer key.Close()

	err = key.SetStringValue(name, value)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
