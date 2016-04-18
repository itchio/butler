
# Utility commands

Some commands in butler aren't related to diffing, patching, or pushing builds.
They're just generally useful commands, used for example, by [the itch app](https://itch.io/app).

`butler dl` will download a file from a given URL and save it somewhere on
disk.  It supports resuming uploads (if the HTTP server supports byte range
requests, otherwise it'll start over), will check the file's size when done,
and if the server responds with Google Cloud Storage's private headers, it will
check the crc32c[^1] hash of the downloaded file.

[^1]: [CRC-32](https://en.wikipedia.org/wiki/Cyclic_redundancy_check) with the Castagnoli polynomial.

`butler wipe` will completely remove a file or a folder and its content,
recursively.

`butler ditto` will copy a folder to another place on your disk, preserving
permissions (with a mask) and symbolic links (as opposed to cp, which copies
the actual files the symlinks point to).

`butler untar` will extract a .tar archive, preserving permissions (with a mask)
and symlinks. It will work with .tar archive missing directory entries by
just creating them.

