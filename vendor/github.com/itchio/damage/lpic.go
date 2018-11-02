package damage

import (
	"bufio"
	"bytes"
	"encoding/binary"

	"github.com/pkg/errors"
)

// LPicResourceID is the identifier for the base
// LPic resource - where languages are listed for an SLA.
const LPicResourceID int64 = 5000

// LPic is the decoded version of UDIF's language information for SLAs.
type LPic struct {
	Missing         bool
	DefaultLanguage Language
	Entries         []LPicEntry
}

// ByLanguage returns the information for a specific language,
// or (, false) if it can't be found.
func (lp *LPic) ByLanguage(lang Language) (LPicEntry, bool) {
	for _, entry := range lp.Entries {
		if entry.Language == lang {
			return entry, true
		}
	}
	return LPicEntry{}, false
}

// An LPicEntry contains information for one of the
// languages an SLA is available in.
type LPicEntry struct {
	Language Language
	// RelativeResourceID is the ID of the STR#/TEXT/styl resource
	// for the SLA in this language, relative to LPicResourceID
	RelativeResourceID int64
	// MultibyteLanguage is true if Language is multibyte
	// See /System/Library/Frameworks/CoreServices.framework/Frameworks/CarbonCore.framework/Headers/Script.h for language IDs.
	MultibyteLanguage bool
}

// ParsedLPic returns a parsed version of the LPic resource from an UDIF.
// It'll mostly error out if the resource is invalid (unlikely) or if our
// parser is wrong (more likely).
func (rez UDIFResources) ParsedLPic() (*LPic, error) {
	var err error

	if rez.parsedLPic == nil {
		if res, ok := rez.LPic.ByID(LPicResourceID); ok {
			rez.parsedLPic, err = parseLPic(res)
		} else {
			rez.parsedLPic = &LPic{Missing: true}
		}
	}

	return rez.parsedLPic, err
}

// The LPic resource is a series of 16-bit big-endian values,
// see https://github.com/adobe/chromium/blob/cfe5bf0b51b1f6b9fe239c2a3c2f2364da9967d7/chrome/installer/mac/pkg-dmg
//
// Reference:
//
// data 'LPic' (5000) {
// 	 // Default language ID, 0 = English
// 	 $"0000"
// 	 // Number of entries in list
// 	 $"0001"
// 	 // Entry 1
// 	 // Language ID, 0 = English
// 	 $"0000"
// 	 // Resource ID, 0 = STR#/TEXT/styl 5000
// 	 $"0000"
// 	 // Multibyte language, 0 = no
// 	 $"0000"
// };
func parseLPic(res UDIFResource) (*LPic, error) {
	lpic := &LPic{}

	r := bufio.NewReader(bytes.NewReader(res.Data))
	bo := binary.BigEndian

	var word int16
	readWord := func() error {
		return binary.Read(r, bo, &word)
	}

	if err := readWord(); err != nil {
		return nil, errors.WithStack(err)
	}
	lpic.DefaultLanguage = Language(word)

	var numEntries int
	if err := readWord(); err != nil {
		return nil, errors.WithStack(err)
	}
	numEntries = int(word)

	for i := 0; i < numEntries; i++ {
		var entry LPicEntry
		if err := readWord(); err != nil {
			return nil, errors.WithStack(err)
		}
		entry.Language = Language(word)

		if err := readWord(); err != nil {
			return nil, errors.WithStack(err)
		}
		entry.RelativeResourceID = int64(word)

		if err := readWord(); err != nil {
			return nil, errors.WithStack(err)
		}
		entry.MultibyteLanguage = word == 1

		lpic.Entries = append(lpic.Entries, entry)
	}

	return lpic, nil
}
