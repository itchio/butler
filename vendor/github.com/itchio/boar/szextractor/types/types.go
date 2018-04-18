package types

type DepSpecMap map[string]DepSpec

type DepSpec struct {
	Entries []DepEntry
	Sources []string
}

type DepEntry struct {
	Name   string
	Size   int64
	Hashes []DepHash
}

type HashAlgo string

const (
	HashAlgoSHA1   = "sha1"
	HashAlgoSHA256 = "sha256"
)

type DepHash struct {
	Algo HashAlgo
	// byte array formatted with `%x` (lower-case hex)
	Value string
}
