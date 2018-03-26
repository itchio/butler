package pelican

import (
	"encoding/binary"
	"unicode/utf16"
)

// Convert a UTF-16 string (as a byte slice) to unicode
func DecodeUTF16(bs []byte) string {
	ints := make([]uint16, len(bs)/2)
	for i := 0; i < len(ints); i++ {
		ints[i] = binary.LittleEndian.Uint16(bs[i*2 : (i+1)*2])
	}
	return string(utf16.Decode(ints))
}
