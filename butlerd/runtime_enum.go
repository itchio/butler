package butlerd

import "github.com/itchio/butler/manager"

func (rc *RequestContext) RuntimeEnumerator() manager.RuntimeEnumerator {
	return manager.DefaultRuntimeEnumerator()
}
