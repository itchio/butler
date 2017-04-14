package wizardry

import "io"

// SearchTest looks for a fixed pattern at any position within a certain length
func SearchTest(r io.ReaderAt, size int64, targetIndex int64, maxLen int64, pattern string) int64 {
	sf := MakeStringFinder(pattern)

	target := make([]byte, maxLen)
	n, err := r.ReadAt(target, int64(targetIndex))
	if err != nil && err != io.EOF {
		return -1
	}

	text := string(target[:n])
	return int64(sf.next(text))
}
