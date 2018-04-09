#!/bin/sh -xe

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
go test -v github.com/itchio/butler/butlerd/integrate --butlerPath=$PWD/built/$TARGET

(cd built/ && 7za a ../butler.7z *)
(cd built/ && 7za a ../butler.zip *)
(cd built/ && 7za a ../butler.gz $TARGET)

# set up a file hierarchy that ibrew can consume, ie:
#
# - windows-amd64
#   - LATEST
#   - v0.11.0
#     - butler.7z
#     - butler.gz
#     - butler.exe
#     - SHA1SUMS
#     - SHA256SUMS

IBREW_DIR="ibrew/$CI_OS-$CI_ARCH"
mkdir -p $IBREW_DIR/$CI_VERSION
mv butler.7z $IBREW_DIR/$CI_VERSION
mv butler.gz $IBREW_DIR/$CI_VERSION
mv butler.zip $IBREW_DIR/$CI_VERSION
cp built/* $IBREW_DIR/$CI_VERSION

(cd $IBREW_DIR/$CI_VERSION && sha1sum * > SHA1SUMS && sha256sum * > SHA256SUMS)

if [ -n "$CI_BUILD_TAG" ]; then
  echo $CI_VERSION > $IBREW_DIR/LATEST
fi

# set up a file hierarchy we can push with butler, ie.
#
# - windows-amd64
#   - butler.exe
#   - c7zip.dll
#   - 7z.dll

BROTH_DIR="broth/$CI_OS-$CI_ARCH"
mkdir -p $BROTH_DIR
cp built/* $BROTH_DIR/
