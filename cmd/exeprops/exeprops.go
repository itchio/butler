package exeprops

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"unicode/utf16"

	xj "github.com/basgys/goxml2json"
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

// ExeProps contains the architecture of a binary file
//
// For command `exeprops`
type ExeProps struct {
	Arch                string              `json:"arch"`
	VersionProperties   map[string]string   `json:"versionProperties"`
	AssemblyInfo        *AssemblyInfo       `json:"assemblyInfo"`
	DependentAssemblies []*AssemblyIdentity `json:"dependentAssemblies"`
}

type AssemblyInfo struct {
	Identity    *AssemblyIdentity `json:"identity"`
	Description string            `json:"description"`

	RequestedExecutionLevel string `json:"requestedExecutionLevel,omitempty"`
}

type AssemblyIdentity struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Type    string `json:"type"`

	ProcessorArchitecture string `json:"processorArchitecture,omitempty"`
	Language              string `json:"language,omitempty"`
	PublicKeyToken        string `json:"publicKeyToken,omitempty"`
}

func do(ctx *mansion.Context) {
	f, err := eos.Open(*args.path)
	ctx.Must(err)
	defer f.Close()

	props, err := Do(f, comm.NewStateConsumer())
	ctx.Must(err)

	comm.ResultOrPrint(props, func() {
		js, err := json.MarshalIndent(props, "", "  ")
		if err == nil {
			comm.Logf(string(js))
		}
	})

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
	EndOffset   int64

	ReadSeekerAt
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

func Do(f eos.File, consumer *state.Consumer) (*ExeProps, error) {
	pf, err := pe.NewFile(f)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}
	defer pf.Close()

	props := &ExeProps{
		VersionProperties: make(map[string]string),
	}

	switch pf.Machine {
	case pe.IMAGE_FILE_MACHINE_I386:
		props.Arch = "386"
	case pe.IMAGE_FILE_MACHINE_AMD64:
		props.Arch = "amd64"
	}

	rsrcSection := pf.Section(".rsrc")
	if rsrcSection != nil {
		consumer.Debugf("Found resource section")
		data, err := rsrcSection.Data()
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
		consumer.Debugf("Section data size: %s (%d bytes)", humanize.IBytes(uint64(len(data))), len(data))

		var readDirectory func(offset uint32, level int, resourceType ResourceType) error
		readDirectory = func(offset uint32, level int, resourceType ResourceType) error {
			prefix := strings.Repeat("  ", level)
			log := func(msg string, args ...interface{}) {
				consumer.Debugf("%s%s", prefix, fmt.Sprintf(msg, args...))
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
							err := interpretManifest(props, js.Bytes())
							if err != nil {
								log("could not interpret manifest: %s", err.Error())
							}
						}
					case ResourceTypeVersion:
						err := parseVersion(props, consumer, rawData)
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
			return nil, errors.Wrap(err, 0)
		}
	}

	return props, nil
}

// see
// https://msdn.microsoft.com/en-us/library/windows/desktop/dd318693(v=vs.85).aspx
func isLanguageWhitelisted(key string) bool {
	localeID := key[:4]
	primaryLangID := localeID[2:]

	switch primaryLangID {
	// neutral
	case "00":
		return true
	// english
	case "09":
		return true
	}
	return false
}

type ReadSeekerAt interface {
	io.ReadSeeker
	io.ReaderAt
}

func parseVersion(props *ExeProps, consumer *state.Consumer, rawData []byte) error {
	br := bytes.NewReader(rawData)
	buf := make([]byte, 2)

	skipPadding := func(r ReadSeekerAt) error {
		for {
			_, err := r.Read(buf)
			if err != nil {
				if err == io.EOF {
					// alles gut
					return nil
				}
				return errors.Wrap(err, 0)
			}

			if buf[0] != 0 || buf[1] != 0 {
				_, err = r.Seek(-2, io.SeekCurrent)
				if err != nil {
					return errors.Wrap(err, 0)
				}
				break
			}
		}
		return nil
	}

	parseNullTerminatedString := func(r ReadSeekerAt) ([]byte, error) {
		var res []byte

		for {
			_, err := r.Read(buf)
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

	parseVSBlock := func(r ReadSeekerAt) (*VsBlock, error) {
		startOffset, err := r.Seek(0, io.SeekCurrent)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		var wLength uint16
		err = binary.Read(r, binary.LittleEndian, &wLength)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		endOffset := startOffset + int64(wLength)
		sr := io.NewSectionReader(r, startOffset+2, int64(wLength)-2 /* we already read the wLength uint16 */)

		var wValueLength uint16
		err = binary.Read(sr, binary.LittleEndian, &wValueLength)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		var wType uint16
		err = binary.Read(sr, binary.LittleEndian, &wType)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		szKey, err := parseNullTerminatedString(sr)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		err = skipPadding(sr)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		return &VsBlock{
			Length:       wLength,
			ValueLength:  wValueLength,
			Type:         wType,
			Key:          szKey,
			EndOffset:    endOffset,
			ReadSeekerAt: sr,
		}, nil
	}

	vsVersionInfo, err := parseVSBlock(br)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	if vsVersionInfo.ValueLength == 0 {
		return nil
	}

	ffi := new(VsFixedFileInfo)
	err = binary.Read(vsVersionInfo, binary.LittleEndian, ffi)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	if ffi.DwSignature != 0xFEEF04BD {
		consumer.Debugf("invalid signature, either the version block is invalid or we messed up")
		return nil
	}

	err = skipPadding(vsVersionInfo)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	for {
		fileInfo, err := parseVSBlock(vsVersionInfo)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return errors.Wrap(err, 0)
		}

		switch fileInfo.KeyString() {
		case "StringFileInfo":
			for {
				stable, err := parseVSBlock(fileInfo)
				if err != nil {
					if errors.Is(err, io.EOF) {
						break
					}
					return errors.Wrap(err, 0)
				}

				if isLanguageWhitelisted(stable.KeyString()) {
					for {
						str, err := parseVSBlock(stable)
						if err != nil {
							if errors.Is(err, io.EOF) {
								break
							}
							return errors.Wrap(err, 0)
						}

						keyString := str.KeyString()

						val, err := parseNullTerminatedString(str)
						if err != nil {
							return errors.Wrap(err, 0)
						}
						valString := strings.TrimSpace(DecodeUTF16(val))

						consumer.Debugf("%s: %s", keyString, valString)
						props.VersionProperties[keyString] = valString
						_, err = stable.Seek(str.EndOffset, io.SeekStart)
						if err != nil {
							return errors.Wrap(err, 0)
						}

						err = skipPadding(stable)
						if err != nil {
							return errors.Wrap(err, 0)
						}
					}
				}

				_, err = fileInfo.Seek(stable.EndOffset, io.SeekStart)
				if err != nil {
					return errors.Wrap(err, 0)
				}
			}
		case "VarFileInfo":
			// skip
		}

		_, err = vsVersionInfo.Seek(fileInfo.EndOffset, io.SeekStart)
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	return nil
}

type node = map[string]interface{}

func visit(n node, key string, f func(c node)) {
	if c, ok := n[key].(node); ok {
		f(c)
	}
}

func visitMany(n node, key string, f func(c node)) {
	if cs, ok := n[key].([]node); ok {
		for _, c := range cs {
			f(c)
		}
	}
	if c, ok := n[key].(node); ok {
		f(c)
	}
}

func getString(n node, key string, f func(s string)) {
	if s, ok := n[key].(string); ok {
		f(s)
	}
}

func interpretManifest(props *ExeProps, manifest []byte) error {
	intermediate := make(node)
	err := json.Unmarshal([]byte(manifest), &intermediate)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	info := &AssemblyInfo{}

	interpretIdentity := func(id node, f func(id *AssemblyIdentity)) {
		ai := &AssemblyIdentity{}
		getString(id, "-name", func(s string) { ai.Name = s })
		getString(id, "-version", func(s string) { ai.Version = s })
		getString(id, "-type", func(s string) { ai.Type = s })

		getString(id, "-processorArchitecture", func(s string) { ai.ProcessorArchitecture = s })
		getString(id, "-publicKeyToken", func(s string) { ai.PublicKeyToken = s })
		getString(id, "-language", func(s string) { ai.Language = s })
		f(ai)
	}

	visit(intermediate, "assembly", func(assembly node) {
		visit(assembly, "assemblyIdentity", func(id node) {
			interpretIdentity(id, func(ai *AssemblyIdentity) {
				info.Identity = ai
			})
		})

		getString(assembly, "description", func(s string) { info.Description = s })

		visit(assembly, "trustInfo", func(ti node) {
			visit(ti, "security", func(sec node) {
				visit(sec, "requestedPrivileges", func(rp node) {
					visit(rp, "requestedExecutionLevel", func(rel node) {
						getString(rel, "-level", func(s string) {
							info.RequestedExecutionLevel = s
						})
					})
				})
			})
		})

		visit(assembly, "dependency", func(dep node) {
			visitMany(dep, "dependentAssembly", func(da node) {
				visit(da, "assemblyIdentity", func(id node) {
					interpretIdentity(id, func(ai *AssemblyIdentity) {
						props.DependentAssemblies = append(props.DependentAssemblies, ai)
					})
				})
			})
		})
	})

	props.AssemblyInfo = info

	return nil
}
