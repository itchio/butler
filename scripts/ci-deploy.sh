#!/bin/sh -xe

# upload all artifacts from a single worker
gsutil -m cp -r -a public-read binaries/* gs://dl.itch.ovh/butler/
