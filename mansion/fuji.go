package mansion

import (
	"sync"

	"github.com/itchio/smaug/fuji"
)

var _fujiInstance fuji.Instance
var _fujiInstanceError error
var _createFujiInstanceOnce sync.Once

func GetFujiInstance() (fuji.Instance, error) {
	_createFujiInstanceOnce.Do(func() {
		_fujiInstance, _fujiInstanceError = fuji.NewInstance(&fuji.Settings{
			CredentialsRegistryKey: `SOFTWARE\itch\Sandbox`,
		})
	})
	return _fujiInstance, _fujiInstanceError
}
