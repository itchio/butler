package prereqs

import (
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/itchio/savior/seeksource"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/pwr"

	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/progress"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/redist"
	"github.com/itchio/wharf/state"
)

const RedistsBaseURL = `https://dl.itch.ovh/itch-redists`

type TaskStateConsumer struct {
	OnState func(state *buse.PrereqsTaskStateNotification)
}

func FetchPrereqs(library Library, consumer *state.Consumer, tsc *TaskStateConsumer, folder string, redistRegistry *redist.RedistRegistry, names []string) error {
	doPrereq := func(name string) error {
		entry := redistRegistry.Entries[name]
		if entry == nil {
			consumer.Warnf("Prereq (%s) not found in registry, skipping")
			return nil
		}
		destDir := filepath.Join(folder, name)

		doDownload := func() error {
			tsc.OnState(&buse.PrereqsTaskStateNotification{
				Name:   name,
				Status: buse.PrereqStatusDownloading,
			})

			signatureURL, err := library.GetURL(name, "signature")
			if err != nil {
				return errors.Wrap(err, 0)
			}
			archiveURL, err := library.GetURL(name, "archive")
			if err != nil {
				return errors.Wrap(err, 0)
			}
			healSpec := fmt.Sprintf("archive,%s", archiveURL)

			consumer.Infof("Extracting (%s) to (%s)", name, destDir)

			counter := progress.NewCounter()
			counter.Start()

			cancel := make(chan struct{})
			defer close(cancel)

			go func() {
				for {
					select {
					case <-time.After(1 * time.Second):
						tsc.OnState(&buse.PrereqsTaskStateNotification{
							Name:     name,
							Status:   buse.PrereqStatusDownloading,
							Progress: counter.Progress(),
							ETA:      counter.ETA().Seconds(),
							BPS:      counter.BPS(),
						})
					case <-cancel:
						return
					}
				}
			}()

			sigFile, err := eos.Open(signatureURL)
			if err != nil {
				return errors.Wrap(err, 0)
			}
			defer sigFile.Close()

			sigSource := seeksource.FromFile(sigFile)
			_, err = sigSource.Resume(nil)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			sig, err := pwr.ReadSignature(sigSource)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			err = os.MkdirAll(destDir, 0755)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			subconsumer := &state.Consumer{
				OnProgress: func(progress float64) {
					counter.SetProgress(progress)
				},
				OnMessage: func(level string, message string) {
					consumer.OnMessage(level, message)
				},
			}

			vctx := pwr.ValidatorContext{
				Consumer:   subconsumer,
				NumWorkers: 1,
				HealPath:   healSpec,
			}

			err = vctx.Validate(destDir, sig)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			return nil
		}

		// first check if we already have it
		needFetch := false

		if len(entry.Files) == 0 {
			consumer.Warnf("Entry %s is missing files info, forcing fetch...")
			needFetch = true
		} else {
			for _, f := range entry.Files {
				filePath := filepath.Join(destDir, f.Name)

				err := func() error {
					stats, err := os.Stat(filePath)
					if err != nil {
						return errors.Wrap(err, 0)
					}

					if stats.Size() != f.Size {
						return fmt.Errorf("expected size: %d, actual: %d", f.Size, stats.Size())
					}

					r, err := os.Open(filePath)
					if err != nil {
						return errors.Wrap(err, 0)
					}

					defer r.Close()

					checkHash := func(expectedHash string, hasher hash.Hash) error {
						if expectedHash == "" {
							return errors.New("missing expected hash")
						}

						_, err := r.Seek(0, io.SeekStart)
						if err != nil {
							return errors.Wrap(err, 0)
						}

						hasher.Reset()

						_, err = io.Copy(hasher, r)
						if err != nil {
							return err
						}

						actualHash := fmt.Sprintf("%x", hasher.Sum(nil))
						if actualHash != expectedHash {
							return fmt.Errorf("expected hash: %s, actual: %s", expectedHash, actualHash)
						}

						return nil
					}

					err = checkHash(f.SHA1, sha1.New())
					if err != nil {
						return errors.Wrap(err, 0)
					}
					err = checkHash(f.SHA256, sha256.New())
					if err != nil {
						return errors.Wrap(err, 0)
					}

					// alright, we good!
					return nil
				}()

				if err != nil {
					consumer.Warnf("(%s): %s, forcing fetch...", filePath, err.Error())
					needFetch = true
					break
				}
			}
		}

		if needFetch {
			err := doDownload()
			if err != nil {
				return errors.Wrap(err, 0)
			}
		} else {
			consumer.Infof("(%s): found in cache and up-to-date", name)
		}

		tsc.OnState(&buse.PrereqsTaskStateNotification{
			Name:   name,
			Status: buse.PrereqStatusReady,
		})

		return nil
	}

	for _, name := range names {
		err := doPrereq(name)
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	return nil
}

func getBaseURL(name string) string {
	return fmt.Sprintf("%s/%s", RedistsBaseURL, name)
}
