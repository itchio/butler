#!/bin/sh -xe

echo "Should build for $CI_OS-$CI_ARCH"

go version

export CURRENT_BUILD_PATH=$(pwd)
export GOPATH=$CURRENT_BUILD_PATH
export PKG=github.com/itchio/butler

mkdir -p $PKG
rsync -az . $PKG
go get -v -d -t $PKG

export GOOS=$CI_OS
export GOARCH=$CI_ARCH
go build -v -x $PKG
