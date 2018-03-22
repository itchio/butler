package exeprops

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
)

var args = struct {
	path *string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("exeprops", "(Advanced) Gives information about an .exe file").Hidden()
	args.path = cmd.Arg("path", "The exe to analyze").Required().String()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(*args.path, comm.NewStateConsumer()))
}

type ImageResourceDirectory struct {
	Characteristics      uint32
	TimeDateStamp        uint32
	MajorVersion         uint16
	MinorVersion         uint16
	NumberOfNamedEntries uint16
	NumberOfIdEntries    uint16
}

type ImageResourceDirectoryEntry struct {
	NameId uint32
	Data   uint32
}

type ImageResourceDataEntry struct {
	Data     uint32
	Size     uint32
	CodePage uint32
	Reserved uint32
}

type ResourceType uint32

// https://msdn.microsoft.com/fr-fr/library/windows/desktop/ms648009(v=vs.85).aspx
const (
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
	ResourceTypeVXD        ResourceType = 20
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

func Do(path string, consumer *state.Consumer) error {
	f, err := eos.Open(path)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer f.Close()

	pf, err := pe.NewFile(f)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer pf.Close()

	props := &mansion.ExePropsResult{}

	switch pf.Machine {
	case pe.IMAGE_FILE_MACHINE_I386:
		props.Arch = "386"
	case pe.IMAGE_FILE_MACHINE_AMD64:
		props.Arch = "amd64"
	}

	rsrcSection := pf.Section(".rsrc")
	if rsrcSection != nil {
		consumer.Logf("Found resource section")
		data, err := rsrcSection.Data()
		if err != nil {
			return errors.Wrap(err, 0)
		}

		var readDirectory func(offset uint32, level int) error
		readDirectory = func(offset uint32, level int) error {
			prefix := strings.Repeat("  ", level)
			log := func(msg string, args ...interface{}) {
				consumer.Logf("%s%s", prefix, fmt.Sprintf(msg, args...))
			}

			br := bytes.NewReader(data[offset:])
			ird := new(ImageResourceDirectory)
			err = binary.Read(br, binary.LittleEndian, ird)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			for i := uint16(0); i < ird.NumberOfNamedEntries+ird.NumberOfIdEntries; i++ {
				irde := new(ImageResourceDirectoryEntry)
				err = binary.Read(br, binary.LittleEndian, irde)
				if err != nil {
					return errors.Wrap(err, 0)
				}

				if irde.NameId&0x80000000 > 0 {
					continue
				} else {
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
				}
				if irde.Data&0x80000000 > 0 {
					offset := irde.Data & 0x7fffffff
					err := readDirectory(offset, level+1)
					if err != nil {
						return errors.Wrap(err, 0)
					}
				} else {
					log("Leaf node!")
				}
			}
			return nil
		}
		err = readDirectory(0, 0)
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	comm.Result(props)

	return nil
}
