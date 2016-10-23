#!/bin/sh -xe

go version

rm -rf src pkg bin

export CURRENT_BUILD_PATH=$(pwd)
export GOPATH=$CURRENT_BUILD_PATH
export PKG=github.com/itchio/wharf
export PATH=$PATH:$GOPATH/bin

mkdir -p src/$PKG
rsync -a --exclude 'src' . src/$PKG

export PKGS="$(go list -f '{{if not (eq .Name "cbrotli")}}{{.ImportPath}}{{end}}' $PKG/... | paste -s -d ',' -)"

go get -v -d -t $PKG/...

go list -f '{{if gt (len .TestGoFiles) 0}}"go test -race -covermode count -coverprofile {{.Name}}.coverprofile -coverpkg $PKGS {{.ImportPath}}"{{end}}' $PKG/... | xargs -I {} bash -c {}

go get -v github.com/wadey/gocovmerge

gocovmerge `ls *.coverprofile` > coverage.txt
go tool cover -func=coverage.txt

bash <(curl -s https://codecov.io/bash)
