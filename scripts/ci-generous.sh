#!/bin/sh -xe

go version

export PATH="$PATH:$(go env GOPATH)/bin"

go get -v -x ./butlerd/generous
generous godocs

gsutil -m cp -r -a public-read ./butlerd/generous/docs/* gs://docs.itch.ovh/butlerd/$CI_BUILD_REF_NAME/

