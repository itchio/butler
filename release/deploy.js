//@ts-check
"use strict";

const { mkdirSync, readdirSync } = require("fs");
const { join } = require("path");
const { $, cd } = require("@itchio/bob");

/**
 * @param {string[]} _args
 */
async function main(_args) {
  /** @type {string} */
  let channelSuffix;
  /** @type {string} */
  let userVersion;

  if (process.env.CI_COMMIT_TAG) {
    // pushing a stable version
    channelSuffix = "";
    // v9.0.0 => 9.0.0
    userVersion = process.env.CI_COMMIT_TAG.replace(/^v/, "");
  } else if (process.env.CI_COMMIT_REF_NAME == "master") {
    // pushing head
    channelSuffix = "-head";
    userVersion = process.env.CI_COMMIT_SHA || "";
  } else {
    // pushing a branch that isn't master
    console.log(
      `Not pushing non-master branch ${process.env.CI_COMMIT_REF_NAME}`
    );
    return;
  }

  // upload to itch.io
  let toolsDir = join(process.cwd(), `tools`);
  mkdirSync(toolsDir, { recursive: true });
  await cd(toolsDir, async () => {
    let ref = `65bf13509d214308111a7b9c0f227099034536c7`;
    let url = `https://broth.itch.ovh/butler/linux-amd64-head/${ref}/.zip`;
    $(`curl -sLo butler.zip "${url}"`);
  });

  $(`${toolsDir}/butler -V`);

  await cd("artifacts", async () => {
    let variants = readdirSync(".");
    for (let variant of variants) {
      let channelName = `${variant}${channelSuffix}`;
      let args = [
        `push`,
        `--userversion "${userVersion}"`,
        `./${variant}`,
        `fasterthanlime/butler:${channelName}`,
      ];
      $(`${toolsDir}/butler ${args.join(" ")}`);
    }
  });
}

main(process.argv.slice(2));
