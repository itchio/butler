// +build windows

package prereqs

import (
	"fmt"
	"regexp"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/state"
	"golang.org/x/sys/windows/registry"
)

var regkeyRegexp = regexp.MustCompile(`^([^\\]+)\\(.*)$`)

func RegistryKeyExists(consumer *state.Consumer, path string) bool {
	matches := regkeyRegexp.FindAllStringSubmatch(path, 1)
	if len(matches) != 1 {
		consumer.Warnf("Could not parse registry key (%s), skipping check...", path)
		return false
	}

	rootKeyName := matches[0][1]
	pathName := matches[0][2]

	rootKey, err := getRootKey(rootKeyName)
	if err != nil {
		consumer.Warnf("%s, skipping check...", err.Error())
		return false
	}

	key, err := registry.OpenKey(rootKey, pathName, registry.QUERY_VALUE)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			// cool, the key does not exist!
			return false
		}

		consumer.Warnf("%s, skipping check...", err.Error())
		return false
	}

	defer key.Close()

	return true
}

func getRootKey(name string) (registry.Key, error) {
	switch name {
	case "HKEY_LOCAL_MACHINE", "HKLM":
		return registry.LOCAL_MACHINE, nil
	case "HKEY_CURRENT_USER", "HKCU":
		return registry.CURRENT_USER, nil
	}

	return 0, fmt.Errorf("Unknown root key (%s)", name)
}
