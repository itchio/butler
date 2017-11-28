package eos

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"github.com/go-errors/errors"
)

type CheckingFile struct {
	Reference File
	Trainee   File
}

var _ File = (*CheckingFile)(nil)

func (cf *CheckingFile) Close() error {
	err1 := cf.Reference.Close()
	err2 := cf.Trainee.Close()

	if err1 != nil {
		return err1
	}

	if err2 != nil {
		return err2
	}
	return nil
}

func (cf *CheckingFile) Read(tBuf []byte) (int, error) {
	rBuf := make([]byte, len(tBuf))
	rReadBytes, rErr := cf.Reference.Read(rBuf)

	tReadBytes, tErr := cf.Trainee.Read(tBuf)

	if rErr != nil {
		if tErr != nil {
			log.Printf("reference error: %s", rErr.Error())
			log.Printf("  trainee error: %s", tErr.Error())
			// cool, we'll return that at the end
		} else {
			must(errors.Wrap(fmt.Errorf("reference had error %s, trainee had no error", rErr.Error()), 0))
		}
	} else {
		if tErr != nil {
			must(errors.Wrap(fmt.Errorf("reference had no error, trainee had error %s", tErr.Error()), 0))
		}
	}

	if rReadBytes != tReadBytes {
		must(errors.Wrap(fmt.Errorf("reference read %d bytes, trainee read %d", rReadBytes, tReadBytes), 0))
	}

	if !bytes.Equal(rBuf[:rReadBytes], tBuf[:rReadBytes]) {
		must(errors.Wrap(fmt.Errorf("reference read %d bytes, trainee read %d", rReadBytes, tReadBytes), 0))
	}

	return tReadBytes, tErr
}

func (cf *CheckingFile) ReadAt(tBuf []byte, offset int64) (int, error) {
	rBuf := make([]byte, len(tBuf))
	rReadBytes, rErr := cf.Reference.ReadAt(rBuf, offset)

	tReadBytes, tErr := cf.Trainee.ReadAt(tBuf, offset)

	if rErr != nil {
		if tErr != nil {
			log.Printf("reference error: %s", rErr.Error())
			log.Printf("  trainee error: %s", tErr.Error())
			// cool, we'll return that later
		} else {
			must(errors.Wrap(fmt.Errorf("reference had error %s, trainee had no error", rErr.Error()), 0))
		}
	} else {
		if tErr != nil {
			must(errors.Wrap(fmt.Errorf("reference had no error, trainee had error %s", tErr.Error()), 0))
		}
	}

	if rReadBytes != tReadBytes {
		must(errors.Wrap(fmt.Errorf("reference read %d bytes, trainee read %d", rReadBytes, tReadBytes), 0))
	}

	if !bytes.Equal(rBuf[:rReadBytes], tBuf[:rReadBytes]) {
		firstDifferent := rReadBytes
		for i := 0; i < rReadBytes; i++ {
			if rBuf[i] != tBuf[i] {
				firstDifferent = i
				break
			}
		}

		numDifferent := 0
		for i := 0; i < rReadBytes; i++ {
			if rBuf[i] != tBuf[i] {
				numDifferent += 1
			}
		}

		must(errors.Wrap(fmt.Errorf("reference & trainee read different bytes, first different is at %d / %d, %d bytes are different", firstDifferent, rReadBytes, numDifferent), 0))
	}

	return tReadBytes, tErr
}

func (cf *CheckingFile) Seek(offset int64, whence int) (int64, error) {
	rOffset, rErr := cf.Reference.Seek(offset, whence)
	tOffset, tErr := cf.Trainee.Seek(offset, whence)

	if rErr != nil {
		if tErr != nil {
			log.Printf("reference error: %s", rErr.Error())
			log.Printf("  trainee error: %s", tErr.Error())
			// cool, we'll return that at the end
		} else {
			must(errors.Wrap(fmt.Errorf("reference had error %s, trainee had no error", rErr.Error()), 0))
		}
	} else {
		if tErr != nil {
			must(errors.Wrap(fmt.Errorf("reference had no error, trainee had error %s", tErr.Error()), 0))
		}
	}

	if rOffset != tOffset {
		must(errors.Wrap(fmt.Errorf("reference seeked to %d, trainee seeked to %d", rOffset, tOffset), 0))
	}

	return tOffset, tErr
}

func (cf *CheckingFile) Stat() (os.FileInfo, error) {
	_, rErr := cf.Reference.Stat()
	tStat, tErr := cf.Trainee.Stat()

	if rErr != nil {
		if tErr != nil {
			log.Printf("reference error: %s", rErr.Error())
			log.Printf("  trainee error: %s", tErr.Error())
			// cool, we'll return that at the end
		} else {
			must(errors.Wrap(fmt.Errorf("reference had error %s, trainee had no error", rErr.Error()), 0))
		}
	} else {
		if tErr != nil {
			must(errors.Wrap(fmt.Errorf("reference had no error, trainee had error %s", tErr.Error()), 0))
		}
	}

	return tStat, tErr
}

func must(err error) {
	if err != nil {
		switch err := err.(type) {
		case *errors.Error:
			panic(err.ErrorStack())
		default:
			panic(err.Error())
		}
	}
}
