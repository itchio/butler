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

  if (process.env.GITHUB_REF_TYPE === "tag") {
    // pushing a stable version
    channelSuffix = "";
    // v9.0.0 => 9.0.0
    userVersion = process.env.GITHUB_REF_NAME.replace(/^v/, "");
  } else if (process.env.GITHUB_REF_NAME == "master") {
    // pushing head
    channelSuffix = "-head";
    userVersion = process.env.GITHUB_SHA || "";
  } else {
    // pushing a branch that isn't master
    console.log(
      `Not pushing non-master branch ${process.env.GITHUB_REF_NAME}`
    );
    return;
  }

  // upload to itch.io
  let toolsDir = join(process.cwd(), `tools`);
  mkdirSync(toolsDir, { recursive: true });
  await cd(toolsDir, async () => {
    // this is hard coded old vesrion that is known to work, so that we can use butler to upload butler
    let ref = `898b30bb2cadaaf561d91c4f7784aa1927c5cfb2`;
    let url = `https://broth.itch.zone/butler/linux-amd64-head/${ref}/.zip`;
    $(`curl -sLo butler.zip "${url}"`);
    $(`unzip butler.zip`);
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
        `itchio/butler:${channelName}`,
      ];
      $(`${toolsDir}/butler ${args.join(" ")}`);
    }
  });
}

main(process.argv.slice(2));
