#!/bin/sh -xe

echo "Should build for $CI_OS-$CI_ARCH"

go version

export CURRENT_BUILD_PATH=$(pwd)
export GOPATH=$CURRENT_BUILD_PATH
export PATH="$PATH:$GOPATH/bin"
export CGO_ENABLED=1

go get github.com/mitchellh/gox

export CI_BUILD_TAG

if [ "$CI_OS" = "windows" ]; then
  if [ "$CI_ARCH" = "386" ]; then
    TRIPLET="i686-w64-mingw32-"
  else
    TRIPLET="x86_64-w64-mingw32-"
  fi
else
  TRIPLET=""
fi

export CC="${TRIPLET}gcc"
export CXX="${TRIPLET}g++"

export CI_VERSION="head"
if [ "$CI_DEPLOY" = "1" ]; then
  export CI_VERSION=$CI_BUILD_TAG
fi
export CI_LDFLAGS="-X main.version=$CI_VERSION"

TARGET=butler
if [ "$CI_OS" = "windows" ]; then
  TARGET=$TARGET.exe
else
  export PATH=$PATH:/usr/local/go/bin
fi

export PKG=github.com/itchio/butler

mkdir -p $PKG
rsync -az . $PKG
GOOS=$CI_OS GOARCH=$CI_ARCH go get -v -d -t $PKG
gox -osarch = "$CI_OS/$CI_ARCH" -ldflags "$CI_LDFLAGS" -cgo -output="butler" $PKG

file $TARGET
./$TARGET -v

7za a butler.7z $TARGET
