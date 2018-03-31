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
export TOOLS_DIR=$PWD/tools/
mkdir -p ${TOOLS_DIR}
pushd ${TOOLS_DIR}
curl -sLo butler.zip "https://broth.itch.ovh/butler/linux-amd64-head/LATEST/.zip"
unzip butler.zip
popd

${TOOLS_DIR}/butler -V

pushd broth
for i in *; do
    CHANNEL_NAME="${i}${CHANNEL_SUFFIX}"
    ${TOOLS_DIR}/butler push --userversion "${USER_VERSION}" ./$i "fasterthanlime/butler:${CHANNEL_NAME}"
done
popd
