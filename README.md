# butler

[![Build Status](https://git.itch.ovh/itchio/butler/badges/master/build.svg)](https://git.itch.ovh/itchio/butler/builds)
[![codecov](https://codecov.io/gh/itchio/butler/branch/master/graph/badge.svg)](https://codecov.io/gh/itchio/butler)
[![Go Report Card](https://goreportcard.com/badge/github.com/itchio/butler)](https://goreportcard.com/report/github.com/itchio/butler)
![MIT licensed](https://img.shields.io/badge/license-MIT-blue.svg)

butler is *the itch.io command-line tools* - all by itself.

**It is used by:**

  * Content creators on [itch.io](https://itch.io) to push builds quickly & reliably
  * [the itch app](https://github.com/itchio/itch) for some network, filesystem and patching operations

## Documentation

Documentation for butler is available as a Gitbook:

  * :memo: <https://itch.io/docs/butler>

Questions about butler are welcome on its [Issue tracker](https://github.com/itchio/butler/issues),
or, if the matter is private, [itch.io support](https://itch.io/support).

## Integrations

The following projects integrate butler as part of their workflow:

  * [itchy-electron](https://github.com/erbridge/itchy-electron) lets you package your Electron games for itch.io and upload them there
  * [gradle-butler-plugin](https://github.com/mini2Dx/gradle-butler-plugin) is a Gradle plugin for automatically installing, updating, and running butler as part of your build.

## Authors

butler was mostly written by [Amos Wenger](https://github.com/fasterthanlime), but wouldn't have
been possible without the work of many before him.

Amos would like to thank in particular Leaf Corcoran, Jesús Higueras and Tomáš Duda.

## License

butler is released under the MIT License. See the [LICENSE](LICENSE) file for details.

## Additional licenses

While butler built from source is fully MIT-licensed, some components it can use at runtime
(if present) have other licenses:

  * The 7-zip decompression engine (the `github.com/itchio/butler/archive/szextractor` package) opens
  dynamic libraries for [libc7zip][], and [7-zip][], which have components licensed under the LGPL 2.1 license
  and the MPL 2.0 license, along with specific terms for the RAR extraction code.

[libc7zip]: https://github.com/itchio/libc7zip
[7-zip]: http://7-zip.org/faq.html
[7-zip FAQ]: http://7-zip.org/faq.html
