package archive

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/itchio/butler/configurator"

	"github.com/itchio/butler/archive/szextractor"
	"github.com/itchio/butler/archive/szextractor/xzsource"
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

type ArchiveStrategy int

const (
	ArchiveStrategyNone ArchiveStrategy = 0

	ArchiveStrategyZip ArchiveStrategy = 100

	ArchiveStrategyTar    ArchiveStrategy = 200
	ArchiveStrategyTarGz  ArchiveStrategy = 201
	ArchiveStrategyTarBz2 ArchiveStrategy = 202
	ArchiveStrategyTarXz  ArchiveStrategy = 203

	ArchiveStrategySevenZip ArchiveStrategy = 300
)

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

type ArchiveInfo struct {
	Strategy         ArchiveStrategy
	Features         savior.ExtractorFeatures
	Format           string
	StageTwoStrategy StageTwoStrategy
	PostExtract      []string
}

func Probe(params *TryOpenParams) (*ArchiveInfo, error) {
	var strategy ArchiveStrategy

	if params.Candidate != nil && params.Candidate.Flavor == configurator.FlavorNativeLinux {
		// might be a mojosetup installer - if not, we won't know what to do with it
		strategy = ArchiveStrategyZip
	} else {
		strategy = getStrategy(params.File, params.Consumer)
	}

	if strategy == ArchiveStrategyNone {
		return nil, ErrUnrecognizedArchiveType
	}

	info := &ArchiveInfo{
		Strategy: strategy,
	}

	// now actually try to open it
	ex, err := info.GetExtractor(params.File, params.Consumer)
	if err != nil {
		return nil, errors.Wrap(err, "getting extractor for file")
	}

	info.Features = ex.Features()
	if szex, ok := ex.(szextractor.SzExtractor); ok {
		info.Format = szex.GetFormat()
	} else {
		info.Format = info.Strategy.String()
	}

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

		params.Consumer.Debugf("Scanning %d entries for a stage two marker...", len(entries))
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

func getStrategy(file eos.File, consumer *state.Consumer) ArchiveStrategy {
	stats, err := file.Stat()
	if err != nil {
		consumer.Warnf("archive: Could not stat file, giving up: %s", err.Error())
		return ArchiveStrategyNone
	}

	lowerName := strings.ToLower(stats.Name())
	ext := filepath.Ext(lowerName)
	if strings.HasSuffix(lowerName, ".tar"+ext) {
		ext = ".tar" + ext
	}

	switch ext {
	case ".zip":
		return ArchiveStrategyZip
	case ".tar":
		return ArchiveStrategyTar
	case ".tar.gz":
		return ArchiveStrategyTarGz
	case ".tar.bz2":
		return ArchiveStrategyTarBz2
	case ".tar.xz":
		return ArchiveStrategyTarXz
	case ".7z", ".rar", ".dmg", ".exe":
		return ArchiveStrategySevenZip
	}

	consumer.Warnf("archive: Unrecognized extension (%s), deferring to 7-zip", ext)
	return ArchiveStrategySevenZip
}

func (ai *ArchiveInfo) GetExtractor(file eos.File, consumer *state.Consumer) (savior.Extractor, error) {
	switch ai.Strategy {
	case ArchiveStrategyZip:
		stats, err := file.Stat()
		if err != nil {
			return nil, errors.Wrap(err, "stat'ing file to open as zip archive")
		}

		ex, err := zipextractor.New(file, stats.Size())
		if err != nil {
			return nil, errors.Wrap(err, "creating zip extractor")
		}
		return ex, nil
	case ArchiveStrategyTar:
		return tarextractor.New(seeksource.FromFile(file)), nil
	case ArchiveStrategyTarGz:
		return tarextractor.New(gzipsource.New(seeksource.FromFile(file))), nil
	case ArchiveStrategyTarBz2:
		return tarextractor.New(bzip2source.New(seeksource.FromFile(file))), nil
	case ArchiveStrategyTarXz:
		xs, err := xzsource.New(file, consumer)
		if err != nil {
			return nil, errors.Wrap(err, "creating xz extractor")
		}
		return tarextractor.New(xs), nil
	case ArchiveStrategySevenZip:
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

	return nil, fmt.Errorf("unknown ArchiveStrategy %d", ai.Strategy)
}

var (
	archiveStrategyStrings = map[ArchiveStrategy]string{
		ArchiveStrategyTar:    "tar",
		ArchiveStrategyTarBz2: "tar.bz2",
		ArchiveStrategyTarGz:  "tar.gz",
		ArchiveStrategyZip:    "zip",
	}
)

func (as ArchiveStrategy) String() string {
	str, ok := archiveStrategyStrings[as]
	if !ok {
		return "?"
	}
	return str
}
