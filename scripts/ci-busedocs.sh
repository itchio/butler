#!/bin/sh -xe

go version

export CURRENT_BUILD_PATH=$(pwd)
export GOPATH=$CURRENT_BUILD_PATH
export PATH="$PATH:$GOPATH/bin"

export PKG=github.com/itchio/butler

mkdir -p src/$PKG

# rsync will complain about vanishing files sometimes, who knows where they come from
rsync -a --exclude 'src' . src/$PKG || echo "rsync complained (code $?)"

go get -v -x $PKG/buse/busegen
busegen godocs

gsutil cp -r -a public-read src/$PKG/buse/busegen/docs/* gs://docs.itch.ovh/buse/$CI_BUILD_REF_NAME/

