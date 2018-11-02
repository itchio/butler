package dmgextract

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/itchio/butler/filtering"
	"github.com/itchio/damage"
	"github.com/itchio/damage/hdiutil"
	"github.com/itchio/wharf/pools/fspool"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/state"
	"github.com/itchio/wharf/tlc"
	"github.com/pkg/errors"
)

// An Extractor can list the contents of a DMG and extract it to a folder
type Extractor interface {
	List() (*ExtractorResult, error)
	ExtractTo(dest string) (*ExtractorResult, error)
}

type extractor struct {
	opts extractorOpts
}

type extractorOpts struct {
	dmgpath     string
	mountFolder string
	consumer    *state.Consumer
	extractSLA  bool
}

// An ExtractorOpt changes the behavior of a dmg extractor.
type ExtractorOpt func(opts *extractorOpts)

// ExtractorResult contains information aobut
type ExtractorResult struct {
	Container *tlc.Container
	SLA       *damage.SLA
}

// WithConsumer hooks up logging and progress to a custom consumer. Otherwise,
// a silent consumer will be used.
func WithConsumer(consumer *state.Consumer) ExtractorOpt {
	return func(opts *extractorOpts) {
		opts.consumer = consumer
	}
}

// WithMountFolder specifies a mount folder for the disk image.
// If none is specified, a temporary path will be generated.
func WithMountFolder(mountFolder string) ExtractorOpt {
	return func(opts *extractorOpts) {
		opts.mountFolder = mountFolder
	}
}

// ExtractSLA instructs the dmg extractor to parse any service
// level agreement stored in the DMG file.
var ExtractSLA ExtractorOpt = func(opts *extractorOpts) {
	opts.extractSLA = true
}

// New creates a new extractor with the specified options.
// It doesn't do anything until one of its methods are called.
func New(dmgpath string, o ...ExtractorOpt) Extractor {
	opts := extractorOpts{
		dmgpath: dmgpath,
	}
	for _, applyOpt := range o {
		applyOpt(&opts)
	}

	return &extractor{opts: opts}
}

func (e *extractor) List() (*ExtractorResult, error) {
	res, err := e.do("")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return res, nil
}

func (e *extractor) ExtractTo(dest string) (*ExtractorResult, error) {
	return e.do(dest)
}

type unmountMode int

const (
	unmountModeSilent unmountMode = 0
	unmountModeNoisy  unmountMode = 1
)

func (e *extractor) do(dest string) (*ExtractorResult, error) {
	res := &ExtractorResult{}
	opts := e.opts
	consumer := opts.consumer
	if consumer == nil {
		consumer = &state.Consumer{}
	}

	host := hdiutil.NewHost(consumer)

	consumer.Opf("Analyzing DMG file...")
	info, err := damage.GetDiskInfo(host, opts.dmgpath)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	consumer.Statf("%s", info)

	if info.Properties.SoftwareLicenseAgreement {
		consumer.Infof("Found SLA (Software License Agreement)")

		if opts.extractSLA {
			consumer.Opf("Attempting to retrieve SLA text...")

			fetchSLA := func() error {
				rez, err := damage.GetUDIFResources(host, opts.dmgpath)
				if err != nil {
					return errors.WithStack(err)
				}

				sla, err := damage.GetDefaultSLA(rez)
				if err != nil {
					return errors.WithStack(err)
				}

				if sla != nil {
					text := sla.Text
					consumer.Statf("Fetched SLA text (%d bytes)", len(text))
					res.SLA = sla
				} else {
					consumer.Infof("No SLA text found")
				}
				return nil
			}
			err = fetchSLA()
			if err != nil {
				consumer.Warnf("While fetching SLA: %+v", err)
				consumer.Infof("Continuing anyway...")
			}
		}
	}

	mountFolder := opts.mountFolder
	if mountFolder == "" {
		mountFolder, err = ioutil.TempDir("", "dmg-mountpoint")
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	consumer.Opf("Mounting...")
	unmount := func(mode unmountMode) {
		// if this gets called we're in cleanup mode
		err := damage.Unmount(host, mountFolder)
		if err != nil && mode == unmountModeNoisy {
			consumer.Warnf("While unmounting: %+v", err)
		}
	}
	// Q: why defer unmount before we mount?
	// well, damage.Mount is a multi-operation call. First it actually
	// calls hdiutil mount, then it parses its result. Maybe it mounted
	// successfully but failed to parse the result. In any case, we don't
	// want to be leaving a mounted volume behind us.
	defer unmount(unmountModeSilent)

	_, err = damage.Mount(host, opts.dmgpath, mountFolder)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	consumer.Statf("Mounted!")

	// should return "false" if file should be excluded from walk
	dmgFilterPaths := func(fileInfo os.FileInfo) bool {
		if !filtering.FilterPaths(fileInfo) {
			return false
		}

		name := fileInfo.Name()
		if strings.Contains(name, "Applications") && fileInfo.Mode()&os.ModeSymlink > 0 {
			// DMG files tend to have a symlink to /Applications in there, we don't
			// want to install that, ohh, no we don't.
			return false
		}
		return true
	}

	consumer.Opf("Walking mounted volume...")

	container, err := tlc.WalkDir(mountFolder, &tlc.WalkOpts{
		Dereference: false,
		Filter:      dmgFilterPaths,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	consumer.Statf("Found %s", container.Stats())

	if dest == "" {
		consumer.Infof("Done! (should not extract)")
	} else {
		consumer.Opf("Preparing dirs and symlinks in (%s)...", dest)
		err = container.Prepare(dest)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		consumer.Infof("Copying to (%s)", dest)
		copy := func() error {
			inPool := fspool.New(container, mountFolder)
			defer inPool.Close()

			outPool := fspool.New(container, dest)
			return pwr.CopyContainer(container, outPool, inPool, consumer)
		}
		err = copy()
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	unmount(unmountModeNoisy)

	res.Container = container
	return res, nil
}
