//@ts-check
"use strict";

const { cd, $ } = require("@itchio/bob");

async function main() {
  $(`npm version`);
  $(`npm install -g gitbook-cli`);

  await cd("docs", async () => {
    $(`npm install`);
    $(`gitbook build`);
  });

  $(
    `gsutil -m cp -r -a public-read docs/_book/* gs://docs.itch.ovh/butler/${process.env.CI_BUILD_REF_NAME}/`,
  );
}

main();
