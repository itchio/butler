#!/bin/sh -xe

echo "Building for $CI_OS-$CI_ARCH"

go version

export CGO_ENABLED=1

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

export CI_LDFLAGS="-X buildinfo.Version=$CI_VERSION -X buildinfo.BuiltAt=$CI_BUILT_AT -X buildinfo.Commit=$CI_BUILD_REF -w -s"

TARGET=butler
if [ "$CI_OS" = "windows" ]; then
  TARGET=$TARGET.exe
else
  export PATH=$PATH:/usr/local/go/bin
fi

# compile manifest before rsync'ing
if [ "$CI_OS" = "windows" ]; then
    ${WINDRES} -o butler.syso butler.rc
fi

# compile
GOOS=$CI_OS GOARCH=$CI_ARCH go build -ldflags "$CI_LDFLAGS" .

# check glibc version (linux)
if [ "$CI_OS" = "linux" ]; then
  ./butler diag --no-net --glibc
fi

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
