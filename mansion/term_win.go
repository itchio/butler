// +build windows

package mansion

func IsTerminal() bool {
	// no way to tell afaik!
	return true
}
