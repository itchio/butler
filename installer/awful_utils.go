package installer

import "strings"

// cf. https://blogs.msdn.microsoft.com/oldnewthing/20100726-00/?p=13323
func HasSuspiciouslySetupLikeName(name string) bool {
	lowerName := strings.ToLower(name)
	var badStrings = []string{"setup", "install"}
	for _, bs := range badStrings {
		if strings.Contains(lowerName, bs) {
			return true
		}
	}
	return false
}
