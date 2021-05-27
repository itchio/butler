//@ts-check
"use strict";

const { $, $$ } = require("@itchio/bob");

async function main() {
  $(`go version`);
  let gopath = $$(`go env GOPATH`);
  process.env.PATH += `:${gopath}/bin`;

  $(`go get -v -x ./butlerd/generous`);
  $(`generous godocs`);

  if (process.env.CI_BUILD_REF_NAME) {
    $(`gsutil -m cp -r -a public-read ./butlerd/generous/docs/* gs://docs.itch.ovh/butlerd/${process.env.CI_BUILD_REF_NAME}/`);
  } else {
    console.warn("Skipping uploading generous docs, no CI_BUILD_REF_NAME environment variable set")
  }
}

main();
