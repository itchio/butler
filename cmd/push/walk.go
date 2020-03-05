package push

import (
	"github.com/itchio/lake"
	"github.com/itchio/lake/pools"
	"github.com/itchio/lake/tlc"

	"github.com/pkg/errors"
)

type walkResult struct {
	container *tlc.Container
	pool      lake.Pool
}

func doWalk(path string, out chan walkResult, errs chan error, fixPerms bool, walkOpts tlc.WalkOpts) {
	container, err := tlc.WalkAny(path, walkOpts)
	if err != nil {
		errs <- errors.WithStack(err)
		return
	}

	pool, err := pools.New(container, path)
	if err != nil {
		errs <- errors.WithStack(err)
		return
	}

	result := walkResult{
		container: container,
		pool:      pool,
	}

	if fixPerms {
		err := result.container.FixPermissions(result.pool)
		if err != nil {
			errs <- errors.WithStack(err)
			return
		}
	}

	if walkOpts.Dereference {
		for _, s := range result.container.Symlinks {
			result.container.Files = append(result.container.Files, &tlc.File{
				Path: s.Path,
			})
		}
	}

	out <- result
}
