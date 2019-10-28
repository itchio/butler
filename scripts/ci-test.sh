#!/bin/sh -xe

go version

export PATH="$PATH:$(go env GOPATH)/bin"

grep "replace" go.mod
if [[ $? == 0 ]]; then
    set +x
    echo "======================================================="
    echo "=                      NOPE                           ="
    echo "======================================================="
    echo "= go.mod: should not have any replace directives      ="
    echo "=                                                     ="
    echo "= signed: your friendly neighborhood automated check  ="
    echo "======================================================="
    exit 1
fi

go test -v -race -cover -coverprofile=coverage.txt ./...

curl -s https://codecov.io/bash | bash
