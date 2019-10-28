#!/bin/sh -xe

export CURRENT_BUILD_PATH=$(pwd)

export CI_VERSION="head"
export CI_BUILT_AT="$(date +%s)"
if [ -n "$CI_BUILD_TAG" ]; then
  export CI_VERSION="$CI_BUILD_TAG"
elif [ "master" != "$CI_BUILD_REF_NAME" ]; then
  export CI_VERSION="$CI_BUILD_REF_NAME"
fi

TARGET=butler
if [ "$CI_OS" = "windows" ]; then
  TARGET=$TARGET.exe
else
  export PATH=$PATH:/usr/local/go/bin
fi

rm -rf built
mkdir -p built
mv $TARGET built/$TARGET

# verify
file built/$TARGET
./built/$TARGET -V
./built/$TARGET fetch-7z-libs

# run integration tests
go test -v ./butlerd/integrate --butlerPath=$PWD/built/$TARGET

# set up a file hierarchy we can push with butler, ie.
#
# - windows-amd64
#   - butler.exe
#   - c7zip.dll
#   - 7z.dll

BROTH_DIR="broth/$CI_OS-$CI_ARCH"
mkdir -p $BROTH_DIR
cp built/* $BROTH_DIR/
