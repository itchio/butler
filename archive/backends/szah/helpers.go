package szah

import (
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/arkive/zip"
	"github.com/itchio/sevenzip-go/sz"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
	"github.com/nightlyone/lockfile"
)

var dontEnsureDeps = os.Getenv("BUTLER_NO_DEPS") == "1"
var ensuredDeps = false

type withArchiveCallback func(a *sz.Archive) error

func withArchive(consumer *state.Consumer, path string, cb withArchiveCallback) error {
	err := ensureDeps(consumer)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	lib, err := sz.NewLib()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	defer lib.Free()

	f, err := eos.Open(path)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	ext := filepath.Ext(path)
	if ext != "" {
		ext = ext[1:] // strip "."
	}

	in, err := sz.NewInStream(f, ext, info.Size())
	if err != nil {
		return errors.Wrap(err, 0)
	}

	// try by extension first
	a, err := lib.OpenArchive(in, false)
	if err != nil {
		// try by signature next
		_, err = in.Seek(0, io.SeekStart)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		a, err = lib.OpenArchive(in, true)
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	return cb(a)
}

type tempLockfileErr interface {
	Temporary() bool
}

func ensureDeps(consumer *state.Consumer) error {
	if dontEnsureDeps {
		consumer.Debugf("Asked not to ensure dependencies, skipping...")
		return nil
	}

	if ensuredDeps {
		return nil
	}

	consumer.Debugf("Ensuring dependencies...")
	depSpec := getDepSpec()
	if depSpec == nil {
		consumer.Debugf("No dependencies for %s-%s", runtime.GOOS, runtime.GOARCH)
		return nil
	}

	log.Printf("depSpec:\n%#v\n", depSpec)

	execPath, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, 0)
	}
	execDir := filepath.Dir(execPath)

	lockFilePath := filepath.Join(execDir, ".butler-deps.lock")
	lf, err := lockfile.New(lockFilePath)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = lf.TryLock()
	tries := 10
	for err != nil {
		if err == lockfile.ErrBusy {
			time.Sleep(2 * time.Second)
			err = lf.TryLock()
			continue
		}

		// lockfile's recommended way to look for a temporary error
		if _, ok := err.(tempLockfileErr); ok {
			consumer.Debugf("Will retry acquiring lock in a few: %s", err.Error())
			tries -= 1
			if tries <= 0 {
				msg := fmt.Sprintf("Too many errors acquiring lock, giving up. Last error was: %s", err.Error())
				return errors.New(msg)
			}

			time.Sleep(2 * time.Second)
		}
	}
	defer lf.Unlock()

	var toFetch []DepEntry

	for _, entry := range depSpec.entries {
		func() {
			entryPath := filepath.Join(execDir, entry.name)

			f, err := os.Open(entryPath)
			if err != nil {
				consumer.Debugf("")
				consumer.Debugf("[%s] could not open, will fetch", entry.name)
				if !os.IsNotExist(err) {
					consumer.Debugf("  %s", err.Error())
				}
				toFetch = append(toFetch, entry)
				return
			}

			hashes := make(map[HashAlgo]hash.Hash)
			for _, dh := range entry.hashes {
				switch dh.algo {
				case HashAlgoSHA1:
					hashes[dh.algo] = sha1.New()
				case HashAlgoSHA256:
					hashes[dh.algo] = sha256.New()
				}
			}

			if len(hashes) == 0 {
				consumer.Debugf("No hashes to check, calling it a day.")
				return
			}

			// oh go, I'm so disappoint in you right now
			var writers []io.Writer
			for _, h := range hashes {
				writers = append(writers, h)
			}
			mw := io.MultiWriter(writers...)

			_, err = io.Copy(mw, f)
			if err != nil {
				consumer.Debugf("Error during hashing of %s, will fetch: %s", entry.name, err.Error())
				toFetch = append(toFetch, entry)
				return
			}

			for _, dh := range entry.hashes {
				h := hashes[dh.algo]
				if h != nil {
					expected := dh.value
					// yes, yes, bytes.Equal is a thing. but also
					// []byte{} literals are not the friendliest. don't @ me.
					actual := fmt.Sprintf("%x", h.Sum(nil))
					if actual != expected {
						consumer.Debugf("")
						consumer.Debugf("[%s] %s hash mismatch, will fetch", entry.name, dh.algo)
						consumer.Debugf("  wanted %s", expected)
						consumer.Debugf("     got %s", actual)
						toFetch = append(toFetch, entry)
						return
					}
				}
			}
		}()
	}

	if len(toFetch) > 0 {
		consumer.Logf("")
		consumer.Opf("Healing %d dependencies...", len(toFetch))

		firstSource := true
		for _, source := range depSpec.sources {
			if !firstSource {
				consumer.Logf("Trying next source...")
			}

			firstSource = false
			err = func() error {
				f, err := eos.Open(source)
				if err != nil {
					return errors.Wrap(err, 0)
				}
				defer f.Close()

				stats, err := f.Stat()
				if err != nil {
					return errors.Wrap(err, 0)
				}

				zr, err := zip.NewReader(f, stats.Size())
				if err != nil {
					return errors.Wrap(err, 0)
				}

				foundFiles := 0
				var installedSize int64
				for _, zf := range zr.File {
					for _, entry := range toFetch {
						if entry.name == zf.Name {
							foundFiles++
							consumer.Opf("%s (%s)...", entry.name, humanize.IBytes(uint64(zf.UncompressedSize64)))
							entryPath := filepath.Join(execDir, entry.name)

							err = func() error {
								zer, err := zf.Open()
								if err != nil {
									return errors.Wrap(err, 0)
								}
								defer zer.Close()

								of, err := os.Create(entryPath)
								if err != nil {
									return errors.Wrap(err, 0)
								}
								defer of.Close()

								writtenBytes, err := io.Copy(of, zer)
								if err != nil {
									return errors.Wrap(err, 0)
								}

								installedSize += writtenBytes
								return nil
							}()

							if err != nil {
								return errors.Wrap(err, 0)
							}
						}
					}
				}

				if foundFiles < len(toFetch) {
					return errors.Wrap(fmt.Errorf("Found only %d files of the required %d", foundFiles, len(toFetch)), 0)
				}
				consumer.Statf("Installed %s's worth of dependencies", humanize.IBytes(uint64(installedSize)))

				return nil
			}()

			if err != nil {
				consumer.Logf("Error while installing dependencies: %s", err.Error())
				continue
			}
			break
		}
		consumer.Logf("")
	}

	ensuredDeps = true
	return nil
}
