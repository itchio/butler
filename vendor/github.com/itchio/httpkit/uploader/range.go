package uploader

import (
	"fmt"
	"strconv"
	"strings"
)

type httpRange struct {
	start int64
	end   int64
}

func parseRangeHeader(rangeHeader string) (r *httpRange, err error) {
	keyval := strings.Split(rangeHeader, "=")
	if len(keyval) != 2 {
		err = fmt.Errorf("invalid range header, expected \"key=val\", got \"%s\"", rangeHeader)
		return
	}
	val := keyval[1]

	startEnd := strings.Split(val, "-")
	if len(startEnd) != 2 {
		err = fmt.Errorf("invalid range header, expected \"start-end\", got \"%s\"", val)
		return
	}

	start, err := strconv.ParseInt(startEnd[0], 10, 64)
	if err != nil {
		return
	}

	end, err := strconv.ParseInt(startEnd[1], 10, 64)
	if err != nil {
		return
	}

	r = &httpRange{start, end + 1}
	return
}

func (r *httpRange) String() string {
	return fmt.Sprintf("bytes=%d-%d", r.start, r.end-1)
}
