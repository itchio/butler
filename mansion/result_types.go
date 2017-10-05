package mansion

// WalkResult is sent for each item that's walked
//
// For command `walk`
type WalkResult struct {
	Type string `json:"type"`
	Path string `json:"path,omitempty"`
	Size int64  `json:"size,omitempty"`
}

// A ContainerResult is sent in json mode by the file command
//
// For command `file`
type ContainerResult struct {
	Type             string   `json:"type"`
	Spell            []string `json:"spell"`
	NumFiles         int      `json:"numFiles"`
	NumDirs          int      `json:"numDirs"`
	NumSymlinks      int      `json:"numSymlinks"`
	UncompressedSize int64    `json:"uncompressedSize"`
}

// FileExtractedResult is sent as json so the consumer can know what we extracted
// It is sent even if we're resuming an extract.
//
// For command `unzip`
type FileExtractedResult struct {
	Type string `json:"type"`
	Path string `json:"path"`
}

// FileMirroredResult is sent as json so the consumer can know what we mirrored
//
// For command `ditto`
type FileMirroredResult struct {
	Type string `json:"type"`
	Path string `json:"path"`
}

// ExePropsResult contains the architecture of a binary file
//
// For command `exeprops`
type ExePropsResult struct {
	Arch string `json:"arch"`
}

// ElfPropsResult contains the architecture of a binary file, and
// optionally a list of libraries it depends on
//
// For command `elfprops`
type ElfPropsResult struct {
	Arch      string   `json:"arch"`
	Libraries []string `json:"libraries"`
}
