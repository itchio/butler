package archive

import "time"

// Saves the state if force is true or the interval has passed
// Returns true if the state was actually saved, false if not
type ThrottledSaveFunc func(state interface{}, force bool) bool

func ThrottledSave(params *ExtractParams) ThrottledSaveFunc {
	lastSave := time.Now()
	interval := 1 * time.Second

	return func(state interface{}, force bool) bool {
		if force || time.Since(lastSave) >= interval {
			lastSave = time.Now()
			err := params.Save(state)
			if err != nil {
				params.Consumer.Warnf("could not save state (ignoring): %s", err.Error())
				return false
			}
			return true
		}
		return false
	}
}
