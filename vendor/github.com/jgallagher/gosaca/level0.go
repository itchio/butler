package gosaca

const (
	maxInt = int(^uint(0) >> 1)
	minInt = -(maxInt - 1)
	empty  = minInt
)

// WorkSpace contains the O(1) scratch space used in constructing a suffix array with an alphabet of sisze 256 (any byte value).
type WorkSpace struct {
	bkt     [256]int // working space buckets
	bktHead [256]int // save off bucket heads
	bktTail [256]int // save off bucket tails
	dirty   bool     // true if the scratch space is dirty from a previous run
}

// Compute the suffix array of S, storing it into SA. len(S) and len(SA) must be equal.
func (ws *WorkSpace) ComputeSuffixArray(S []byte, SA []int) {
	n := len(S)
	bkt := ws.bkt[:]

	// scan S once, computing all bucket heads/tails
	ws.computeBuckets(S)

	// *********************************************
	// Stage 1: Induced-sort the LMS-substrings of S
	// *********************************************

	// step 1 - initialize SA as empty
	setAllToEmpty(SA)

	// step 2 - put all LMS substrings into buckets based on their first character
	// insert from the end to the head of the buckets (bkt currently holds the tails of the buckets from computeBuckets)
	for i := n - 2; i >= 0; i-- {
		if S[i] >= S[i+1] {
			// S[i] is L-type; move on
			continue
		}

		// S[i] is S-type; keep moving back until S[i-1] is L-type
		for i >= 1 && (S[i-1] < S[i] || S[i-1] == S[i]) {
			// S[i-1] is also S-type; keep moving back
			i--
		}

		// unless we hit S[0] (which is not LMS by definition), S[i] begins an LMS suffix, so insert it into its bucket
		if i > 0 {
			SA[bkt[S[i]]] = i
			bkt[S[i]]--
		}
	}

	// step 3 - induced sort the L-type suffixes of S into their buckets
	copy(bkt, ws.bktHead[:])
	induceSortL0(S, SA, bkt)

	// step 4 - induced sort the S-type suffixes of S into their buckets
	copy(bkt, ws.bktTail[:])
	induceSortS0(S, SA, bkt)

	// NOT DESCRIBED IN PAPER BUT STILL NECESSARY (see SA-IS)
	// We need to compact all the now-sorted LMS substrings into the first n1 positions of SA
	// To do this, make use of the fact that since we just inserted all the S-type
	// suffixes into SA from tail-to-head of the buckets, we can loop over the buckets
	// themselves and pull out the S-type suffixes: all the S-type suffixes starting with c
	// are contained in SA[bkt[c]+1] to SA[bktTail[c]]
	n1 := 0
	for c := 0; c < 256; c++ {
		for i := bkt[c] + 1; i <= ws.bktTail[c]; i++ {
			j := SA[i]
			// we know S[j] is S-type; now see if it's LMS (i.e., preceded by an L-type)
			if j > 0 && S[j-1] > S[j] {
				SA[n1] = j
				n1++
			}
		}
	}

	// *********************************************
	// Stage 2: Rename the LMS substrings
	// *********************************************

	// provably, n1 is at most floor(n/2), so the following overlapping works
	SA1 := SA[:n1] // SA1 overlaps the front of SA
	work := SA[n1:] // workspace overlaps the rest of SA
	S1 := SA[n-n1:] // S1 overlaps the end of SA (including part of "work", but rename deals with that correctly)
	k1 := rename0(S, SA1, work, S1)

	// *********************************************
	// Stage 3: Sort recursively
	// *********************************************
	sortRecursively(S1, SA1, k1)

	// NOT DESCRIBED IN PAPER BUT STILL NECESSARY (see SA-IS)
	// We need to undo the renaming of the LMS suffixes.
	// We no longer need S1, so reuse it to hold all the LMS indices.
	j := n1 - 1
	for i := n - 2; i >= 0; i-- {
		if S[i] >= S[i+1] {
			// S[i] is L-type
			continue
		}
		// S[i] is S-type; walk backwards to find LMS
		for i >= 1 && (S[i-1] < S[i] || S[i-1] == S[i]) {
			// S[i-1] is also S-type
			i--
		}
		// S[0] is not LMS by definition, but otherwise S[i] is LMS
		if i > 0 {
			S1[j] = i
			j--
		}
	}
	// Now convert SA1 from renamed values to true values.
	for i, s := range SA1 {
		SA1[i] = S1[s]
	}

	// *********************************************
	// Stage 4: Induced-sort SA(S) from SA1(S1)
	// *********************************************

	// step 1 - initialize SA[n1:] as empty
	setAllToEmpty(SA[n1:])

	// step 2 - put all sorted LMS substrings into buckets based on their first character
	// insert from the end to the head of the buckets
	copy(bkt, ws.bktTail[:])
	for i := n1 - 1; i >= 0; i-- {
		j := SA1[i]
		SA1[i] = empty // clear it out in preparation for steps 3-4
		if j == 0 {
			panic("unexpected j == 0")
		}
		c := S[j]
		SA[bkt[c]] = j
		bkt[c]--
	}

	// step 3 - induced sort the L-type suffixes of S into their buckets
	copy(bkt, ws.bktHead[:])
	induceSortL0(S, SA, bkt)

	// step 4 - induced sort the S-type suffixes of S into their buckets
	copy(bkt, ws.bktTail[:])
	induceSortS0(S, SA, bkt)
}

func (ws *WorkSpace) computeBuckets(S []byte) {
	if ws.dirty {
		// clear out bucket counters from a previous call to ComputeSuffixArray
		for i := 0; i < 256; i++ {
			ws.bkt[i] = 0
		}
	}

	// compute sizes of each bucket
	for _, c := range S {
		ws.bkt[c]++
	}

	// record head and tail of each bucket (also store tails into bkt, as that's the one we need first)
	sum := 0
	for i := 0; i < 256; i++ {
		ws.bktHead[i] = sum
		sum += ws.bkt[i]
		ws.bktTail[i] = sum - 1
		ws.bkt[i] = sum - 1
	}

	// record that our buckets are dirty in case ws is used again
	ws.dirty = true
}

// pre-condition: SA contains properly bucketed LMS substrings
// pre-condition: bkt contains the head of each character's bucket
// post-condition: SA contains properly bucketed L-type and LMS suffixes
func induceSortL0(S []byte, SA, bkt []int) {
	n := len(S)

	// special case to deal with the (virtual) sentinel:
	// S[n-1] is L-type because of the sentinel, and if we were treating
	// the sentinel as a real character, it would be at the front of SA[]
	// (it's effectively stored in "SA[-1]")
	c := S[n-1]
	SA[bkt[c]] = n - 1
	bkt[c]++

	// at each step, look at the character *before* S[SA[i]]; if it's L-type, insert it
	for _, SAi := range SA {
		if SAi <= 0 {
			// if SA[i] is empty or points to S[0], we don't have a preceding character to check
			continue
		}

		j := SAi - 1
		c := S[j] // character we care about

		// check for L-type (described in section 3)
		// since SA only holds L-type and LMS suffixes, c must be L-type if it is >= S[j]
		if c >= S[SAi] {
			SA[bkt[c]] = j
			bkt[c]++
		}
	}
}

// pre-condition: SA contains properly bucketed L and LMS suffixes
// pre-condition: bkt contains the tail of each character's bucket
// post-condition: SA contains properly also contains all properly bucketed S-type suffixes
func induceSortS0(S []byte, SA, bkt []int) {
	n := len(S)

	// at each step, look at the character *before* S[SA[i]]; if it's S-type, insert it
	for i := n - 1; i >= 0; i-- {
		SAi := SA[i]
		if SAi <= 0 {
			continue
		}

		j := SAi - 1
		c := S[j] // character we care about

		// check for S-type (use Property 3.1)
		if c < S[SAi] || (c == S[SAi] && bkt[c] < i) {
			SA[bkt[c]] = j
			bkt[c]--
		}
	}

	// we don't need to do anything special for the sentinel - by definition the character before it is not S-type
}
