package butlerd

// @name Nest.GetRunningVersion
// @category Utilities
// @caller client
type NestGetRunningVersionParams struct{}

func (p NestGetRunningVersionParams) Validate() error {
	return nil
}

type NestGetRunningVersionResult struct {
	Version     int64
	UserVersion string
	BuildID     int64
}

type NestUpload struct {
	ID       int64
	Filename string
}
