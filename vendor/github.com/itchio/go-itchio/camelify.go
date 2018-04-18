package itchio

func camelify(input interface{}) interface{} {
	if m, ok := input.(map[string]interface{}); ok {
		return camelifyMap(m)
	}

	if a, ok := input.([]interface{}); ok {
		return camelifyArray(a)
	}

	return input
}

// FIXME: this is bad, find another way to fix it
var camelifyBlacklist = map[string]struct{}{
	"upload_headers": {},
}

func camelifyArray(input []interface{}) []interface{} {
	var result []interface{}

	for _, el := range input {
		result = append(result, camelify(el))
	}

	return result
}

func camelifyMap(input map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for k, v := range input {
		if _, ok := camelifyBlacklist[k]; ok {
			result[camelcase(k)] = v
		} else {
			result[camelcase(k)] = camelify(v)
		}
	}

	return result
}

// Adapted from https://github.com/segmentio/go-camelcase

// camelcase the given string.
func camelcase(s string) string {
	b := make([]byte, 0, 64)
	l := len(s)
	i := 0

	for i < l {
		// skip leading bytes that aren't letters or digits
		for i < l && !isWord(s[i]) {
			i++
		}

		// set the first byte to uppercase if it needs to
		if i < l {
			c := s[i]

			// simply append contiguous digits
			if isDigit(c) {
				for i < l {
					if c = s[i]; !isDigit(c) {
						break
					}
					b = append(b, c)
					i++
				}
				continue
			}

			// the sequence starts with and uppercase letter, we append
			// all following uppercase letters as equivalent lowercases
			if isUpper(c) {
				b = append(b, c)
				i++

				for i < l {
					if c = s[i]; !isUpper(c) {
						break
					}
					b = append(b, toLower(c))
					i++
				}

			} else {
				b = append(b, toUpper(c))
				i++
			}

			// append all trailing lowercase letters
			for i < l {
				if c = s[i]; !isLower(c) {
					break
				}
				b = append(b, c)
				i++
			}
		}
	}

	// the first byte must always be lowercase
	if len(b) != 0 {
		b[0] = toLower(b[0])
	}

	return string(b)
}

func isWord(c byte) bool {
	return isLetter(c) || isDigit(c)
}

func isLetter(c byte) bool {
	return isLower(c) || isUpper(c)
}

func isUpper(c byte) bool {
	return c >= 'A' && c <= 'Z'
}

func isLower(c byte) bool {
	return c >= 'a' && c <= 'z'
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func toLower(c byte) byte {
	if isUpper(c) {
		return c + ('a' - 'A')
	}
	return c
}

func toUpper(c byte) byte {
	if isLower(c) {
		return c - ('a' - 'A')
	}
	return c
}
