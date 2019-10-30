package butlerd

import "github.com/itchio/butler/manager"

func (rc *RequestContext) HostEnumerator() manager.HostEnumerator {
	return manager.DefaultHostEnumerator()
}
