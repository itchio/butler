package zip

import (
	"bytes"
	"unicode/utf8"

	"github.com/gogits/chardet"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
)

const nameEncodingSampleSize = 4096

/**
 * See https://github.com/itchio/itch/issues/2103
 *
 * Historically, zip file names are CP-437.
 * Except on some platforms where they're actually Shift-JIS, or
 * something else.
 *
 * So far, we've seen mostly CP-437 and Shift-JIS in the wild, so
 * we: collect file names, put them in a single buffer, try to detect
 * encoding from there. If the confidence index is high enough, we
 * convert them to utf-8.
 */
func normalizeNameEncoding(z *Reader) error {
	buf := new(bytes.Buffer)
	buf.Grow(nameEncodingSampleSize)

	hasNonUtf8 := false

	for _, f := range z.File {
		if f.NonUTF8 && !utf8.ValidString(f.Name) {
			hasNonUtf8 = true
			buf.WriteString(f.Name)
			buf.WriteByte(' ')
			if buf.Len() > nameEncodingSampleSize {
				break
			}
		}
	}

	if hasNonUtf8 {
		// default and fallback
		var encoding encoding.Encoding = charmap.CodePage437

		detector := chardet.NewTextDetector()
		res, err := detector.DetectBest(buf.Bytes())
		if err == nil && res.Confidence > 70 {
			switch res.Charset {
			case "Shift_JIS":
				// TODO: support other encodings when we find them in the wild.
				// I've been looking for ISO-8859-x files to no avail.
				encoding = japanese.ShiftJIS
			}
		}

		decoder := encoding.NewDecoder()
		for _, f := range z.File {
			if f.NonUTF8 {
				decoded, err := decoder.String(f.Name)
				if err == nil {
					f.Name = decoded
				}
			}
		}
	}
	return nil
}
