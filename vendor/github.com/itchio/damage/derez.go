package damage

import (
	"fmt"

	"github.com/itchio/damage/hdiutil"
	"github.com/pkg/errors"
)

// UDIFResources contains a subset of the resources contained in a UDIF
// (Apple Universal Disk Image Format).
type UDIFResources struct {
	// LPic is an acronym I don't know, but it contains
	// default language information and a language list for the SLA (Software License Agreement)
	LPic    UDIFResourceGroup `plist:"LPic"`
	StrHash UDIFResourceGroup `plist:"STR#"`
	Text    UDIFResourceGroup `plist:"TEXT"`
	Tmpl    UDIFResourceGroup `plist:"TMPL"`
	Plst    UDIFResourceGroup `plist:"plst"`
	Styl    UDIFResourceGroup `plist:"styl"`

	parsedLPic *LPic
}

type UDIFResourceGroup []UDIFResource

func (grp UDIFResourceGroup) ByIDString(idString string) (UDIFResource, bool) {
	for _, res := range grp {
		if res.ID == idString {
			return res, true
		}
	}
	return UDIFResource{}, false
}

func (grp UDIFResourceGroup) ByID(id int64) (UDIFResource, bool) {
	return grp.ByIDString(fmt.Sprintf("%d", id))
}

type UDIFResource struct {
	Data       []byte `plist:"Data"`
	ID         string `plist:"ID"`
	Name       string `plist:"Name"`
	Attributes string `plist:"Attributes"`
}

// StringData returns Data as an ASCII (7-bit) string.
// Multibyte encodings are not supported.
func (res UDIFResource) StringData() string {
	return string(res.Data)
}

func GetUDIFResources(host hdiutil.Host, dmgpath string) (*UDIFResources, error) {
	var rez UDIFResources
	err := host.Command("udifderez").WithArgs("-xml", dmgpath).RunAndDecode(&rez)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &rez, nil
}
