package mansion

// WalkResult is sent for each item that's walked
type WalkResult struct {
	Type string `json:"type"`
	Path string `json:"path,omitempty"`
	Size int64  `json:"size,omitempty"`
}

// A ContainerResult is sent in json mode by the file command
type ContainerResult struct {
	Type             string   `json:"type"`
	Spell            []string `json:"spell"`
	NumFiles         int      `json:"numFiles"`
	NumDirs          int      `json:"numDirs"`
	NumSymlinks      int      `json:"numSymlinks"`
	UncompressedSize int64    `json:"uncompressedSize"`
}
