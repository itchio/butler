
# Offline diffing & patching

The `butler push` commands runs a few operations in parallel, but its basic
primitives are available as separate commands that do not require network
connectivity.

---

`butler diff` will compute the differences between two given folders or
archives. It will generate both a patch file (`patch.pwr`) and a signature file
(`patch.pwr.sig`)

You can use `/dev/null` in place of the old archive to produce a patch
against an empty container. This works everywhere and does not require
the special file `/dev/null` to actually exist or make sense in your
operating system.

---

`butler verify` will read hashes from a signature file and compare them
with the contents of a folder. It will exit with a non-zero code if they
don't match.

This can be used to verify that an installation of a game wasn't corrupted.

---

`butler apply` will use a patch file to transform an old version into
a new version. It can either rebuild the new version in a different directory
or patch the old version in place, with the `--inplace` option.

A signature can be given to the `apply` command via the `--signature file.sig` option,
to verify that the patching was successful. When no signature is given, butler
assumes that the folder being patched is a non-corrupted instance of the older version.

*In the following paragraph, 'added' files are files present in the newer version
but not the older version, 'removed' files are files present in the older version,
but not the newer, and 'touched' files are files present in both, but with a
different content.*

When working in-place (input directory == output directory + `--inplace`):

  * butler creates a temporary directory
  * rebuilds the newer version of 'touched' files in it
  * compares rebuilt files against signature, if any
    * if there's a hash mismatch, butler knows the patch will not
    apply cleanly and exits with a non-zero code. There is no override for this.
  * deleted 'removed' files from the output directory
  * moves all files from the temporary directory into
  the output directory, overwriting their previous version

This means, when working in-place, butler:

  * will not produce corrupted files
  * doesn't check the signature of untouched files
    * this allows for game updates without breaking mods

---

`butler sign` will generate a signature file, in the same format as the
`butler diff` command, and suitable to be used by the `butler verify` command.

This could be used in a scenario where patching is irrelevant, but integrity
checking is important.

---

`butler file` will display whether a file is a patch file, a signature file,
or another type of file, along with some general informations about the file.

`butler ls` will display the list of files contained in a patch file or
the list of files that can be checked via a signature file.

## Using butler programmatically

butler's output tries really hard to be readable by humans, but on occasion,
will lend itself to being parsed by other tools.

To enable JSON output mode, use the `-j` (or `--json`) flag. In JSON mode,
each line butler outputs is a valid JSON object of the following form:

  * `{type: "log", "message": "Doing something", level: "info"}`
  * `{type: "progress", "percentage": "80"}`

This is, notably, how [the itch app](https://itch.io/app) uses butler.

