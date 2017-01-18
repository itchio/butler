package bsdiff

const Cutoff = 16

type SuffixArrayZ struct {
	input []byte
	index []int64
	n     int64
}

func NewSuffixArrayZ(input []byte) *SuffixArrayZ {
	sa := &SuffixArrayZ{
		input: append(input, 0),
		index: make([]int64, len(input)),
		n:     int64(len(input)),
	}

	for i := int64(0); i < sa.n; i++ {
		sa.index[i] = i
	}

	// shuffle
	sa.sort(0, sa.n-1, 0)

	return sa
}

func (sa *SuffixArrayZ) sort(lo, hi, d int64) {
	// cutoff to insertion sort for small subarrays
	if hi <= lo+Cutoff {
		sa.insertion(lo, hi, d)
		return
	}
	lt, gt := lo, hi
	v := sa.input[sa.index[lo]+d]
	i := lo + 1
	for i <= gt {
		// fmt.Printf("Accessing index %d / %d\n", sa.index[i]+d, sa.n)
		t := sa.input[sa.index[i]+d]
		if t < v {
			sa.exch(lt, i)
			lt++
			i++
		} else if t > v {
			sa.exch(i, gt)
			gt--
		} else {
			i++
		}
	}

	// a[lo..lt-1] < v = a[lt..gt] < a[gt+1..hi].
	sa.sort(lo, lt-1, d)
	if v > 0 {
		sa.sort(lt, gt, d+1)
	}
	sa.sort(gt+1, hi, d)
}

// exchange index[i] and index[j]
func (sa *SuffixArrayZ) exch(i, j int64) {
	swap := sa.index[i]
	sa.index[i] = sa.index[j]
	sa.index[j] = swap
}

// sort from a[lo] to a[hi], starting at the dth character
func (sa *SuffixArrayZ) insertion(lo, hi, d int64) {
	for i := lo; i <= hi; i++ {
		for j := i; j > lo && sa.less(sa.index[j], sa.index[j-1], d); j-- {
			sa.exch(j, j-1)
		}
	}
}

// is text[i+d..N) < text[j+d..N) ?
func (sa *SuffixArrayZ) less(i, j, d int64) bool {
	if i == j {
		return false
	}
	i = i + d
	j = j + d
	for i < sa.n && j < sa.n {
		if sa.input[i] < sa.input[j] {
			return true
		}
		if sa.input[i] > sa.input[j] {
			return false
		}
		i++
		j++
	}
	return i > j
}
