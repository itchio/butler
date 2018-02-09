// +build windows

package winsandbox

import (
	"github.com/go-errors/errors"
	"golang.org/x/sys/windows/registry"
)

type PlayerData struct {
	Username string
	Password string
}

func GetPlayerData() (*PlayerData, error) {
	username, err := getItchPlayerData("username")
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	password, err := getItchPlayerData("password")
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	pd := &PlayerData{
		Username: username,
		Password: password,
	}
	return pd, nil
}

func (pd *PlayerData) Save() error {
	var err error

	err = setItchPlayerData("username", pd.Username)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = setItchPlayerData("password", pd.Password)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}

// registry utilities

const itchPlayerRegistryKey = `SOFTWARE\itch\Sandbox`

func getItchPlayerData(name string) (string, error) {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, itchPlayerRegistryKey, registry.READ)
	if err != nil {
		return "", errors.Wrap(err, 0)
	}

	defer key.Close()

	ret, _, err := key.GetStringValue(name)
	if err != nil {
		return "", errors.Wrap(err, 0)
	}

	return ret, nil
}

func setItchPlayerData(name string, value string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, itchPlayerRegistryKey, registry.WRITE)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	defer key.Close()

	err = key.SetStringValue(name, value)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
