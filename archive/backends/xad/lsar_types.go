package xad

// We use lsar in JSON output mode, these structs map to it

type LsarResult struct {
	FormatVersion int64    `json:"lsarFormatVersion"`
	Contents      []*Entry `json:"lsarContents"`
	Encoding      string   `json:"lsarEncoding"`
	Confidence    int64    `json:"lsarConfidence"`
	FormatName    string   `json:"lsarFormatName"`
}

// Entry contains the fields we care about most often
type Entry struct {
	XADFileName string
	XADFileSize int64
}

type FullEntry struct {
	Entry

	XADIndex      int64
	XADPosixGroup int64
	XADPosixUser  int64
	// examples: "None",
	XADCompressionName string
	XADDataLength      int64
	ZipLocalDate       int64

	/////////////////////////////////////////
	// date format: "2016-09-27 02:15:12 +0200"
	/////////////////////////////////////////
	XADLastModificationDate string
	XADLastAccessDate       string

	/////////////////////////////////////////
	// zip-specific entries
	/////////////////////////////////////////
	ZipOS             int64
	ZipOSName         string
	ZipExtractVersion int64
	ZipFlags          int64
	ZipCRC32          int64
	ZipFileAttributes int64
}
