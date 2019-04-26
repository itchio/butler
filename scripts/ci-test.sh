#!/bin/sh -xe

go version

export PKG=github.com/itchio/butler
export PATH="$PATH:$(go env GOPATH)/bin"
export PATH=$PATH:$GOPATH/bin

go get -v -d -t ./...
go test -v -race -cover -coverprofile=coverage.txt ./...

curl -s https://codecov.io/bash | bash
