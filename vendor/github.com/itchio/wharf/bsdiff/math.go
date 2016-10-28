package bsdiff

import (
	"bytes"
	"fmt"
	"os"
	"runtime"

	"github.com/itchio/wharf/state"
)

var parallel = os.Getenv("PARALLEL_BSDIFF") == "1"

// Ternary-Split Quicksort, cf. http://www.larsson.dogma.net/ssrev-tr.pdf
func split(I, V, V2 []int32, start, length, h int32) {
	// fmt.Fprintf(os.Stderr, "split(%d, %d)\n", start, length)

	var i, j, k, x, jj, kk int32

	if length < 16 {
		for k = start; k < start+length; k += j {
			j = 1
			x = V2[I[k]+h]
			for i = 1; k+i < start+length; i++ {
				if V2[I[k+i]+h] < x {
					x = V2[I[k+i]+h]
					j = 0
				}
				if V2[I[k+i]+h] == x {
					I[k+i], I[k+j] = I[k+j], I[k+i]
					j++
				}
			}
			for i = 0; i < j; i++ {
				V2[I[k+i]] = k + j - 1
			}
			if j == 1 {
				I[k] = -1
			}
		}
		return
	}

	// fmt.Fprintf(os.Stderr, "start+length/2 = %d, len(I) = %d\n", start+length/2, len(I))
	// fmt.Fprintf(os.Stderr, "V[%d] (read)\n", I[start+length/2]+h)
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
		// fmt.Fprintf(os.Stderr, "lsplit I[%d-%d]\n", start, jj)
		split(I, V, V2, start, jj-start, h)
	}

	for i = 0; i < kk-jj; i++ {
		// fmt.Fprintf(os.Stderr, "V[%d] = %d\n", I[jj+i], kk-1)
		V2[I[jj+i]] = kk - 1
	}
	if jj == kk-1 {
		// fmt.Fprintf(os.Stderr, "I[%d] = %d\n", I[jj], -1)
		I[jj] = -1
	}

	if start+length > kk {
		// fmt.Fprintf(os.Stderr, "rsplit I[%d-%d]\n", kk, start+length)
		split(I, V, V2, kk, start+length-kk, h)
	}
}

type mark struct {
	index int32
	value int32
}

type sortTask struct {
	start  int32
	length int32
	h      int32
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

	V2 := append([]int32{}, V...)
	marks := make([]mark, 0)
	numWorkers := runtime.NumCPU()
	if parallel {
		consumer.Debugf("parallel suffix sorting (%d workers)", numWorkers)
	} else {
		consumer.Debugf("single-core suffix sorting")
	}

	for h = 1; I[0] != -(obuflen + 1); h += h {
		done := make(chan bool)
		tasks := make(chan sortTask, 1024)

		if parallel {
			for i := 0; i < numWorkers; i++ {
				go func() {
					for task := range tasks {
						split(I, V, V2, task.start, task.length, task.h)
					}
					done <- true
				}()
			}
		}

		marks = marks[:0]

		consumer.ProgressLabel(fmt.Sprintf("Suffix sorting (%d-order)", h))
		// consumer.Debugf("\n>> Pass %d", h)

		var lastI int32
		var n int32
		var doneI int32

		for i = 0; i < obuflen+1; {
			if i-lastI > progressInterval {
				progress := float64(i) / float64(obuflen)
				consumer.Progress(progress)
				lastI = i
			}

			if I[i] < 0 {
				// consumer.Debugf("Found %d combined-sorted", -I[i])
				doneI -= I[i]
				n -= I[i]
				i -= I[i]
			} else {
				if n != 0 {
					marks = append(marks, mark{index: i - n, value: -n})
					// I[i-n] = -n
				}
				n = V[I[i]] + 1 - i
				// consumer.Debugf("\n> Splitting %d-%d array", i, i+n)

				if parallel {
					tasks <- sortTask{
						start:  i,
						length: n,
						h:      h,
					}
				} else {
					split(I, V, V2, i, n, h)
				}

				i += n
				n = 0
			}
		}

		if parallel {
			close(tasks)
			for i := 0; i < numWorkers; i++ {
				<-done
			}
		}

		for _, mark := range marks {
			// consumer.Debugf("Setting I[%d] to %d", I[i-n], -n)
			I[mark.index] = mark.value
		}

		if n != 0 {
			I[i-n] = -n
		}

		// consumer.Debugf("%d/%d was already done (%.2f%%)", doneI, (obuflen + 1),
		// 	100.0*float64(doneI)/float64(obuflen+1))

		copy(V, V2)
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
