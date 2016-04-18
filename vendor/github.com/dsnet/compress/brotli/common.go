// Copyright 2015, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

// Package brotli implements the Brotli compressed data format.
package brotli

import "runtime"

// Error is the wrapper type for errors specific to this library.
type Error string

func (e Error) Error() string { return "brotli: " + string(e) }

var (
	ErrCorrupt error = Error("stream is corrupted")
)

func errRecover(err *error) {
	switch ex := recover().(type) {
	case nil:
		// Do nothing.
	case runtime.Error:
		panic(ex)
	case error:
		*err = ex
	default:
		panic(ex)
	}
}

var (
	reverseLUT [256]uint8
	mtfLUT     [256]uint8
)

func init() {
	initLUTs()
	printLUTs() // Only occurs in debug mode
}

func initLUTs() {
	initCommonLUTs()
	initPrefixLUTs()
	initContextLUTs()
	initDictLUTs()
}

func initCommonLUTs() {
	for i := range mtfLUT {
		mtfLUT[i] = uint8(i)
	}
	for i := range reverseLUT {
		b := uint8(i)
		b = (b&0xaa)>>1 | (b&0x55)<<1
		b = (b&0xcc)>>2 | (b&0x33)<<2
		b = (b&0xf0)>>4 | (b&0x0f)<<4
		reverseLUT[i] = b
	}
}

// neededBits computes the minimum number of bits needed to encode n elements.
func neededBits(n uint32) (nb uint) {
	for n -= 1; n > 0; n >>= 1 {
		nb++
	}
	return
}

// reverseUint32 reverses all bits of v.
func reverseUint32(v uint32) (x uint32) {
	x |= uint32(reverseLUT[byte(v>>0)]) << 24
	x |= uint32(reverseLUT[byte(v>>8)]) << 16
	x |= uint32(reverseLUT[byte(v>>16)]) << 8
	x |= uint32(reverseLUT[byte(v>>24)]) << 0
	return x
}

// reverseBits reverses the lower n bits of v.
func reverseBits(v uint32, n uint) uint32 {
	return reverseUint32(v << (32 - n))
}

// moveToFront is a data structure that allows for more efficient move-to-front
// transformations (described in RFC section 7.3). Since most transformations
// only involve a fairly low number of symbols, it can be quite expensive
// filling out the dict with values 0..255 for every call. Instead, we remember
// what part of the dict was altered and make sure we reset it at the beginning
// of every encode and decode operation.
type moveToFront struct {
	dict [256]uint8 // Mapping from indexes to values
	tail int        // Number of tail bytes that are already ordered
}

func (m *moveToFront) Encode(vals []uint8) {
	// Reset dict to be identical to mtfLUT.
	copy(m.dict[:], mtfLUT[:256-m.tail])

	var max int
	for i, val := range vals {
		var idx uint8 // Reverse lookup idx in dict
		for di, dv := range m.dict {
			if dv == val {
				idx = uint8(di)
				break
			}
		}
		vals[i] = idx

		max |= int(idx)
		copy(m.dict[1:], m.dict[:idx])
		m.dict[0] = val
	}
	m.tail = 256 - max - 1
}

func (m *moveToFront) Decode(idxs []uint8) {
	// Reset dict to be identical to mtfLUT.
	copy(m.dict[:], mtfLUT[:256-m.tail])

	var max int
	for i, idx := range idxs {
		val := m.dict[idx] // Forward lookup val in dict
		idxs[i] = val

		max |= int(idx)
		copy(m.dict[1:], m.dict[:idx])
		m.dict[0] = val
	}
	m.tail = 256 - max - 1
}

func allocUint8s(s []uint8, n int) []uint8 {
	if cap(s) >= n {
		return s[:n]
	}
	return make([]uint8, n, n*3/2)
}

func allocUint32s(s []uint32, n int) []uint32 {
	if cap(s) >= n {
		return s[:n]
	}
	return make([]uint32, n, n*3/2)
}

func extendSliceUints32s(s [][]uint32, n int) [][]uint32 {
	if cap(s) >= n {
		return s[:n]
	}
	ss := make([][]uint32, n, n*3/2)
	copy(ss, s[:cap(s)])
	return ss
}

func extendDecoders(s []prefixDecoder, n int) []prefixDecoder {
	if cap(s) >= n {
		return s[:n]
	}
	ss := make([]prefixDecoder, n, n*3/2)
	copy(ss, s[:cap(s)])
	return ss
}
