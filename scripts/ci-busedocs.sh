#!/bin/sh -xe

go version

export CURRENT_BUILD_PATH=$(pwd)
export GOPATH=$CURRENT_BUILD_PATH
export PATH="$PATH:$GOPATH/bin"

export PKG=github.com/itchio/butler

mkdir -p src/$PKG

# rsync will complain about vanishing files sometimes, who knows where they come from
rsync -a --exclude 'src' . src/$PKG || echo "rsync complained (code $?)"

cd src/$PKG/buse/busegen

go get -v -x
busegen

gsutil cp -r -a public-read docs/* gs://docs.itch.ovh/buse/$CI_BUILD_REF_NAME/

