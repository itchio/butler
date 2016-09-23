#!/bin/sh

go version

export CURRENT_BUILD_PATH=$(pwd)
export GOPATH=$CURRENT_BUILD_PATH
export PKG=github.com/itchio/wharf

mkdir -p $PKG
rsync -a . $PKG
go get -v -d -t $PKG/...
go test -v $PKG/...
