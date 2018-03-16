package configure

import (
	"runtime"
	"time"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/configurator"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/wharf/state"
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
	Consumer   *state.Consumer
}

func do(ctx *mansion.Context) {
	verdict, err := Do(&Params{
		Path:       *args.path,
		ShowSpell:  *args.showSpell,
		OsFilter:   *args.osFilter,
		ArchFilter: *args.archFilter,
		NoFilter:   *args.noFilter,
		Consumer:   comm.NewStateConsumer(),
	})
	ctx.Must(err)

	comm.ResultOrPrint(verdict, func() {
		comm.Statf("Final candidates are:\n%s", verdict)
	})
}

func Do(params *Params) (*configurator.Verdict, error) {
	consumer := params.Consumer

	root := params.Path

	startTime := time.Now()

	verdict, err := configurator.Configure(root, params.ShowSpell)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	fixedExecs, err := verdict.FixPermissions(false /* not dry run */)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	if len(fixedExecs) > 0 {
		consumer.Statf("Fixed permissions of %d executables:", len(fixedExecs))
		for _, fixedExec := range fixedExecs {
			consumer.Logf("  - %s", fixedExec)
		}
	}

	if params.NoFilter {
		consumer.Opf("Not filtering, by request")
	} else {
		verdict.FilterPlatform(params.OsFilter, params.ArchFilter)
	}
	consumer.Statf("Configured in %s", time.Since(startTime))

	return verdict, nil
}
