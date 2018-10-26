package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

type MacLaunchTarget struct {
	Path        string
	IsAppBundle bool
}

func (t *MacLaunchTarget) String() string {
	kind := "naked executable"
	if t.IsAppBundle {
		kind = "app bundle"
	}
	return fmt.Sprintf("(%s) [%s]", t.Path, kind)
}

// PrepareMacLaunchTarget looks at a path and tries to figure out if
// it's a mac app bundle, an executable inside of a mac app bundle,
// or just as naked executable.
func PrepareMacLaunchTarget(params RunnerParams) (*MacLaunchTarget, error) {
	consumer := params.Consumer

	target := &MacLaunchTarget{
		Path: params.FullTargetPath,
	}

	stats, err := os.Stat(params.FullTargetPath)
	if err != nil {
		return nil, errors.WithMessage(err, "while preparing mac launch target")
	}

	if stats.IsDir() {
		if PathLooksLikeAppBundle(target.Path) {
			consumer.Infof("(%s) is a directory and ends with .app - looks like an app bundle alright.", target.Path)
			target.IsAppBundle = true
			return target, nil
		}
		return nil, errors.New("(%s) is a directory but does not in .app - doesn't look like an app bundle")
	}

	{
		currentPath := target.Path
		for currentPath != params.InstallFolder {
			nextPath := filepath.Dir(currentPath)
			if nextPath == currentPath {
				break
			}

			if PathLooksLikeAppBundle(nextPath) {
				target.Path = nextPath
				target.IsAppBundle = true
				consumer.Infof("(%s) looks like the real bundle, using that.", target.Path)
				return target, nil
			}
			currentPath = nextPath
		}
	}

	consumer.Infof("(%s) assumed naked executable (not an app bundle and not contained in an app bundle)", target.Path)
	return target, nil
}

func PathLooksLikeAppBundle(dir string) bool {
	return strings.HasSuffix(strings.ToLower(dir), ".app")
}
