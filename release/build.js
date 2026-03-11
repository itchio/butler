//@ts-check
"use strict";

const {
  $,
  $$,
  chalk,
  debug,
  header,
  setenv,
  detectOS,
  setVerbose,
} = require("@itchio/bob");

const { resolve } = require("path");
const { mkdirSync } = require("fs");

const DEFAULT_ARCH = "x86_64";

/**
 * @typedef OsInfo
 * @type {{
 *   architectures: {
 *     [key: string]: {
 *       prependPath?: string,
 *     }
 *   }
 * }}
 */

/**
 * @type {{[name: string]: OsInfo}}
 */
const OS_INFOS = {
  windows: {
    architectures: {
      i686: {
        prependPath: "/mingw32/bin",
      },
      x86_64: {
        prependPath: "/mingw64/bin",
      },
    },
  },
  linux: {
    architectures: {
      x86_64: {},
      arm64: {},
    },
  },
  darwin: {
    architectures: {
      x86_64: {},
      arm64: {},
    },
  },
};

/**
 * @param {string[]} args
 */
async function main(args) {
  /**
   * @type {{
   *   os: "windows" | "linux" | "darwin",
   *   arch: "i686" | "x86_64",
   *   userSpecifiedOS?: boolean,
   *   userSpecifiedArch?: boolean,
   * }}
   */
  let opts = {
    os: detectOS(),
    arch: DEFAULT_ARCH,
    skipSigning: false,
  };

  for (let i = 0; i < args.length; i++) {
    let arg = args[i];

    let matches = /^--(.*)$/.exec(arg);
    if (matches) {
      let k = matches[1];
      if (k == "verbose") {
        setVerbose(true);
        continue;
      }

      if (k === "skip-signing") {
        opts.skipSigning = true;
        continue;
      }

      if (k === "os" || k === "arch") {
        i++;
        let v = args[i];

        if (k === "os") {
          if (v === "linux" || v === "windows" || v === "darwin") {
            opts.os = v;
            opts.userSpecifiedOS = true;
          } else {
            throw new Error(`Unsupported os ${chalk.yellow(v)}`);
          }
        } else if (k === "arch") {
          if (v === "i686" || v === "x86_64" || v === "arm64") {
            opts.arch = v;
            opts.userSpecifiedArch = true;
          } else {
            throw new Error(`Unsupported arch ${chalk.yellow(v)}`);
          }
        }
      } else {
        throw new Error(`Unsupported long option ${chalk.yellow(arg)}`);
      }
    }
  }

  if (opts.userSpecifiedOS) {
    console.log(`Using user-specified OS ${chalk.yellow(opts.os)}`);
  } else {
    console.log(
      `Using detected OS ${chalk.yellow(opts.os)} (use --os to override)`,
    );
  }

  if (opts.userSpecifiedArch) {
    console.log(`Using user-specified arch ${chalk.yellow(opts.arch)}`);
  } else {
    console.log(
      `Using detected arch ${chalk.yellow(opts.arch)} (use --arch to override)`,
    );
  }

  let osInfo = OS_INFOS[opts.os];
  debug({ osInfo });
  if (!osInfo) {
    throw new Error(`Unsupported OS ${chalk.yellow(opts.os)}`);
  }

  let archInfo = osInfo.architectures[opts.arch];
  debug({ archInfo });
  if (!archInfo) {
    throw new Error(`Unsupported arch '${opts.arch}' for os '${opts.os}'`);
  }

  if (archInfo.prependPath) {
    if (opts.os === "windows") {
      let prependPath = $$(`cygpath -w ${archInfo.prependPath}`).trim();
      console.log(
        `Prepending ${chalk.yellow(archInfo.prependPath)} (aka ${chalk.yellow(
          prependPath,
        )}) to $PATH`,
      );
      process.env.PATH = `${prependPath};${process.env.PATH}`;
    } else {
      console.log(`Prepending ${chalk.yellow(archInfo.prependPath)} to $PATH`);
      process.env.PATH = `${archInfo.prependPath}:${process.env.PATH}`;
    }
  }

  header("Showing tool versions");
  $(`node --version`);
  $(`go version`);

  let version = "head";
  let builtAt = $$(`date +%s`);
  if (process.env.GITHUB_REF_TYPE === "tag") {
    version = process.env.GITHUB_REF_NAME;
  } else if (
    process.env.GITHUB_REF_NAME &&
    process.env.GITHUB_REF_NAME !== "master"
  ) {
    version = process.env.GITHUB_REF_NAME;
  }

  let bi = `github.com/itchio/butler/buildinfo`;

  let ldflags = [
    `-X ${bi}.Version=${version}`,
    `-X ${bi}.BuiltAt=${builtAt}`,
    `-X ${bi}.Commit=${process.env.GITHUB_SHA || ""}`,
    "-w",
    "-s",
  ].join(" ");

  if (opts.os === "darwin") {
    // Add -arch flag for cross-compilation on ARM Mac
    let archFlag = opts.arch === "x86_64" ? "-arch x86_64" : "";
    setenv(`CGO_CFLAGS`, `-mmacosx-version-min=10.10 ${archFlag}`.trim());
    setenv(`CGO_LDFLAGS`, `-mmacosx-version-min=10.10 ${archFlag}`.trim());
  }

  let target = `butler`;
  if (opts.os === "windows") {
    target = `${target}.exe`;
  }

  if (opts.os === "windows") {
    console.log(`Compiling Windows manifest`);
    $(`windres -o butler.syso butler.rc`);
  }

  console.log(`Compiling binary`);
  let goArch = archToGoArch(opts.arch);
  setenv(`GOOS`, opts.os);
  setenv(`GOARCH`, goArch);
  setenv(`CGO_ENABLED`, `1`);
  $(`go build -ldflags "${ldflags}"`);

  if (opts.os === "linux") {
    console.log(`Checking minimum glibc version`);
    try {
      $(`./butler diag --no-net --glibc`);
    } catch (e) {
      if (process.env.CI) {
        throw e;
      } else {
        console.log(`Ignoring butler diag failure becaus we're not on CI`);
      }
    }
  }

  if (opts.os === "windows" && !opts.skipSigning) {
    let signArgs = [
      "sign", // verb
      "//v", // verbose
      "//s MY", // store
      `//n "itch corp"`, // name
      `//fd sha256`, // file digest algo (default is SHA-1)
      `//tr http://timestamp.comodoca.com/?td=sha256`, // URL of RFC 3161 timestamp server
      `//td sha256`, // timestamp digest algo
      `//a`, // choose best cert
      target,
    ];

    $(`tools/signtool.exe ${signArgs.join(" ")}`);
  }

  if (opts.os === "darwin" && !opts.skipSigning) {
    let signKey = "Developer ID Application: itch corp. (AK2D34UDP2)";
    $(`codesign --deep --force --verbose --sign "${signKey}" "${target}"`);
    $(`codesign --verify -vvvv "${target}"`);
    // We don't use spctl -a on purpose, see
    // https://stackoverflow.com/questions/39811791/mac-os-gatekeeper-blocking-signed-command-line-tool
  }

  header("Packaging...");
  let artifactDir = `./artifacts/${opts.os}-${goArch}`;
  mkdirSync(artifactDir, { recursive: true });

  let fullTarget = `${artifactDir}/${target}`;
  $(`mv ${target} ${fullTarget}`);
  $(`file ${fullTarget}`);
  $(`${fullTarget} -V`);
  $(`${fullTarget} fetch-7z-libs`);

  let fullButlerPath = resolve(process.cwd(), fullTarget);
  $(`go test -v ./butlerd/integrate --butlerPath='${fullButlerPath}'`);
}

/**
 * @param {"i686" | "x86_64"} arch
 * @returns {"386" | "amd64"}
 */
function archToGoArch(arch) {
  switch (arch) {
    case "i686":
      return "386";
    case "x86_64":
      return "amd64";
    case "arm64":
      return "arm64";
    default:
      throw new Error(`unsupported arch: ${chalk.yellow(arch)}`);
  }
}

main(process.argv.slice(2));
