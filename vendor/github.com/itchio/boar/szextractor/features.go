package szextractor

import (
	"strings"

	"github.com/itchio/savior"
)

var KnownFeatures = struct {
	None     savior.ExtractorFeatures
	Rar      savior.ExtractorFeatures
	SevenZip savior.ExtractorFeatures
	Generic  savior.ExtractorFeatures
}{
	None: savior.ExtractorFeatures{
		Name:          "sz::unknown",
		Preallocate:   false,
		ResumeSupport: savior.ResumeSupportNone,
		RandomAccess:  false,
	},
	Rar: savior.ExtractorFeatures{
		Name:        "sz::rar",
		Preallocate: true,
		// has interleaved blocks
		ResumeSupport: savior.ResumeSupportNone,
		// file metadata is mixed with blocks, no "central directory"
		RandomAccess: false,
	},
	SevenZip: savior.ExtractorFeatures{
		Name:        "sz::7z",
		Preallocate: true,
		// has interleaved blocks
		ResumeSupport: savior.ResumeSupportNone,
		// has "header" (at end of file) with list of files
		// see http://www.romvault.com/Understanding7z.pdf
		RandomAccess: true,
	},
	Generic: savior.ExtractorFeatures{
		Name:          "sz::generic",
		Preallocate:   true,
		RandomAccess:  true,
		ResumeSupport: savior.ResumeSupportEntry,
	},
}

// Query extractor features by file extension.
// ext must include the dot, for example ".rar"
func FeaturesByExtension(ext string) savior.ExtractorFeatures {
	switch strings.ToLower(ext) {
	case "":
		return KnownFeatures.None
	case ".rar":
		return KnownFeatures.Rar
	case ".7z":
		return KnownFeatures.SevenZip
	default:
		return KnownFeatures.Generic
	}
}

// Query extractor features by format (as reported by 7-zip)
func FeaturesByFormat(format string) savior.ExtractorFeatures {
	switch strings.ToLower(format) {
	case "":
		return KnownFeatures.None
	case "rar", "rar5":
		return KnownFeatures.Rar
	case "7z":
		return KnownFeatures.SevenZip
	default:
		return KnownFeatures.Generic
	}
}
