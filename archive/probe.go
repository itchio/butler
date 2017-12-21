package archive

import (
	"fmt"
	"path/filepath"
	"strings"

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
}

func Probe(params *TryOpenParams) (*ArchiveInfo, error) {
	strategy := getStrategy(params.File, params.Consumer)

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

	consumer.Warnf("archive: Unfamiliar extension '%s', handing it to 7-zip hoping for a miracle", ext)
	return ArchiveStrategySevenZip
}

// checkpoint every 1MiB
const standardThreshold = 1 * 1024 * 1024

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
		return tarextractor.New(gzipsource.New(seeksource.FromFile(file), standardThreshold)), nil
	case ArchiveStrategyTarBz2:
		return tarextractor.New(bzip2source.New(seeksource.FromFile(file), standardThreshold)), nil
	case ArchiveStrategySevenZip:
		return szextractor.New(file, consumer)
	}

	return nil, fmt.Errorf("unknown ArchiveStrategy %d", ai.Strategy)
}
