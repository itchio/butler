package crc32c

import "hash/crc32"

// Table is a crc32 table based on the Castagnoli polynomial, and can be used
// to compute CRC32-C hashes, which are used on Google Cloud Storage for example.
var Table = crc32.MakeTable(crc32.Castagnoli)
