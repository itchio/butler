package pwr

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"runtime"

	humanize "github.com/dustin/go-humanize"
	"github.com/itchio/wharf/state"
	"github.com/itchio/wharf/wire"
)

// MaxBsdiffSize is the largest size bsdiff will diff (for both old and new file)
const MaxBsdiffSize = int64(math.MaxInt32 - 1)

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

// BSDiff computes the difference between old and new, according to the bsdiff
// algorithm, and writes the result to patch.
func BSDiff(old, new io.Reader, patch *wire.WriteContext, consumer *state.Consumer) error {
	var memstats runtime.MemStats
	runtime.ReadMemStats(&memstats)
	consumer.Debugf("Allocated bytes at start of bsdiff: %s (%s total)", humanize.IBytes(uint64(memstats.Alloc)), humanize.IBytes(uint64(memstats.TotalAlloc)))

	obuf, err := ioutil.ReadAll(old)
	if err != nil {
		return err
	}
	if int64(len(obuf)) > MaxBsdiffSize {
		return fmt.Errorf("bsdiff: old file too large (%s > %s)", humanize.IBytes(uint64(len(obuf))), humanize.IBytes(uint64(MaxBsdiffSize)))
	}
	obuflen := int32(len(obuf))

	nbuf, err := ioutil.ReadAll(new)
	if err != nil {
		return err
	}
	if int64(len(nbuf)) > MaxBsdiffSize {
		return fmt.Errorf("bsdiff: new file too large (%s > %s)", humanize.IBytes(uint64(len(nbuf))), humanize.IBytes(uint64(MaxBsdiffSize)))
	}
	nbuflen := int32(len(nbuf))

	runtime.ReadMemStats(&memstats)
	consumer.Debugf("Allocated bytes after ReadAll: %s (%s total)", humanize.IBytes(uint64(memstats.Alloc)), humanize.IBytes(uint64(memstats.TotalAlloc)))

	var lenf int32
	I := qsufsort(obuf, consumer)

	runtime.ReadMemStats(&memstats)
	consumer.Debugf("Allocated bytes after qsufsort: %s (%s total)", humanize.IBytes(uint64(memstats.Alloc)), humanize.IBytes(uint64(memstats.TotalAlloc)))

	// FIXME: the streaming format allows us to allocate less than that
	db := make([]byte, len(nbuf))
	eb := make([]byte, len(nbuf))

	bsdc := &BsdiffControl{}

	consumer.ProgressLabel("Scanning...")

	// Compute the differences, writing ctrl as we go
	var scan, pos, length int32
	var lastscan, lastpos, lastoffset int32
	for scan < nbuflen {
		var oldscore int32
		scan += length

		progress := float64(scan) / float64(nbuflen)
		consumer.Progress(progress)

		for scsc := scan; scan < nbuflen; scan++ {
			pos, length = search(I, obuf, nbuf[scan:], 0, obuflen)

			for ; scsc < scan+length; scsc++ {
				if scsc+lastoffset < obuflen &&
					obuf[scsc+lastoffset] == nbuf[scsc] {
					oldscore++
				}
			}

			if (length == oldscore && length != 0) || length > oldscore+8 {
				break
			}

			if scan+lastoffset < obuflen && obuf[scan+lastoffset] == nbuf[scan] {
				oldscore--
			}
		}

		if length != oldscore || scan == nbuflen {
			var s, Sf int32
			lenf = 0
			for i := int32(0); lastscan+i < scan && lastpos+i < obuflen; {
				if obuf[lastpos+i] == nbuf[lastscan+i] {
					s++
				}
				i++
				if s*2-i > Sf*2-lenf {
					Sf = s
					lenf = i
				}
			}

			lenb := int32(0)
			if scan < nbuflen {
				var s, Sb int32
				for i := int32(1); (scan >= lastscan+i) && (pos >= i); i++ {
					if obuf[pos-i] == nbuf[scan-i] {
						s++
					}
					if s*2-i > Sb*2-lenb {
						Sb = s
						lenb = i
					}
				}
			}

			if lastscan+lenf > scan-lenb {
				overlap := (lastscan + lenf) - (scan - lenb)
				s := int32(0)
				Ss := int32(0)
				lens := int32(0)
				for i := int32(0); i < overlap; i++ {
					if nbuf[lastscan+lenf-overlap+i] == obuf[lastpos+lenf-overlap+i] {
						s++
					}
					if nbuf[scan-lenb+i] == obuf[pos-lenb+i] {
						s--
					}
					if s > Ss {
						Ss = s
						lens = i + 1
					}
				}

				lenf += lens - overlap
				lenb -= lens
			}

			for i := int32(0); i < lenf; i++ {
				db[i] = nbuf[lastscan+i] - obuf[lastpos+i]
			}
			for i := int32(0); i < (scan-lenb)-(lastscan+lenf); i++ {
				eb[i] = nbuf[lastscan+lenf+i]
			}

			bsdc.Add = db[:lenf]
			bsdc.Copy = eb[:(scan-lenb)-(lastscan+lenf)]
			bsdc.Seek = int64((pos - lenb) - (lastpos + lenf))

			err := patch.WriteMessage(bsdc)
			if err != nil {
				return err
			}

			lastscan = scan - lenb
			lastpos = pos - lenb
			lastoffset = pos - scan
		}
	}

	runtime.ReadMemStats(&memstats)
	consumer.Debugf("Allocated bytes after scan: %s (%s total)", humanize.IBytes(uint64(memstats.Alloc)), humanize.IBytes(uint64(memstats.TotalAlloc)))

	// Write sentinel control message
	bsdc.Reset()
	bsdc.Eof = true
	err = patch.WriteMessage(bsdc)
	if err != nil {
		return err
	}

	return nil
}

// ErrCorrupt indicates that a patch is corrupted, most often that it would produce a longer file
// than specified
var ErrCorrupt = errors.New("corrupt patch")

// BSPatch applies patch to old, according to the bspatch algorithm,
// and writes the result to new.
func BSPatch(old io.Reader, new io.Writer, newSize int64, patch *wire.ReadContext) error {
	obuf, err := ioutil.ReadAll(old)
	if err != nil {
		return err
	}

	nbuf := make([]byte, newSize)

	var oldpos, newpos int64

	ctrl := &BsdiffControl{}

	for {
		ctrl.Reset()

		err = patch.ReadMessage(ctrl)
		if err != nil {
			return err
		}

		if ctrl.Eof {
			break
		}

		// Sanity-check
		if newpos+int64(len(ctrl.Add)) > newSize {
			return ErrCorrupt
		}

		// Add old data to diff string
		for i := int64(0); i < int64(len(ctrl.Add)); i++ {
			nbuf[newpos+i] = ctrl.Add[i] + obuf[oldpos+i]
		}

		// Adjust pointers
		newpos += int64(len(ctrl.Add))
		oldpos += int64(len(ctrl.Add))

		// Sanity-check
		if newpos+int64(len(ctrl.Copy)) > newSize {
			return ErrCorrupt
		}

		// Read extra string
		copy(nbuf[newpos:], ctrl.Copy)

		// Adjust pointers
		newpos += int64(len(ctrl.Copy))
		oldpos += ctrl.Seek
	}

	if newpos != newSize {
		return fmt.Errorf("bsdiff: expected new file to be %d, was %d (%s difference)", newSize, newpos, humanize.IBytes(uint64(newSize-newpos)))
	}

	// Write the new file
	_, err = new.Write(nbuf)
	if err != nil {
		return err
	}

	return nil
}
