#!/bin/sh -xe

go version

export CURRENT_BUILD_PATH=$(pwd)
export GOPATH=$CURRENT_BUILD_PATH
export IMPORT_PATH=github.com/itchio/butler

mkdir -p $PKG
rsync -az . $PKG
go get -v -x -d -t $PKG
go test -v -x $PKG
