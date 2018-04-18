package boar

import (
	"path"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/itchio/savior"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/eos/option"
	"github.com/itchio/wharf/state"
)

func (ai *Info) ApplyStageTwo(consumer *state.Consumer, aRes *savior.ExtractorResult, installFolder string) (*savior.ExtractorResult, error) {
	switch ai.StageTwoStrategy {
	case StageTwoStrategyMojoSetup:
		return ai.applyMojoSetupStageTwo(consumer, aRes, installFolder)
	}

	consumer.Infof("No stage-two strategy to apply, all is well.")
	return aRes, nil
}

func (ai *Info) applyMojoSetupStageTwo(consumer *state.Consumer, aRes *savior.ExtractorResult, installFolder string) (*savior.ExtractorResult, error) {
	if len(ai.PostExtract) == 0 {
		consumer.Infof("No post-extract for mojosetup stage two")
	}

	for _, pe := range ai.PostExtract {
		err := func() error {
			absolutePath := filepath.Join(installFolder, pe)
			file, err := eos.Open(absolutePath, option.WithConsumer(consumer))
			if err != nil {
				return errors.Wrap(err, "opening stage-two file")
			}
			defer file.Close()

			Info, err := Probe(&ProbeParams{
				Consumer: consumer,
				File:     file,
			})
			if err != nil {
				return errors.Wrap(err, "probing stage-two file")
			}
			consumer.Infof("✓ Post-extract is a supported archive format (%s)", Info.Format)

			ex, err := Info.GetExtractor(file, consumer)
			if err != nil {
				return errors.Wrap(err, "getting extractor for stage-two file")
			}

			sink := &savior.FolderSink{
				Consumer:  consumer,
				Directory: filepath.Dir(absolutePath),
			}
			consumer.Infof(`Extracting (%s)`, absolutePath)
			consumer.Infof(`... to (%s)`, sink.Directory)

			ex.SetConsumer(consumer)

			nestedRes, err := ex.Resume(nil, sink)
			if err != nil {
				return errors.Wrap(err, "extracting stage-two file")
			}

			err = sink.Close()
			if err != nil {
				return errors.Wrap(err, "closing sink after stage-two extraction")
			}

			for _, ne := range nestedRes.Entries {
				ne.CanonicalPath = path.Join(path.Dir(pe), ne.CanonicalPath)
				aRes.Entries = append(aRes.Entries, ne)
			}

			consumer.Infof(`Hey everything went fine!`)
			return nil
		}()
		if err != nil {
			return nil, errors.Wrap(err, "handling post-extract file (stage two)")
		}
	}

	return aRes, nil
}
