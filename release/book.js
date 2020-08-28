//@ts-check
"use strict";

const { cd, $, $$ } = require("@itchio/bob");

async function main() {
  $(`npm version`);
  $(`npm install -g gitbook-cli`);

  if (process.env.CI) {
    // cf. https://github.com/GitbookIO/gitbook-cli/issues/110#issuecomment-669640662
    let npm_prefix = $$(`npm config get prefix`);
    await cd(
      `${npm_prefix}/lib/node_modules/gitbook-cli/node_modules/npm/node_modules`,
      async () => {
        $(`npm install graceful-fs@latest --save`);
      }
    );

    await cd(
      `${process.env.HOME}/.gitbook/versions/3.2.3/node_modules/npm`,
      async () => {
        $(`npm install graceful-fs@latest --save`);
      }
    );
  }

  await cd("docs", async () => {
    $(`npm install`);
    $(`gitbook install`);
    $(`gitbook build`);
  });

  $(
    `gsutil -m cp -r -a public-read docs/_book/* gs://docs.itch.ovh/butler/${process.env.CI_BUILD_REF_NAME}/`
  );
}

main();
