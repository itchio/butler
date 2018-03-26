package dl

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/itchio/wharf/crc32c"
	"github.com/itchio/wharf/state"
	"github.com/pkg/errors"
)

// BadSizeErr is returned when the size of a file on disk doesn't match what we expected it to be
type BadSizeErr struct {
	Expected int64
	Actual   int64
}

func (bse *BadSizeErr) Error() string {
	return fmt.Sprintf("size on disk didn't match expected size: wanted %d, got %d", bse.Expected, bse.Actual)
}

// BadHashErr is returned when the hash of a file on disk doesn't match what we expected it to be
type BadHashErr struct {
	Algo     string
	Expected []byte
	Actual   []byte
}

func (bhe *BadHashErr) Error() string {
	return fmt.Sprintf("%s hash mismatch: wanted %x, got %x", bhe.Algo, bhe.Expected, bhe.Actual)
}

// IsIntegrityError returns true if the error is a size or a hash mismatch.
// Simple reference equality cannot be used because the error might be wrapped (for stack traces)
func IsIntegrityError(err error) bool {
	cause := errors.Cause(err)

	if _, ok := cause.(*BadSizeErr); ok {
		return true
	}
	if _, ok := cause.(*BadHashErr); ok {
		return true
	}

	return false
}

func CheckIntegrity(consumer *state.Consumer, header http.Header, contentLength int64, file string) error {
	diskSize := int64(0)
	stats, err := os.Lstat(file)
	if err == nil {
		diskSize = stats.Size()
	}

	// some servers will return a negative content-length, or 0
	// they both mostly mean they didn't know the length of the response
	// at the time the request was made (streaming proxies, for example)
	if contentLength > 0 {
		if diskSize != contentLength {
			return &BadSizeErr{
				Expected: contentLength,
				Actual:   diskSize,
			}
		}
		consumer.Debugf("%10s pass (%d bytes)", "size", diskSize)
	}

	return checkHashes(consumer, header, file)
}

func checkHashes(consumer *state.Consumer, header http.Header, file string) error {
	googHashes := header[http.CanonicalHeaderKey("x-goog-hash")]

	for _, googHash := range googHashes {
		tokens := strings.SplitN(googHash, "=", 2)
		hashType := tokens[0]
		hashValue, err := base64.StdEncoding.DecodeString(tokens[1])
		if err != nil {
			consumer.Infof("Could not verify %s hash: %s", hashType, err)
			continue
		}

		start := time.Now()
		checked, err := checkHash(hashType, hashValue, file)
		if err != nil {
			consumer.Warnf("%10s fail: %s", hashType, err.Error())
			return errors.Wrapf(err, "checking %s hash", hashType)
		}

		if checked {
			consumer.Debugf("%10s pass (took %s)", hashType, time.Since(start))
		} else {
			consumer.Debugf("%10s skip", hashType)
		}
	}

	return nil
}

func checkHash(hashType string, hashValue []byte, file string) (checked bool, err error) {
	checked = true

	switch hashType {
	case "crc32c":
		err = checkHashCRC32C(hashValue, file)
	default:
		checked = false
	}

	if err != nil {
		err = errors.WithStack(err)
	}
	return
}

func checkHashCRC32C(hashValue []byte, file string) error {
	fr, err := os.Open(file)
	if err != nil {
		return errors.WithStack(err)
	}
	defer fr.Close()

	hasher := crc32.New(crc32c.Table)
	io.Copy(hasher, fr)

	hashComputed := hasher.Sum(nil)
	if !bytes.Equal(hashValue, hashComputed) {
		return &BadHashErr{
			Algo:     "crc32c",
			Actual:   hashComputed,
			Expected: hashValue,
		}
	}

	return nil
}
