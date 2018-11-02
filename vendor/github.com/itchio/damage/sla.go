package damage

import "github.com/pkg/errors"

// An SLA is a Service License Agreement, which can be embedded into a disk image.
type SLA struct {
	Language Language
	Text     string
}

// GetDefaultSLA returns the Service License Agreement for the default language.
func GetDefaultSLA(rez *UDIFResources) (*SLA, error) {
	lpic, err := rez.ParsedLPic()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return GetSLA(rez, lpic.DefaultLanguage)
}

// GetSLA returns the Service License Agreement for a specific language.
func GetSLA(rez *UDIFResources, lang Language) (*SLA, error) {
	lpic, err := rez.ParsedLPic()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	entry, ok := lpic.ByLanguage(lang)
	if !ok {
		return nil, nil
	}

	textRes, ok := rez.Text.ByID(LPicResourceID + entry.RelativeResourceID)
	if !ok {
		return nil, nil
	}

	return &SLA{
		Language: entry.Language,
		Text:     textRes.StringData(),
	}, nil
}
