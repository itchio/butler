package pelican

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/itchio/pelican/pe"

	xj "github.com/basgys/goxml2json"
	humanize "github.com/dustin/go-humanize"
	"github.com/itchio/wharf/state"
	"github.com/pkg/errors"
)

type imageResourceDirectory struct {
	Characteristics      uint32
	TimeDateStamp        uint32
	MajorVersion         uint16
	MinorVersion         uint16
	NumberOfNamedEntries uint16
	NumberOfIdEntries    uint16
}

type imageResourceDirectoryEntry struct {
	NameId uint32
	Data   uint32
}

type imageResourceDataEntry struct {
	Data     uint32
	Size     uint32
	CodePage uint32
	Reserved uint32
}

type ResourceType uint32

// https://msdn.microsoft.com/fr-fr/library/windows/desktop/ms648009(v=vs.85).aspx
const (
	ResourceTypeNone ResourceType = 0

	ResourceTypeCursor       ResourceType = 1
	ResourceTypeBitmap       ResourceType = 2
	ResourceTypeIcon         ResourceType = 3
	ResourceTypeMenu         ResourceType = 4
	ResourceTypeDialog       ResourceType = 5
	ResourceTypeString       ResourceType = 6
	ResourceTypeFontDir      ResourceType = 7
	ResourceTypeFont         ResourceType = 8
	ResourceTypeAccelerator  ResourceType = 9
	ResourceTypeRcData       ResourceType = 10
	ResourceTypeMessageTable ResourceType = 11

	ResourceTypeGroupCursor ResourceType = ResourceTypeCursor + 11 // 12
	ResourceTypeGroupIcon   ResourceType = ResourceTypeIcon + 11   // 14

	ResourceTypeVersion    ResourceType = 16
	ResourceTypeDlgInclude ResourceType = 17
	ResourceTypePlugPlay   ResourceType = 19
	ResourceTypeVXD        ResourceType = 20 // vxd = virtual device
	ResourceTypeAniCursor  ResourceType = 21
	ResourceTypeAniIcon    ResourceType = 22
	ResourceTypeHTML       ResourceType = 23
	ResourceTypeManifest   ResourceType = 24
)

var ResourceTypeNames = map[ResourceType]string{
	ResourceTypeCursor:       "Cursor",
	ResourceTypeBitmap:       "Bitmap",
	ResourceTypeIcon:         "Icon",
	ResourceTypeMenu:         "Menu",
	ResourceTypeDialog:       "Dialog",
	ResourceTypeString:       "String",
	ResourceTypeFontDir:      "FontDir",
	ResourceTypeFont:         "Font",
	ResourceTypeAccelerator:  "Accelerator",
	ResourceTypeRcData:       "RcData",
	ResourceTypeMessageTable: "MessageTable",
	ResourceTypeGroupCursor:  "GroupCursor",
	ResourceTypeGroupIcon:    "GroupIcon",
	ResourceTypeVersion:      "Version",
	ResourceTypeDlgInclude:   "DlgInclude",
	ResourceTypePlugPlay:     "PlugPlay",
	ResourceTypeVXD:          "VXD",
	ResourceTypeAniCursor:    "AniCursor",
	ResourceTypeAniIcon:      "AniIcon",
	ResourceTypeHTML:         "HTML",
	ResourceTypeManifest:     "Manifest",
}

func parseResources(consumer *state.Consumer, info *PeInfo, sect *pe.Section) error {
	consumer.Debugf("Found resource section (%s)", humanize.IBytes(uint64(sect.Size)))

	var readDirectory func(offset uint32, level int, resourceType ResourceType) error
	readDirectory = func(offset uint32, level int, resourceType ResourceType) error {
		prefix := strings.Repeat("  ", level)
		log := func(msg string, args ...interface{}) {
			consumer.Debugf("%s%s", prefix, fmt.Sprintf(msg, args...))
		}

		br := io.NewSectionReader(sect, int64(offset), int64(sect.Size)-int64(offset))
		ird := new(imageResourceDirectory)
		err := binary.Read(br, binary.LittleEndian, ird)
		if err != nil {
			return errors.WithStack(err)
		}

		for i := uint16(0); i < ird.NumberOfNamedEntries+ird.NumberOfIdEntries; i++ {
			irde := new(imageResourceDirectoryEntry)
			err = binary.Read(br, binary.LittleEndian, irde)
			if err != nil {
				return errors.WithStack(err)
			}

			if irde.NameId&0x80000000 > 0 {
				continue
			}

			id := irde.NameId & 0xffff
			if level == 0 {
				typ := ResourceType(id)
				if name, ok := ResourceTypeNames[typ]; ok {
					log("=> %s", name)
				} else {
					log("=> type #%d (unknown)", id)
				}
			} else {
				log("=> %d", id)
			}

			if irde.Data&0x80000000 > 0 {
				offset := irde.Data & 0x7fffffff
				recResourceType := resourceType
				if level == 0 {
					recResourceType = ResourceType(id)
				}

				err := readDirectory(offset, level+1, recResourceType)
				if err != nil {
					return errors.WithStack(err)
				}
				continue
			}

			dbr := io.NewSectionReader(sect, int64(irde.Data), int64(sect.Size)-int64(irde.Data))

			irda := new(imageResourceDataEntry)
			err = binary.Read(dbr, binary.LittleEndian, irda)
			if err != nil {
				return errors.WithStack(err)
			}

			if resourceType == ResourceTypeManifest || resourceType == ResourceTypeVersion {
				log("@ %x (%s, %d bytes)", irda.Data, humanize.IBytes(uint64(irda.Size)), irda.Size)

				sr := io.NewSectionReader(sect, int64(irda.Data-sect.VirtualAddress), int64(irda.Size))
				rawData, err := ioutil.ReadAll(sr)
				if err != nil {
					return errors.WithStack(err)
				}

				switch resourceType {
				case ResourceTypeManifest:
					// actually not utf-16,
					// but TODO: figure out
					// codepage
					stringData := string(rawData)
					log("=========================")
					for _, l := range strings.Split(stringData, "\n") {
						log("%s", l)
					}
					log("=========================")

					js, err := xj.Convert(strings.NewReader(stringData))
					if err != nil {
						log("could not convert xml to json: %s", err.Error())
					} else {
						err := interpretManifest(info, js.Bytes())
						if err != nil {
							log("could not interpret manifest: %s", err.Error())
						}
					}
				case ResourceTypeVersion:
					err := parseVersion(info, consumer, rawData)
					if err != nil {
						return errors.WithStack(err)
					}
				}
			}
		}
		return nil
	}

	err := readDirectory(0, 0, 0)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
