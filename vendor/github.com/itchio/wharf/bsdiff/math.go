package bsdiff

import (
	"bytes"
	"fmt"

	"github.com/itchio/wharf/state"
)

// Ternary-Split Quicksort, cf. http://www.larsson.dogma.net/ssrev-tr.pdf
func split(I, V []int32, start, length, h int32) {
	var i, j, k, x, jj, kk int32

	if length < 16 {
		for k = start; k < start+length; k += j {
			j = 1
			x = V[I[k]+h]
			for i = 1; k+i < start+length; i++ {
				if V[I[k+i]+h] < x {
					x = V[I[k+i]+h]
					j = 0
				}
				if V[I[k+i]+h] == x {
					I[k+i], I[k+j] = I[k+j], I[k+i]
					j++
				}
			}
			for i = 0; i < j; i++ {
				V[I[k+i]] = k + j - 1
			}
			if j == 1 {
				I[k] = -1
			}
		}
		return
	}

	x = V[I[start+length/2]+h]
	jj = 0
	kk = 0
	for i = start; i < start+length; i++ {
		if V[I[i]+h] < x {
			jj++
		}
		if V[I[i]+h] == x {
			kk++
		}
	}
	jj += start
	kk += jj

	i = start
	j = 0
	k = 0
	for i < jj {
		if V[I[i]+h] < x {
			i++
		} else if V[I[i]+h] == x {
			I[i], I[jj+j] = I[jj+j], I[i]
			j++
		} else {
			I[i], I[kk+k] = I[kk+k], I[i]
			k++
		}
	}

	for jj+j < kk {
		if V[I[jj+j]+h] == x {
			j++
		} else {
			I[jj+j], I[kk+k] = I[kk+k], I[jj+j]
			k++
		}
	}

	if jj > start {
		split(I, V, start, jj-start, h)
	}

	for i = 0; i < kk-jj; i++ {
		V[I[jj+i]] = kk - 1
	}
	if jj == kk-1 {
		I[jj] = -1
	}

	if start+length > kk {
		split(I, V, kk, start+length-kk, h)
	}
}

// Faster Suffix Sorting, see: http://www.larsson.dogma.net/ssrev-tr.pdf
// Output `I` is a sorted suffix array.
// TODO: implement parallel sorting as a faster alternative for high-RAM environments
// see http://www.zbh.uni-hamburg.de/pubs/pdf/FutAluKur2001.pdf
func qsufsort(obuf []byte, consumer *state.Consumer) []int32 {
	var buckets [256]int32
	var i, h int32
	var obuflen = int32(len(obuf))

	I := make([]int32, obuflen+1)
	V := make([]int32, obuflen+1)

	for _, c := range obuf {
		buckets[c]++
	}
	for i = 1; i < 256; i++ {
		buckets[i] += buckets[i-1]
	}
	copy(buckets[1:], buckets[:])
	buckets[0] = 0

	for i, c := range obuf {
		buckets[c]++
		I[buckets[c]] = int32(i)
	}

	I[0] = obuflen
	for i, c := range obuf {
		V[i] = buckets[c]
	}

	V[obuflen] = 0
	for i = 1; i < 256; i++ {
		if buckets[i] == buckets[i-1]+1 {
			I[buckets[i]] = -1
		}
	}
	I[0] = -1

	const progressInterval = 64 * 1024

	for h = 1; I[0] != -(obuflen + 1); h += h {
		consumer.ProgressLabel(fmt.Sprintf("Suffix sorting (%d-order)", h))

		var lastI int32
		var n int32

		for i = 0; i < obuflen+1; {
			if i-lastI > progressInterval {
				progress := float64(i) / float64(obuflen)
				consumer.Progress(progress)
				lastI = i
			}

			if I[i] < 0 {
				n -= I[i]
				i -= I[i]
			} else {
				if n != 0 {
					I[i-n] = -n
				}
				n = V[I[i]] + 1 - i
				split(I, V, i, n, h)
				i += n
				n = 0
			}
		}
		if n != 0 {
			I[i-n] = -n
		}
	}

	for i = 0; i < obuflen+1; i++ {
		I[V[i]] = i
	}
	return I
}

// Returns the number of bytes common to a and b
func matchlen(a, b []byte) (i int32) {
	alen := int32(len(a))
	blen := int32(len(b))
	for i < alen && i < blen && a[i] == b[i] {
		i++
	}
	return i
}

func search(I []int32, obuf, nbuf []byte, st, en int32) (pos, n int32) {
	if en-st < 2 {
		x := matchlen(obuf[I[st]:], nbuf)
		y := matchlen(obuf[I[en]:], nbuf)

		if x > y {
			return I[st], x
		}
		return I[en], y
	}

	x := st + (en-st)/2
	if bytes.Compare(obuf[I[x]:], nbuf) < 0 {
		return search(I, obuf, nbuf, x, en)
	}
	return search(I, obuf, nbuf, st, x)
}
