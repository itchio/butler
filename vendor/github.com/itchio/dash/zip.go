package dash

import (
	"bufio"
	"io"
	"path/filepath"
	"strings"

	"github.com/itchio/arkive/zip"
)

func sniffZip(r io.ReadSeeker, size int64) (*Candidate, error) {
	ra := &readerAtFromSeeker{r}

	zr, err := zip.NewReader(ra, size)
	if err != nil {
		// not a zip, probably
		return nil, nil
	}

	for _, f := range zr.File {
		path := filepath.ToSlash(filepath.Clean(filepath.ToSlash(f.Name)))
		if path == "META-INF/MANIFEST.MF" {
			rc, err := f.Open()
			if err != nil {
				// :(
				return nil, nil
			}
			defer rc.Close()

			s := bufio.NewScanner(rc)

			for s.Scan() {
				tokens := strings.SplitN(s.Text(), ":", 2)
				if len(tokens) > 0 && tokens[0] == "Main-Class" {
					mainClass := strings.TrimSpace(tokens[1])
					res := &Candidate{
						Flavor: FlavorJar,
						JarInfo: &JarInfo{
							MainClass: mainClass,
						},
					}
					return res, nil
				}
			}

			// we found the manifest, even if we couldn't read it
			// or it didn't have a main class
			break
		}
	}

	return nil, nil
}
