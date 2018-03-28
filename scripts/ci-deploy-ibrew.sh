#!/bin/bash -xe

# upload to legacy ibrew repo
gsutil -m cp -r -a public-read ibrew/* gs://dl.itch.ovh/butler/
