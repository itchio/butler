#!/bin/sh -xe

go version

export GOPATH=$PWD/gopath
rm -rf $GOPATH

export PKG=github.com/itchio/butler
export PATH=$PATH:$GOPATH/bin

mkdir -p $GOPATH/src/$PKG
rsync -a --exclude 'gopath' . $GOPATH/src/$PKG || echo "rsync complained (code $?)"

go get -v -d -t $PKG/...
go test -v -race -cover -coverprofile=coverage.txt $PKG/...

curl -s https://codecov.io/bash | bash
