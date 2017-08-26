
# Single files

In order to understand why butler is reluctant to push "single files", we have to dig a little deeper.

## Portable builds

The best case scenario for butler is when you have a "portable build"
of your game, which is a folder that contains the game's executables
and its resources.

> By resources, I mean: sound effects, music, textures, levels, anything that isn't code.

[Unity](https://unity3d.com/) exports have the following structure, for example:

```
- UnityGame/
  - Game.exe
  - Data/
    - level0
    - level1
    - etc.
```

It's called a portable build because if you have that folder on your disk, you can just run the executable, and **the game will work**. No extra steps needed.

If you're already using Unity, Unreal Engine 4, Godot Engine, or any other engine that exports a similar folder structure, you can stop reading here: Good job, you did it! Just push that folder and get a nice beverage of your choice.

### Zipped portable builds

If you try to push a .zip using butler, it will treat it just like a folder.

So it will decompress it on-the-fly, then apply its own compression while uploading.

There's really no upside to pushing a .zip instead of a folder, since the result is
exactly the same. It's just a convenience if all you have is a .zip of a portable build
of your game - saves you the time to decompress it yourself.

> I lied: there is one situation where pushing a .zip would save you. If you're pushing,
for example, a macOS build of your game from a version of Windows that does not support
symbolic links. But that's about it.

### Diffing and patching portable builds

When you push successive versions of your game using butler, it attempts to create "patch"
files, that are used to upgrade from one version to the next using as little bandwidth
as possible.

If each build of your game is around 500MB, but you only make a few changes every version,
the patches may be around 1MB. So if a user is ten versions behind, they only have to download
10MB to update, not 500MB. So far so good (great, even!)

But *compression* can take two builds that look a lot alike, and make them look completely
different. The numbers often look as follows:

  * We have two builds, each build is 500MB uncompressed
  * A patch from one *uncompressed* build to the other is 1MB
  * When compressed, each build is around 230MB
  * A patch from one *compressed* build to the other is 150MB

How is that possible? When comparing both uncompressed versions, the changes were very
localized and easily encoded in a series of instructions to upgrade from one version to another - which makes up a patch file.

But the compressed versions don't have much in common - the changes had a cascading effect
and changed many other parts of the compressed file - and as a result, it's much longer to
describe "how to go from compressed file 1 to compressed file 2", than it is to describe
for their uncompressed counterparts.

**TL;DR it's much better to push the uncompressed version:**

  * itch.io will compress each build individually, and those will be used for first-time installations, and direct downloads from the website
  * small patches will be used to upgrade from one version to the next

## All-in-one executables

All-in-one executables are almost like portable builds, except instead of being a folder,
they're a single executable file.

Executables have a "data" section where anything that isn't code can be stored. All-in-one executables
use that section to store *all the resources needed by the game*, often in compressed form.

When the all-in-one executable is started, it either:

  * uses a temporary folder to extract all the resource files needed by the game

or, it

  * has a privileged relationship with the game, and can extract individual resources on-demand, whenever the game needs them

If the resources are stored in compressed form, then patches to
upgrade from one version of the game to the next will be
unnecessarily large (see "Diffing & patching" above).

If they're not compressed, then the situation is almost as good
as with a real portable build.

> Almost as good, because every file "modified" by a patch
> is duplicated on disk when upgrading (so that if the upgrade
> fails, a working version of the game remains). In a single
> file scenario, if the file is 2GB, then an additional 2GB
> on disk will be used on every upgrade. That's far from optimal!

## Installers

In short, **installers are the absolute worst**.

  * Unlike portable builds, there are extra steps between downloading and playing
  * Like all-in-one executables, they store the game's resources (and executable!) in compressed form

So, not only are patches between installer versions large (see 
"Diffing & patching" above), but they're also useless!

Since the installer file is not the "final form" of the game
(it's not playable), it's useless trying to patch it from one
version to the next.

> In fact, the installer is removed by [the itch app][] after installation, so even there was a way to patch it efficiently, it wouldn't.

Besides, installers bring a host of other problems:

  * Some installers require the Administrator password to proceed

> Your players may not have administrative rights on their
gaming computer! They shouldn't be needed to play your game.

  * Some installers have an interface that cannot be hidden (ie. they have no silent mode)
  * Some installers do not let users pick the install location

And finally, all installers have various failure conditions that
aren't just "your disk is full".

> For example, MSIs (microsoft installer packages) like to fail
when the game was already installed before on this computer. Even if all files have been removed since. Air installers are even more specific about what state a computer should be in before it will deem it worthy of having your game installed to it.

### Where installers shine

That said, there are actual reasons to distribute your game or application as an installer.

  * Your game requires some packages to be installed in order to run correctly
  * You want to create a shortcut on the desktop and in the start menu
  * You want to associate some file types with your application
  * You want to create some registry entries
  * etc.

These problems are addressed in part by [the itch app][]:

  * You can include an [app manifest][], listing prerequisites that [the itch app][] should install before first launching your game
  * Automatic shortcut creation is on its way
  * File associations and registry entries could also be handled by the app, via entries in the manifest

[the itch app]: https://itch.io/app
[app manifest]: https://itch.io/docs/itch/integrating/manifest.html

However, the app should be optional - there should be a good way
for users to download and run your game even without using [the itch app][].

For now, *install instructions* can be added to your itch.io page
to help get your users up and running.

In the future, we could imagine the itch.io backend generating installers directly for each game, so that they can be properly installed without [the itch app][].

This would re-use the information included in the [app manifest][], and would require absolutely no effort on your part. In the meantime, the easy way to handle non-tech-savvy users is to direct
them to [the itch app][].

## I still really want to push a single file though

If you've read all this, but you have your reasons, you can still
pass a single file to butler push and it'll work transparently.

butler will behave as if you had created a folder and put your
single file in it. The upload will work as usual for users of [the itch app][]
(although diffs might be unusually large, if your single file
is compressed or just has high entropy), and users who download
from [the itch.io website][] will download the single file
directly (it won't be packaged in a .zip for them).

[the itch.io website]: https://itch.io/
