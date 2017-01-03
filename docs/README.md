
# butler

butler is a small command-line tool that lets you:

  * [Upload builds](pushing.md) of your games quickly & reliably to [itch.io](https://itch.io)
  * Generate patches and apply them [offline](offline.md)
  * Run a few other [utility commands](utilities.md)

When someone refers to **the itch.io command-line tools**, they're talking about butler.

butler is easily integrated into an automated build/deploy pipeline. Like most
of the itch.io delivery infrastructure, it is open-source (MIT licensed):

  * <https://github.com/itchio/butler>

It is also a good showcase of the *Wharf open specification*, using only
openly documented algorithms and file formats, encouraging the development
of compatible tools:

  * <https://docs.itch.ovh/wharf>

And is used by [the itch app](https://itch.io/app) for various operations.

Keep reading to learn how to use it! First you'll want to [install butler](install.md).

