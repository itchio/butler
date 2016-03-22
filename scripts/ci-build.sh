#!/bin/sh -xe

echo "Should build for $CI_OS-$CI_ARCH"

if [ "$CI_OS" = "windows" ]; then
  echo "$CI_OS-$CI_ARCH" > butler.exe
else
  echo "$CI_OS-$CI_ARCH" > butler
fi
