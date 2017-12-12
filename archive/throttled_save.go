package archive

import "time"

type ThrottledSaveFunc func(state interface{}, force bool)

func ThrottledSave(params *ExtractParams) ThrottledSaveFunc {
	lastSave := time.Now()
	interval := 1 * time.Second

	return func(state interface{}, force bool) {
		if force || time.Since(lastSave) >= interval {
			params.Consumer.Infof("saving state...")

			lastSave = time.Now()
			err := params.Save(state)
			if err != nil {
				params.Consumer.Warnf("could not save state (ignoring): %s", err.Error())
			}
		}
	}
}
