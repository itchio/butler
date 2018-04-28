#!/bin/sh -xe

npm version
npm install -g gitbook-cli

(cd docs && npm install && gitbook build)

gsutil -m cp -r -a public-read docs/_book/* gs://docs.itch.ovh/butler/$CI_BUILD_REF_NAME/
