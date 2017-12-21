package savior

import (
	"io"

	"github.com/go-errors/errors"
)

var StopErr = errors.New("copy was stopped after save!")

type MakeCheckpointFunc func() (*ExtractorCheckpoint, error)
type EmitProgressFunc func()

type CopyResult struct {
	Action AfterSaveAction
}

type CopyParams struct {
	Src   io.Reader
	Dst   io.Writer
	Entry *Entry

	SaveConsumer SaveConsumer

	MakeCheckpoint MakeCheckpointFunc
	EmitProgress   EmitProgressFunc
}

const progressThreshold = 512 * 1024

func CopyWithSaver(params *CopyParams) (*CopyResult, error) {
	if params == nil {
		return nil, errors.New("CopyWithSaver called with nil params")
	}

	if params.SaveConsumer == nil {
		return nil, errors.New("CopyWithSaver called with a nil SaveConsumer")
	}

	buf := make([]byte, 32*1024)
	var progressCounter int64

	for {
		n, readErr := params.Src.Read(buf)

		m, err := params.Dst.Write(buf[:n])
		params.Entry.WriteOffset += int64(m)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		progressCounter += int64(m)
		if progressCounter > progressThreshold {
			progressCounter = 0
			if params.EmitProgress != nil {
				params.EmitProgress()
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				// cool, we're done!
				return &CopyResult{
					Action: AfterSaveContinue,
				}, nil
			}
			return nil, errors.Wrap(err, 0)
		}

		if params.SaveConsumer.ShouldSave(int64(n)) {
			checkpoint, err := params.MakeCheckpoint()
			if err != nil {
				return nil, errors.Wrap(err, 0)
			}

			action, err := params.SaveConsumer.Save(checkpoint)
			if err != nil {
				return nil, errors.Wrap(err, 0)
			}
			if action != AfterSaveContinue {
				return &CopyResult{
					Action: AfterSaveStop,
				}, nil
			}
		}
	}
}
