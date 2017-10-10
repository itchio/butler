package xad

import (
	"bufio"
	"os/exec"
	"regexp"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/archive"
)

var extractRe = regexp.MustCompile(`^ {2}(.+) {2}\(.+\)\.\.\. OK\.$`)

func (h *Handler) Extract(params *archive.ExtractParams) error {
	cmd := exec.Command(
		"unar",
		// Always overwrite files when a file to be unpacked already exists on disk.
		// By default, the program asks the user if possible, otherwise skips the file.
		"-force-overwrite",
		// Never create a containing directory for the contents of the unpacked archive.
		"-no-directory",
		// The directory to write the contents of the archive to. Defaults to the
		// current directory. If set to a single dash (-), no files will be created,
		// and all data will be output to stdout.
		"-output-directory",
		params.OutputPath,
		params.Path,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = cmd.Start()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	totalSize := int64(0)
	sizeMap := make(map[string]int64)
	for _, entry := range params.ListResult.Entries() {
		sizeMap[entry.Name] = entry.UncompressedSize
		totalSize += entry.UncompressedSize
	}
	var doneSize int64

	s := bufio.NewScanner(stdout)

	go func() {
		for s.Scan() {
			line := s.Text()
			submatches := extractRe.FindStringSubmatch(line)
			if submatches != nil {
				if len(submatches) >= 2 {
					path := submatches[1]
					if size, ok := sizeMap[archive.CleanFileName(path)]; ok {
						doneSize += size
						params.Consumer.Progress(float64(doneSize) / float64(totalSize))
					}
				}
			}
		}

		// TODO: error handling?
	}()

	err = cmd.Wait()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	params.Consumer.Progress(1.0)

	return nil
}
