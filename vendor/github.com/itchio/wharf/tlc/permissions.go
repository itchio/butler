package tlc

import (
	"io"

	"github.com/itchio/wharf/wsync"
	"github.com/pkg/errors"
)

const minScannedFileSize = 4

func (c *Container) FixPermissions(pool wsync.Pool) error {
	defer pool.Close()

	buf := make([]byte, minScannedFileSize)
	for index, f := range c.Files {
		if f.Size < minScannedFileSize {
			continue
		}

		r, err := pool.GetReader(int64(index))
		if err != nil {
			return errors.WithStack(err)
		}

		_, err = io.ReadFull(r, buf)
		if err != nil {
			return errors.WithStack(err)
		}

		if isExecutable(buf) {
			f.Mode |= 0111
		}
	}

	return nil
}

// cf. https://github.com/itchio/fnout/blob/master/src/index.js
func isExecutable(buf []byte) bool {
	// intel Mach-O executables start with 0xCEFAEDFE or 0xCFFAEDFE
	// (old PowerPC Mach-O executables started with 0xFEEDFACE)
	if (buf[0] == 0xCE || buf[0] == 0xCF) && buf[1] == 0xFA && buf[2] == 0xED && buf[3] == 0xFE {
		return true
	}

	// Mach-O universal binaries start with 0xCAFEBABE
	// it's Apple's 'fat binary' stuff that contains multiple architectures
	if buf[0] == 0xCA && buf[1] == 0xFE && buf[2] == 0xBA && buf[3] == 0xBE {
		return true
	}

	// ELF executables start with 0x7F454C46
	// (e.g. 0x7F + 'ELF' in ASCII)
	if buf[0] == 0x7F && buf[1] == 0x45 && buf[2] == 0x4C && buf[3] == 0x46 {
		return true
	}

	// Shell scripts start with a shebang (#!)
	// https://en.wikipedia.org/wiki/Shebang_(Unix)
	if buf[0] == 0x23 && buf[1] == 0x21 {
		return true
	}

	return false
}
