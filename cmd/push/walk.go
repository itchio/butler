package push

import (
	"context"

	"github.com/itchio/lake"
	"github.com/itchio/lake/pools"
	"github.com/itchio/lake/pools/fspool"
	"github.com/itchio/lake/tlc"

	"github.com/pkg/errors"
)

// poolOptions is shared by every pool opened on the build, so a
// dereferenced walk is read back the same way everywhere.
func poolOptions(dereference bool) pools.Options {
	return pools.Options{Fs: fspool.Options{Dereference: dereference}}
}

type walkResult struct {
	container *tlc.Container
	pool      lake.Pool
}

func doWalk(ctx context.Context, path string, out chan walkResult, errs chan error, fixPerms bool, walkOpts tlc.WalkOpts) {
	container, err := tlc.WalkAny(ctx, path, walkOpts)
	if err != nil {
		errs <- errors.WithStack(err)
		return
	}

	pool, err := pools.NewWithOptions(ctx, container, path, poolOptions(walkOpts.Dereference))
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
			result.pool.Close()
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
