// @ts-check
"use strict";

const { $, $$, header, chalk } = require("@itchio/bob");

function main() {
  let lines = $$(`go list -m -f '{{ .Replace }}' all`)
    .split("\n")
    .map(x => x.trim())
    .filter(x => x !== "")
    .filter(x => x !== "<nil>");
  if (lines.length > 0) {
    header(`Precondition failed`);
    console.log(`Found replace directives in ${chalk.magenta("go.mod")}`);
    for (let line of lines) {
      console.log(` - ${chalk.magenta(line)}`);
    }
    console.log(``);
    console.log(`Refusing to even run tests, bye`);
    process.exit(1);
  }

  header(`Running tests...`);
  $(`go test -v -race -cover -coverprofile=coverage.txt ./...`);

  if (!process.env.SKIP_CODECOV) {
    header(`Uploading coverage information...`);
    $(`curl -s https://codecov.io/bash | bash`);
  }
}

main();
