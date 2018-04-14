package werrors

import "errors"

// Returned by a function when a context is cancelled before
// we could finish
var ErrCancelled = errors.New("cancelled")
