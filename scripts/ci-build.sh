#!/bin/sh -xe

echo "Building for $CI_OS-$CI_ARCH"

go version

export CURRENT_BUILD_PATH=$(pwd)
export GOPATH=$CURRENT_BUILD_PATH
export PATH="$PATH:$GOPATH/bin"
export CGO_ENABLED=1

# set up go cross-compile
go get github.com/mitchellh/gox

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
export WINDRES="${TRIPLET}windres"

export CI_VERSION="head"
export CI_BUILT_AT="$(date +%s)"
if [ -n "$CI_BUILD_TAG" ]; then
  export CI_VERSION="$CI_BUILD_TAG"
elif [ "master" != "$CI_BUILD_REF_NAME" ]; then
  export CI_VERSION="$CI_BUILD_REF_NAME"
fi

export CI_LDFLAGS="-X main.version=$CI_VERSION -X main.builtAt=$CI_BUILT_AT -X main.commit=$CI_BUILD_REF"

TARGET=butler
if [ "$CI_OS" = "windows" ]; then
  TARGET=$TARGET.exe
else
  export PATH=$PATH:/usr/local/go/bin
fi

export PKG=github.com/itchio/butler

mkdir -p src/$PKG

# rsync will complain about vanishing files sometimes, who knows where they come from
rsync -a --exclude 'src' . src/$PKG || echo "rsync complained (code $?)"

# grab deps
GOOS=$CI_OS GOARCH=$CI_ARCH go get -v -d -t $PKG

# compile maniest
if [ "$CI_OS" = "windows" ]; then
    ${WINDRES} -o butler.syso butler.rc
fi

# compile
gox -osarch "$CI_OS/$CI_ARCH" -ldflags "$CI_LDFLAGS" -cgo -output="butler" $PKG

# sign (win)
if [ "$CI_OS" = "windows" ]; then
  WIN_SIGN_KEY="itch corp."
  WIN_SIGN_URL="http://timestamp.comodoca.com"

  signtool.exe sign //v //s MY //n "$WIN_SIGN_KEY" //fd sha256 //tr "$WIN_SIGN_URL" //td sha256 $TARGET
fi

# sign (osx)
# restore when https://github.com/golang/go/issues/11887 is fixed
if [ "$CI_OS-disabled" = "darwin" ]; then
  OSX_SIGN_KEY="Developer ID Application: Amos Wenger (B2N6FSRTPV)"

  codesign --deep --force --verbose --sign "$OSX_SIGN_KEY" $TARGET
  codesign --verify -vvvv $TARGET
  spctl -a -vvvv $TARGET
fi

# verify
file $TARGET
./$TARGET -V

7za a butler.7z $TARGET
7za a butler.gz $TARGET

# set up a file hierarchy that ibrew can consume, ie:
#
# - dl.itch.ovh
#   - butler
#     - windows-amd64
#       - LATEST
#       - v0.11.0
#         - butler.7z
#         - butler.gz
#         - butler.exe
#         - SHA1SUMS

BINARIES_DIR="binaries/$CI_OS-$CI_ARCH"
mkdir -p $BINARIES_DIR/$CI_VERSION
mv butler.7z $BINARIES_DIR/$CI_VERSION
mv butler.gz $BINARIES_DIR/$CI_VERSION
mv $TARGET $BINARIES_DIR/$CI_VERSION

(cd $BINARIES_DIR/$CI_VERSION && sha1sum * > SHA1SUMS && sha256sum * > SHA256SUMS)

if [ -n "$CI_BUILD_TAG" ]; then
  echo $CI_VERSION > $BINARIES_DIR/LATEST
fi

