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
    export PATH="/mingw32/bin:$PATH"
  else
    export PATH="/mingw64/bin:$PATH"
  fi
else
  TRIPLET=""
fi

export CC="gcc"
export CXX="g++"
export WINDRES="windres"

export CI_VERSION="head"
export CI_BUILT_AT="$(date +%s)"
if [ -n "$CI_BUILD_TAG" ]; then
  export CI_VERSION="$CI_BUILD_TAG"
elif [ "master" != "$CI_BUILD_REF_NAME" ]; then
  export CI_VERSION="$CI_BUILD_REF_NAME"
fi

export CI_LDFLAGS="-X main.butlerVersion=$CI_VERSION -X main.butlerBuiltAt=$CI_BUILT_AT -X main.butlerCommit=$CI_BUILD_REF -w -s"

TARGET=butler
if [ "$CI_OS" = "windows" ]; then
  TARGET=$TARGET.exe
else
  export PATH=$PATH:/usr/local/go/bin
fi

export PKG=github.com/itchio/butler

mkdir -p src/$PKG

# compile manifest before rsync'ing
if [ "$CI_OS" = "windows" ]; then
    ${WINDRES} -o butler.syso butler.rc
fi

# rsync will complain about vanishing files sometimes, who knows where they come from
rsync -a --exclude 'src' . src/$PKG || echo "rsync complained (code $?)"

# grab deps
GOOS=$CI_OS GOARCH=$CI_ARCH go get -v -d -t $PKG

# compile
gox -osarch "$CI_OS/$CI_ARCH" -ldflags "$CI_LDFLAGS" -cgo -output="butler" $PKG

# sign (win)
if [ "$CI_OS" = "windows" ]; then
  WIN_SIGN_KEY="itch corp."
  WIN_SIGN_URL="http://timestamp.comodoca.com"

  signtool.exe sign //v //s MY //n "$WIN_SIGN_KEY" //fd sha256 //tr "$WIN_SIGN_URL" //td sha256 $TARGET
fi

# sign (osx)
if [ "$CI_OS" = "darwin" ]; then
  OSX_SIGN_KEY="Developer ID Application: Amos Wenger (B2N6FSRTPV)"

  codesign --deep --force --verbose --sign "$OSX_SIGN_KEY" $TARGET
  codesign --verify -vvvv $TARGET
  # Ignore that for now, see https://stackoverflow.com/questions/39811791/mac-os-gatekeeper-blocking-signed-command-line-tool
  # spctl -a -vvvv $TARGET
fi

# verify
rm -rf built
mkdir -p built
mv $TARGET built/$TARGET

file built/$TARGET
./built/$TARGET -V
./built/$TARGET fetch-7z-libs

go test -v github.com/itchio/butler/butlerd/butlerd_test --butlerPath=built/$TARGET

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

