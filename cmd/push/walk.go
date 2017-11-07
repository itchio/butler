package push

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/filtering"
	"github.com/itchio/wharf/pools"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wsync"
)

type walkResult struct {
	container *tlc.Container
	pool      wsync.Pool
}

func doWalk(path string, out chan walkResult, errs chan error, fixPerms bool, dereference bool) {
	container, err := tlc.WalkAny(path, &tlc.WalkOpts{
		Filter:      filtering.FilterPaths,
		Dereference: dereference,
	})
	if err != nil {
		errs <- errors.Wrap(err, 1)
		return
	}

	pool, err := pools.New(container, path)
	if err != nil {
		errs <- errors.Wrap(err, 1)
		return
	}

	result := walkResult{
		container: container,
		pool:      pool,
	}

	if fixPerms {
		result.container.FixPermissions(result.pool)
	}

	if dereference {
		for _, s := range result.container.Symlinks {
			result.container.Files = append(result.container.Files, &tlc.File{
				Path: s.Path,
			})
		}
	}

	out <- result
}
