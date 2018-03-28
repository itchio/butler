#!/bin/bash -xe

export CHANNEL_SUFFIX="-head"
export CI_VERSION="head"
if [ -n "$CI_BUILD_TAG" ]; then
  # pushing a stable version
  export CHANNEL_SUFFIX=""
  export CI_VERSION="$CI_BUILD_TAG"
elif [ "master" != "$CI_BUILD_REF_NAME" ]; then
  # pushing a branch that isn't master
  echo "Not pushing non-master branch $CI_BUILD_REF_NAME"
  exit 0
fi

# upload to itch.io
UPLOADER_VERSION=`curl https://dl.itch.ovh/butler/linux-amd64/LATEST`
mkdir -p tools/
curl -sLo ./tools/butler "https://dl.itch.ovh/butler/linux-amd64/${UPLOADER_VERSION}/butler"
chmod +x ./tools/butler
export PATH=$PWD/tools:$PATH
butler -V

USER_VERSION=`echo ${CI_BUILD_TAG} | tr -d "v"`

pushd broth
for i in *; do
    CHANNEL_NAME="${i}-${CHANNEL_SUFFIX}"
    butler push --userversion "${USER_VERSION}" ./$i "fasterthanlime/butler:${CHANNEL_NAME}"
done
popd
