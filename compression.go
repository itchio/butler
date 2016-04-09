package main

import (
	_ "github.com/itchio/wharf/compressors/cbrotli"
	_ "github.com/itchio/wharf/decompressors/cbrotli"

	_ "github.com/itchio/wharf/compressors/gzip"
	_ "github.com/itchio/wharf/decompressors/gzip"
)
