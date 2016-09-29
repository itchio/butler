package wsync

// NewBlockLibrary returns a new block library containing
// all the given hashes, for fast lookup later.
func NewBlockLibrary(hashes []BlockHash) *BlockLibrary {
	// A single β-hash may correlate with many unique hashes.
	hashLookup := make(map[uint32][]BlockHash)

	for _, hash := range hashes {
		key := hash.WeakHash
		if hashLookup[key] == nil {
			hashLookup[key] = []BlockHash{hash}
		} else {
			hashLookup[key] = append(hashLookup[key], hash)
		}
	}

	return &BlockLibrary{hashLookup}
}
