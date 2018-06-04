package gosaca

func setAllToEmpty(SA []int) {
	for i := range SA {
		SA[i] = empty
	}
}

// compute the length of the LMS substring at the front of the LMS suffix s[:]
// pre-condition: s[:] is an LMS suffix
// WARNING: if s[:] ends on the sentinel, the returned value will be len(s)+1!
func lmsSubstringLength0(s []byte) int {
	n := len(s)
	for i := 2; i < n; i++ {
		if s[i] < s[i-1] {
			// s[i-1] is L-type; move on to step 2
			for j := i; j < n; j++ {
				if s[j] > s[j-1] {
					return i
				} else if s[j] < s[j-1] {
					i = j + 1
				}
			}
		}
	}
	return n + 1 // add one to indicate substring ended with the sentinel
}

// for level 0, rename the LMS substrings sitting in SA1, and return the new alphabet size (k1)
func rename0(S []byte, SA1, work, S1 []int) int {
	n := len(S)
	n1 := len(SA1)

	if n1 == 0 {
		return 0
	}

	// currently work only holds positive values; we save the time of clearing
	// it out by inserting bitwise-negated (negative) values so we know which
	// ones we put in vs which ones just contain old data

	// walk SA1 from left to right, creating Z1 (spread throughout work)

	// first, record the first LMS suffix
	k1 := 1
	bktHead := 0 // renamed value == head of bucket in SA1 (part of property 4.1)
	prev := SA1[0]
	work[prev/2] = ^bktHead
	SA1[0] = 1 // after we read SA[i], reuse it as a bucket size (needed for post-Z1 step)

	// at each step, we need to see if the LMS substring starting at S[SA1[i]] (S[pos])
	// is the same as the one we just saw starting at S[SA1[i-1]] (S[prev])
	for i := 1; i < n1; i++ {
		pos := SA1[i]
		SA1[i] = 0 // reused as bucket size
		diff := false

		// quick first test - if initial character is different we're done
		if S[prev] != S[pos] {
			diff = true
		} else {
			// TODO - this walks both LMS substrings to calculate their lengths; can we combine this to short-circuit earlier if possible? tricky to do correctly!
			prevLen := lmsSubstringLength0(S[prev:])
			posLen := lmsSubstringLength0(S[pos:])
			if prev+prevLen == n+1 || // S[prev:] ends with sentinel
				pos+posLen == n+1 || // S[pos:] ends with sentinel
				prevLen != posLen { // different lengths
				diff = true
			} else {
				// if we get here:
				//   (a) first character is the same
				//   (b) both end before the sentinel
				//   (c) both have the same length
				// so we need to check the rest of the characters one-by-one
				for j := 1; j < prevLen; j++ {
					if S[prev+j] != S[pos+j] {
						diff = true
						break
					}
				}
			}
		}

		if diff {
			bktHead = i
			k1++
		}
		work[pos/2] = ^bktHead
		SA1[bktHead]++ // increment bucket size
		prev = pos
	}

	// Z1 is now sitting (sparsely) in work[]
	buildS1FromZ1(S1, SA1, work)

	return k1
}

// this function is almost the same as rename0; differences:
//   (a) S is an []int
//   (b) our LMS substring comparision is different (and easier)
func rename1(S, SA1, work, S1 []int) int {
	n := len(S)
	n1 := len(SA1)

	if n1 == 0 {
		return 0
	}

	// currently work only holds positive values; we save the time of clearing
	// it out by inserting bitwise-negated (negative) values so we know which
	// ones we put in vs which ones just contain old data

	// walk SA1 from left to right, creating Z1 (spread throughout work)

	// first, record the first LMS suffix
	k1 := 1
	bktHead := 0 // renamed value == head of bucket in SA1 (part of property 4.1)
	prev := SA1[0]
	work[prev/2] = ^bktHead
	SA1[0] = 1 // after we read SA[i], reuse it as a bucket size (needed for post-Z1 step)

	// at each step, we need to see if the LMS substring starting at S[SA1[i]] (S[pos])
	// is the same as the one we just saw starting at S[SA1[i-1]] (S[prev])
	for i := 1; i < n1; i++ {
		pos := SA1[i]
		SA1[i] = 0 // reused as bucket size
		diff := false

		// walk both strings character-by-character until (a) we get a
		// difference or (b) we begin the L+ sequence
		j := 0
		for j = 0; j < n; j++ {
			if S[prev+j] != S[pos+j] {
				diff = true
				break
			} else if S[prev+j] >= 0 {
				break
			}
		}

		if !diff {
			// both strings started L+ at the same place; now walk until:
			//  (a) either hits the end (=> different)
			//  (b) we get a different character (=> different)
			//  (c) both hit the same S-type value (=> same)
			for j++; j < n; j++ {
				if prev+j == n || pos+j == n || S[prev+j] != S[pos+j] {
					diff = true
					break
				} else if S[prev+j] < 0 {
					break
				}
			}
		}

		if diff {
			bktHead = i
			k1++
		}
		work[pos/2] = ^bktHead
		SA1[bktHead]++ // increment bucket size
		prev = pos
	}

	// Z1 is now sitting (sparsely) in work[]
	buildS1FromZ1(S1, SA1, work)

	return k1
}

// Build S1 from Z1 (which is sitting sparsely in work[] - all the negative
// values for work are the bitwise inversions of Z1).
func buildS1FromZ1(S1, SA1, work []int) {
	n1 := len(S1)

	// walk work[] from right-to-left and adjust any S-type characters to point to the end of their bucket instead of the beginning
	Z1pos := len(work) - 1
	for i := 0; i < n1; i++ {
		// find next element of Z1
		for work[Z1pos] >= 0 {
			Z1pos--
		}

		// record character (head of bucket, only correct for L-type)
		c := ^work[Z1pos]
		S1[n1-1-i] = c
		Z1pos--

		// check and see if c is S-type
		if i > 0 && // S1[n-1] is L-type by definition due to sentinel
			((S1[n1-i] < 0 && c <= ^S1[n1-i]) || // S1[n1-i] was S-type and we are <= it
				(S1[n1-i] >= 0 && c < S1[n1-i])) { // S1[n1-i] was L-type and we are < it
			// Adjust c so it points to the end of its bucket instead of the
			// head. Note that in the Z1 construction loop above, we stored
			// the width of each bucket in SA1[c]. Also, bitwise negate it so
			// the recursive computeSuffixArray1 doesn't have to.
			S1[n1-1-i] = ^(S1[n1-1-i] + SA1[c] - 1)
		}
	}
}

func sortRecursively(S1, SA1 []int, k1 int) {
	if k1 == len(S1) {
		for i, s := range S1 {
			if s < 0 {
				SA1[^s] = i
			} else {
				SA1[s] = i
			}
		}
	} else {
		computeSuffixArray1(S1, SA1, k1)
	}
}
