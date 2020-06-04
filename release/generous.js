//@ts-check
"use strict";

const { $, $$ } = require("@itchio/bob");

async function main() {
  $(`go version`);
  let gopath = $$(`go env GOPATH`);
  process.env.PATH += `:${gopath}/bin`;

  $(`go get -v -x ./butlerd/generous`);
  $(`generous godocs`);

  $(
    `gsutil -m cp -r -a public-read ./butlerd/generous/docs/* gs://docs.itch.ovh/butlerd/${process.env.CI_BUILD_REF_NAME}/`,
  );
}

main();
