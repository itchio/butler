#!/bin/sh -e

go version
export PATH="$PATH:$(go env GOPATH)/bin"

set +e
replaced=$(go list -m -f '{{ .Replace }}' all | grep -v -F "<nil>")
set -e
if [[ -n $replaced ]]; then
    echo ""
    echo "======================================================="
    echo "=           Error: precondition failed                ="
    echo "======================================================="
    echo "go.mod: found replace directives:"
    echo "$replaced"
    echo "======================================================="
    echo ""
    echo "Refusing to even run tests, bye"
    exit 1
fi

set -x

go test -v -race -cover -coverprofile=coverage.txt ./...

curl -s https://codecov.io/bash | bash
