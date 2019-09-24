package configure

import (
	"runtime"
	"sort"
	"time"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/filtering"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/dash"
	"github.com/itchio/headway/state"
	"github.com/pkg/errors"
)

var args = struct {
	path       string
	showSpell  bool
	osFilter   string
	archFilter string
	noFilter   bool
	showStats  bool
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("configure", "(Advanced) Look for launchables in a directory").Hidden()
	cmd.Arg("path", "The directory to configure").Required().StringVar(&args.path)
	cmd.Flag("show-spell", "Show spell for all targets").BoolVar(&args.showSpell)
	cmd.Flag("os-filter", "OS filter").Default(runtime.GOOS).StringVar(&args.osFilter)
	cmd.Flag("arch-filter", "Architecture filter").Default(runtime.GOARCH).StringVar(&args.archFilter)
	cmd.Flag("no-filter", "Do not filter at all").BoolVar(&args.noFilter)
	cmd.Flag("show-stats", "Show configure stats (how many files were sniffed, their extensions)").BoolVar(&args.showStats)
	ctx.Register(cmd, do)
}

type Params struct {
	Path       string
	ShowSpell  bool
	OsFilter   string
	ArchFilter string
	NoFilter   bool
	ShowStats  bool
	Consumer   *state.Consumer
}

func do(ctx *mansion.Context) {
	verdict, err := Do(Params{
		Path:       args.path,
		ShowSpell:  args.showSpell,
		OsFilter:   args.osFilter,
		ArchFilter: args.archFilter,
		NoFilter:   args.noFilter,
		ShowStats:  args.showStats,
		Consumer:   comm.NewStateConsumer(),
	})
	ctx.Must(err)

	comm.ResultOrPrint(verdict, func() {
		comm.Statf("Final candidates are:\n%s", verdict)
	})
}

func Do(params Params) (*dash.Verdict, error) {
	consumer := params.Consumer

	root := params.Path

	startTime := time.Now()

	var stats *dash.VerdictStats
	if params.ShowStats {
		stats = &dash.VerdictStats{}
	}

	verdict, err := dash.Configure(root, dash.ConfigureParams{
		Consumer: consumer,
		Filter:   filtering.FilterPaths,
		Stats:    stats,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if !params.ShowSpell {
		for _, c := range verdict.Candidates {
			c.Spell = nil
		}
	}

	fixedExecs, err := dash.FixPermissions(verdict, dash.FixPermissionsParams{
		Consumer: consumer,
		DryRun:   false,
	})
	if err != nil {
		return nil, errors.WithStack(err)
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
		v2 := verdict.Filter(consumer, dash.FilterParams{
			OS:   params.OsFilter,
			Arch: params.ArchFilter,
		})
		verdict = &v2
	}
	consumer.Statf("Configured in %s", time.Since(startTime))

	if params.ShowStats {
		consumer.Statf("%d total sniffs", stats.NumSniffs)

		var sniffs []Sniff
		for ext, num := range stats.SniffsByExt {
			sniffs = append(sniffs, Sniff{ext, num})
		}
		sort.Stable(byNum(sniffs))

		for _, sniff := range sniffs {
			consumer.Infof("- %d sniffs for (%s) files", sniff.num, sniff.ext)
		}
	}

	return verdict, nil
}

type Sniff struct {
	ext string
	num int
}

type byNum []Sniff

func (s byNum) Len() int {
	return len(s)
}
func (s byNum) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s byNum) Less(i, j int) bool {
	return s[i].num > s[j].num
}
