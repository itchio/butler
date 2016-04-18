# butler

![MIT licensed](https://img.shields.io/badge/license-MIT-blue.svg)

butler is a command-line tool written in Go.

itch (the itch.io app) uses it for some network, filesystem and patching operations:

  * <https://github.com/itchio/itch>

## Usage & rationale

butler's diffing & patching hasn't been officially released yet, but
Amos published a small write-up on the [itch community](https://itch.io/post/16715).

Early testers are welcome, especially for the `sign`, `verify`, `diff`, and `apply`
commands, which are detailed within the program's inline help.

The `push` command now works, but be warned: the backend is in flux.

## License

Licensed under MIT License, see `LICENSE` for details.
