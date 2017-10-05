package configure

import (
	"runtime"
	"time"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/configurator"
	"github.com/itchio/butler/mansion"
)

var args = struct {
	path       *string
	showSpell  *bool
	osFilter   *string
	archFilter *string
	noFilter   *bool
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("configure", "(Advanced) Look for launchables in a directory").Hidden()
	args.path = cmd.Arg("path", "The directory to configure").Required().String()
	args.showSpell = cmd.Flag("show-spell", "Show spell for all targets").Bool()
	args.osFilter = cmd.Flag("os-filter", "OS filter").Default(runtime.GOOS).String()
	args.archFilter = cmd.Flag("arch-filter", "Architecture filter").Default(runtime.GOARCH).String()
	args.noFilter = cmd.Flag("no-filter", "Do not filter at all").Bool()
	ctx.Register(cmd, do)
}

type Params struct {
	Path       string
	ShowSpell  bool
	OsFilter   string
	ArchFilter string
	NoFilter   bool
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(&Params{
		Path:       *args.path,
		ShowSpell:  *args.showSpell,
		OsFilter:   *args.osFilter,
		ArchFilter: *args.archFilter,
		NoFilter:   *args.noFilter,
	}))
}

func Do(params *Params) error {
	root := params.Path

	startTime := time.Now()

	comm.Opf("Collecting initial candidates")

	verdict, err := configurator.Configure(root, params.ShowSpell)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	comm.Statf("Initial candidates are:\n%s", verdict)

	fixedExecs, err := verdict.FixPermissions(false /* not dry run */)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	if len(fixedExecs) > 0 {
		comm.Statf("Fixed permissions of %d executables:", len(fixedExecs))
		for _, fixedExec := range fixedExecs {
			comm.Logf("  - %s", fixedExec)
		}
	}

	if params.NoFilter {
		comm.Opf("Not filtering, by request")
	} else {
		comm.Opf("Filtering for os %s, arch %s", params.OsFilter, params.ArchFilter)

		verdict.FilterPlatform(params.OsFilter, params.ArchFilter)
	}

	comm.Statf("Configured in %s", time.Since(startTime))

	comm.ResultOrPrint(verdict, func() {
		comm.Statf("Final candidates are:\n%s", verdict)
	})

	return nil
}
