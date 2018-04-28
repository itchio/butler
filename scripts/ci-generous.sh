#!/bin/sh -xe

go version

export CURRENT_BUILD_PATH=$(pwd)
export GOPATH=$CURRENT_BUILD_PATH
export PATH="$PATH:$GOPATH/bin"

export PKG=github.com/itchio/butler

mkdir -p src/$PKG

# rsync will complain about vanishing files sometimes, who knows where they come from
rsync -a --exclude 'src' . src/$PKG || echo "rsync complained (code $?)"

go get -v -x $PKG/butlerd/generous
generous godocs

gsutil -m cp -r -a public-read src/$PKG/butlerd/generous/docs/* gs://docs.itch.ovh/butlerd/$CI_BUILD_REF_NAME/

