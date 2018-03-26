package pelican

// see
// https://msdn.microsoft.com/en-us/library/windows/desktop/dd318693(v=vs.85).aspx
func isLanguageWhitelisted(key string) bool {
	localeID := key[:4]
	primaryLangID := localeID[2:]

	switch primaryLangID {
	// neutral
	case "00":
		return true
	// english
	case "09":
		return true
	}
	return false
}
