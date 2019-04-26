#!/bin/sh -xe

go version

export PATH="$PATH:$(go env GOPATH)/bin"

go test -v -race -cover -coverprofile=coverage.txt ./...

curl -s https://codecov.io/bash | bash
