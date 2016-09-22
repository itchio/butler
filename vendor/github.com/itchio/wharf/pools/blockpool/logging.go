package blockpool

import "github.com/itchio/wharf/tlc"

// A PrintfFunc can be provided to a LoggingSink
type PrintfFunc func(fmt string, args ...interface{})

// A LoggingSink logs all store attempts and throws them away
type LoggingSink struct {
	Container *tlc.Container
	Printf    PrintfFunc
}

var _ Sink = (*LoggingSink)(nil)

// Store prints something then throws it away
func (ls *LoggingSink) Store(loc BlockLocation, data []byte) error {
	if ls.Printf != nil {
		ls.Printf("storing %v", loc)
	}
	return nil
}

// GetContainer returns the associated tlc container
func (ls *LoggingSink) GetContainer() *tlc.Container {
	return ls.Container
}

// Clone actually returns the same logging sink since there's nothing to clone
func (ls *LoggingSink) Clone() Sink {
	return ls
}
