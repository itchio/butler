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

// registry utilities

const itchPlayerRegistryKey = `SOFTWARE\itch\Sandbox`

func getItchPlayerData(name string) (string, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, itchPlayerRegistryKey, registry.QUERY_VALUE)
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
