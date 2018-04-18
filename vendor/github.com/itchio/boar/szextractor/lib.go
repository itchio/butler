package szextractor

import (
	"sync"

	"github.com/itchio/sevenzip-go/sz"
	"github.com/itchio/wharf/state"
)

var _libMutex sync.Mutex
var _lib *sz.Lib
var _libErr error
var _loadOnce sync.Once

func GetLib(consumer *state.Consumer) (*sz.Lib, error) {
	_libMutex.Lock()
	defer _libMutex.Unlock()

	_loadOnce.Do(func() {
		_libErr = EnsureDeps(consumer)
		if _libErr != nil {
			return
		}

		_lib, _libErr = sz.NewLib()
	})

	if _libErr != nil {
		return nil, _libErr
	}
	return _lib, nil
}
