package archive

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/itchio/butler/configurator"

	"github.com/itchio/butler/archive/szextractor"
	"github.com/itchio/savior/bzip2source"
	"github.com/itchio/savior/gzipsource"
	"github.com/itchio/savior/seeksource"

	"github.com/go-errors/errors"
	"github.com/itchio/savior/tarextractor"
	"github.com/itchio/savior/zipextractor"

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

	ArchiveStrategySevenZip ArchiveStrategy = 300
)

type ArchiveInfo struct {
	Strategy ArchiveStrategy
	Features savior.ExtractorFeatures
	Format   string
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
		return nil, errors.Wrap(err, 0)
	}

	info.Features = ex.Features()
	if szex, ok := ex.(szextractor.SzExtractor); ok {
		info.Format = szex.GetFormat()
	} else {
		info.Format = info.Strategy.String()
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
			return nil, errors.Wrap(err, 0)
		}

		ex, err := zipextractor.New(file, stats.Size())
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
		return ex, nil
	case ArchiveStrategyTar:
		return tarextractor.New(seeksource.FromFile(file)), nil
	case ArchiveStrategyTarGz:
		return tarextractor.New(gzipsource.New(seeksource.FromFile(file))), nil
	case ArchiveStrategyTarBz2:
		return tarextractor.New(bzip2source.New(seeksource.FromFile(file))), nil
	case ArchiveStrategySevenZip:
		szex, err := szextractor.New(file, consumer)
		if err != nil {
			return nil, errors.Wrap(err, 0)
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
