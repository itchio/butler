package wire

import "encoding/binary"

// Endianness is the byte order of everything in wharf's wire format
var Endianness = binary.LittleEndian

// DebugWire controls debug printouts of wharf's wire format
const DebugWire = false
