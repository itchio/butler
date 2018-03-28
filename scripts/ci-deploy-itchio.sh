#!/bin/bash -xe

if [ -n "${CI_COMMIT_TAG}" ]; then
  # pushing a stable version
  export CHANNEL_SUFFIX=""
  export USER_VERSION=`echo ${CI_COMMIT_TAG} | tr -d "v"` # v9.0.0 => 9.0.0
elif [ "master" == "${CI_COMMIT_REF_NAME}" ]; then
  # pushing head
  export CHANNEL_SUFFIX="-head"
  export USER_VERSION="${CI_COMMIT_SHA}"
else
  # pushing a branch that isn't master
  echo "Not pushing non-master branch ${CI_COMMIT_REF_NAME}"
  exit 0
fi

# upload to itch.io
UPLOADER_VERSION=`curl https://dl.itch.ovh/butler/linux-amd64/LATEST`
mkdir -p tools/
curl -sLo ./tools/butler "https://dl.itch.ovh/butler/linux-amd64/${UPLOADER_VERSION}/butler"
chmod +x ./tools/butler
export PATH=$PWD/tools:$PATH
butler -V

pushd broth
for i in *; do
    CHANNEL_NAME="${i}${CHANNEL_SUFFIX}"
    butler push --userversion "${USER_VERSION}" ./$i "fasterthanlime/butler:${CHANNEL_NAME}"
done
popd
