package boar

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/itchio/dash"

	"github.com/itchio/boar/szextractor"
	"github.com/itchio/boar/szextractor/xzsource"
	"github.com/itchio/savior/bzip2source"
	"github.com/itchio/savior/gzipsource"
	"github.com/itchio/savior/seeksource"

	"github.com/itchio/savior/tarextractor"
	"github.com/itchio/savior/zipextractor"
	"github.com/pkg/errors"

	"github.com/itchio/savior"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
)

type Strategy int

const (
	StrategyNone Strategy = 0

	StrategyZip Strategy = 100

	StrategyTar    Strategy = 200
	StrategyTarGz  Strategy = 201
	StrategyTarBz2 Strategy = 202
	StrategyTarXz  Strategy = 203

	StrategySevenZip Strategy = 300
)

func (as Strategy) String() string {
	switch as {
	case StrategyZip:
		return "zip"
	case StrategyTar:
		return "tar"
	case StrategyTarGz:
		return "tar.gz"
	case StrategyTarBz2:
		return "tar.bz2"
	case StrategyTarXz:
		return "tar.xz"
	case StrategySevenZip:
		return "7-zip"
	default:
		return "<no strategy>"
	}
}

type StageTwoStrategy int

const (
	StageTwoStrategyNone StageTwoStrategy = 0

	StageTwoStrategyMojoSetup StageTwoStrategy = 666
)

func (sts StageTwoStrategy) String() string {
	switch sts {
	case StageTwoStrategyNone:
		return "none"
	case StageTwoStrategyMojoSetup:
		return "MojoSetup"
	}
	return "<unknown stage two strategy>"
}

type EntriesLister interface {
	Entries() []*savior.Entry
}

type Info struct {
	Strategy         Strategy
	Features         savior.ExtractorFeatures
	Format           string
	StageTwoStrategy StageTwoStrategy
	PostExtract      []string
}

func (ai *Info) String() string {
	res := ""
	res += fmt.Sprintf("%s (via %s)", ai.Format, ai.Strategy)
	res += fmt.Sprintf(", %s", ai.Features)
	if ai.StageTwoStrategy != StageTwoStrategyNone {
		res += fmt.Sprintf(", stage two: %s", ai.StageTwoStrategy)
		res += fmt.Sprintf(", post-extract: %v", ai.PostExtract)
	}
	return res
}

func Probe(params *ProbeParams) (*Info, error) {
	var strategy Strategy

	if params.Candidate != nil && params.Candidate.Flavor == dash.FlavorNativeLinux {
		// might be a mojosetup installer - if not, we won't know what to do with it
		strategy = StrategyZip
	} else {
		strategy = getStrategy(params.File, params.Consumer)
	}

	if strategy == StrategyNone {
		return nil, ErrUnrecognizedArchiveType
	}

	info := &Info{
		Strategy: strategy,
	}

	// now actually try to open it
	ex, err := info.GetExtractor(params.File, params.Consumer)
	if err != nil {
		return nil, errors.Wrap(err, "getting extractor for file")
	}

	if szex, ok := ex.(szextractor.SzExtractor); ok {
		info.Format = szex.GetFormat()
		preferNative := true
		switch info.Format {
		case "gzip":
			info.Strategy = StrategyTarGz
		case "bzip2":
			info.Strategy = StrategyTarBz2
		case "xz":
			info.Strategy = StrategyTarXz
		case "tar":
			info.Strategy = StrategyTar
		case "zip":
			info.Strategy = StrategyZip
		default:
			preferNative = false
		}

		if preferNative {
			ex, err = info.GetExtractor(params.File, params.Consumer)
			if err != nil {
				return nil, errors.Wrap(err, "getting extractor for file")
			}

			info.Format = info.Strategy.String()
		}
	} else {
		info.Format = info.Strategy.String()
	}
	info.Features = ex.Features()

	var entries []*savior.Entry
	stageTwoStrategy := StageTwoStrategyNone
	if el, ok := ex.(EntriesLister); ok {
		entries = el.Entries()
		if params.OnEntries != nil {
			params.OnEntries(entries)
		}
	}

	if len(entries) > 0 {
		stageTwoMarkers := map[string]StageTwoStrategy{
			"scripts/mojosetup_init.lua":  StageTwoStrategyMojoSetup,
			"scripts/mojosetup_init.luac": StageTwoStrategyMojoSetup,
		}

		for _, e := range entries {
			if strat, ok := stageTwoMarkers[e.CanonicalPath]; ok {
				stageTwoStrategy = strat
				break
			}
		}

		consumer := params.Consumer
		if stageTwoStrategy != StageTwoStrategyNone {
			consumer.Infof("Will apply stage-two strategy %s", stageTwoStrategy)
			switch stageTwoStrategy {
			case StageTwoStrategyMojoSetup:
				info.StageTwoStrategy = stageTwoStrategy

				// Note: canonical paths are slash-separated on all platforms
				// Also, MojoSetup lets folks specify a different data-prefix,
				// but *strongly* suggests staying with the default. The code that
				// follows is probably just one of the many reasons why.
				dataPrefix := "data/"
				var dataFiles []string
				for _, e := range entries {
					if e.Kind == savior.EntryKindFile {
						if strings.HasPrefix(e.CanonicalPath, dataPrefix) {
							dataFiles = append(dataFiles, e.CanonicalPath)
						}
					}
				}

				consumer.Infof("Found %d data files:", len(dataFiles))
				knownSuffixes := []string{
					".tar.gz",
					".tar.bz2",
					".tar.xz",
					".zip",
				}

				for _, df := range dataFiles {
					for _, suffix := range knownSuffixes {
						if strings.HasSuffix(strings.ToLower(df), suffix) {
							info.PostExtract = append(info.PostExtract, df)
						}
					}
				}

				if len(info.PostExtract) > 0 {
					consumer.Infof("Found %d post-extract files: ", len(info.PostExtract))
					for _, pe := range info.PostExtract {
						consumer.Infof("- %s", pe)
					}
				} else {
					consumer.Infof("No post-extract files (crossing fingers)")
				}
			}
		}
	}

	return info, nil
}

func getStrategy(file eos.File, consumer *state.Consumer) Strategy {
	stats, err := file.Stat()
	if err != nil {
		consumer.Warnf("archive: Could not stat file, giving up: %s", err.Error())
		return StrategyNone
	}

	lowerName := strings.ToLower(stats.Name())
	ext := filepath.Ext(lowerName)
	if strings.HasSuffix(lowerName, ".tar"+ext) {
		ext = ".tar" + ext
	}

	switch ext {
	case ".zip":
		return StrategyZip
	case ".tar":
		return StrategyTar
	case ".tar.gz":
		return StrategyTarGz
	case ".tar.bz2":
		return StrategyTarBz2
	case ".tar.xz":
		return StrategyTarXz
	case ".7z", ".rar", ".dmg", ".exe":
		return StrategySevenZip
	}

	return StrategySevenZip
}

func (ai *Info) GetExtractor(file eos.File, consumer *state.Consumer) (savior.Extractor, error) {
	switch ai.Strategy {
	case StrategyZip:
		stats, err := file.Stat()
		if err != nil {
			return nil, errors.Wrap(err, "stat'ing file to open as zip archive")
		}

		ex, err := zipextractor.New(file, stats.Size())
		if err != nil {
			return nil, errors.Wrap(err, "creating zip extractor")
		}
		return ex, nil
	case StrategyTar:
		return tarextractor.New(seeksource.FromFile(file)), nil
	case StrategyTarGz:
		return tarextractor.New(gzipsource.New(seeksource.FromFile(file))), nil
	case StrategyTarBz2:
		return tarextractor.New(bzip2source.New(seeksource.FromFile(file))), nil
	case StrategyTarXz:
		xs, err := xzsource.New(file, consumer)
		if err != nil {
			return nil, errors.Wrap(err, "creating xz extractor")
		}
		return tarextractor.New(xs), nil
	case StrategySevenZip:
		szex, err := szextractor.New(file, consumer)
		if err != nil {
			return nil, errors.Wrap(err, "creating 7-zip extractor")
		}

		// apply blacklist
		switch szex.GetFormat() {
		// cf. https://github.com/itchio/itch/issues/1700
		case "ELF":
			return nil, fmt.Errorf("won't extract ELF executable")
		case "PE":
			return nil, fmt.Errorf("won't extract PE executable")
		default:
			return szex, nil
		}
	}

	return nil, fmt.Errorf("unknown Strategy %d", ai.Strategy)
}
