package exeprops

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"unicode/utf16"

	humanize "github.com/dustin/go-humanize"
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

type VsBlock struct {
	Length      uint16
	ValueLength uint16
	Type        uint16
	Key         []byte
}

func DecodeUTF16(bs []byte) string {
	ints := make([]uint16, len(bs)/2)
	for i := 0; i < len(ints); i++ {
		ints[i] = binary.LittleEndian.Uint16(bs[i*2 : (i+1)*2])
	}
	return string(utf16.Decode(ints))
}

func (vb *VsBlock) KeyString() string {
	return DecodeUTF16(vb.Key)
}

type VsFixedFileInfo struct {
	DwSignature        uint32
	DwStrucVersion     uint32
	DwFileVersionMS    uint32
	DwFileVersionLS    uint32
	DwProductVersionMS uint32
	DwProductVersionLS uint32
	DwFileFlagsMask    uint32
	DwFileFlags        uint32
	DwFileOS           uint32
	DwFileType         uint32
	DwFileSubtype      uint32
	DwFileDateMS       uint32
	DwFileDateLS       uint32
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
		consumer.Logf("Section data size: %s (%d bytes)", humanize.IBytes(uint64(len(data))), len(data))

		var readDirectory func(offset uint32, level int, resourceType ResourceType) error
		readDirectory = func(offset uint32, level int, resourceType ResourceType) error {
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
						return errors.Wrap(err, 0)
					}
					continue
				}

				dbr := bytes.NewReader(data[irde.Data:])

				irda := new(ImageResourceDataEntry)
				err = binary.Read(dbr, binary.LittleEndian, irda)
				if err != nil {
					return errors.Wrap(err, 0)
				}

				if resourceType == ResourceTypeManifest || resourceType == ResourceTypeVersion {
					log("@ %x (%s, %d bytes)", irda.Data, humanize.IBytes(uint64(irda.Size)), irda.Size)

					sr := io.NewSectionReader(f, int64(rsrcSection.Offset+irda.Data-rsrcSection.VirtualAddress), int64(irda.Size))
					rawData, err := ioutil.ReadAll(sr)
					if err != nil {
						return errors.Wrap(err, 0)
					}

					switch resourceType {
					case ResourceTypeManifest:
						// actually not utf-16 for some
						// reason
						stringData := string(rawData)
						// log("=========================")
						// for _, l := range strings.Split(stringData, "\n") {
						// 	log("%s", l)
						// }
						// log("=========================")
						props.Manifest = stringData
					case ResourceTypeVersion:
						err := parseVersion(consumer, rawData)
						if err != nil {
							return errors.Wrap(err, 0)
						}
					}
				}
			}
			return nil
		}
		err = readDirectory(0, 0, 0)
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	comm.Result(props)

	return nil
}

func parseVersion(consumer *state.Consumer, rawData []byte) error {
	br := bytes.NewReader(rawData)
	buf := make([]byte, 2)

	skipPadding := func() error {
		for {
			_, err := br.Read(buf)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			if buf[0] != 0 || buf[1] != 0 {
				_, err = br.Seek(-2, io.SeekCurrent)
				if err != nil {
					return errors.Wrap(err, 0)
				}
				break
			}
		}
		return nil
	}

	parseNullTerminatedString := func() ([]byte, error) {
		var res []byte

		for {
			_, err := br.Read(buf)
			if err != nil {
				return nil, errors.Wrap(err, 0)
			}

			if buf[0] == 0 && buf[1] == 0 {
				break
			} else {
				res = append(res, buf...)
			}
		}
		return res, nil
	}

	parseVSBlock := func() (*VsBlock, error) {
		var wLength uint16
		err := binary.Read(br, binary.LittleEndian, &wLength)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		consumer.Logf("wLength = %d", wLength)

		var wValueLength uint16
		err = binary.Read(br, binary.LittleEndian, &wValueLength)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
		consumer.Logf("wValueLength = %d", wValueLength)

		var wType uint16
		err = binary.Read(br, binary.LittleEndian, &wType)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
		consumer.Logf("wType = %d", wType)

		szKey, err := parseNullTerminatedString()
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		err = skipPadding()
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		return &VsBlock{
			Length:      wLength,
			ValueLength: wValueLength,
			Type:        wType,
			Key:         szKey,
		}, nil
	}

	vsVersionInfo, err := parseVSBlock()
	if err != nil {
		return errors.Wrap(err, 0)
	}
	consumer.Logf("vsVersionInfo key = %s", vsVersionInfo.KeyString())

	if vsVersionInfo.ValueLength == 0 {
		consumer.Logf("no value, skipping")
		return nil
	}

	ffi := new(VsFixedFileInfo)
	err = binary.Read(br, binary.LittleEndian, ffi)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	consumer.Logf("ffi: %#v", *ffi)

	if ffi.DwSignature != 0xFEEF04BD {
		consumer.Logf("invalid signature, either the version block is invalid or we messed up")
		return nil
	}

	var fileVersion = int64(ffi.DwFileVersionMS)<<32 + int64(ffi.DwFileVersionLS)
	consumer.Logf("file version:    %d", fileVersion)
	var productVersion = int64(ffi.DwProductVersionMS)<<32 + int64(ffi.DwProductVersionLS)
	consumer.Logf("product version: %d", productVersion)

	err = skipPadding()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	child, err := parseVSBlock()
	consumer.Logf("child key: %s", child.KeyString())
	switch child.KeyString() {
	case "StringFileInfo":
		stable, err := parseVSBlock()
		if err != nil {
			return errors.Wrap(err, 0)
		}

		consumer.Logf("found string table of length %d, key = %s", stable.Length, stable.KeyString())

		for {
			str, err := parseVSBlock()
			if err != nil {
				return errors.Wrap(err, 0)
			}
			consumer.Logf("found string, key = %s", str.KeyString())

			val, err := parseNullTerminatedString()
			if err != nil {
				return errors.Wrap(err, 0)
			}
			consumer.Logf("value = %s", DecodeUTF16(val))
		}
	case "VarFileInfo":
		consumer.Logf("not supported, skipping")
		return nil
	}

	return nil
}
