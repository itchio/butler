PKG := github.com/itchio/butler
BUILDINFO_PKG := $(PKG)/buildinfo

VERSION ?= head
COMMIT := $(shell git rev-parse HEAD)
BUILT_AT := $(shell date +%s)

LDFLAGS := -X $(BUILDINFO_PKG).Version=$(VERSION) \
           -X $(BUILDINFO_PKG).Commit=$(COMMIT) \
           -X $(BUILDINFO_PKG).BuiltAt=$(BUILT_AT) \
           -w -s

.PHONY: build install clean

build:
	CGO_ENABLED=1 go build -ldflags "$(LDFLAGS)" -o butler .

install:
	CGO_ENABLED=1 go install -ldflags "$(LDFLAGS)" .

clean:
	rm -f butler
