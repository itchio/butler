
# Installing butler

## The human-friendly way

You can download stable and bleeding-edge builds of butler from its itch.io page:

  - <https://fasterthanlime.itch.io/butler>

Better yet, you can install it using the [itch app](https://itch.io/app), so it stays up-to-date.

## The automation-friendly way

You can download the latest, bleeding-edge version of butler from broth:

  - <https://broth.itch.ovh/butler>

broth is the download server used by the desktop app to install its own copy of butler.

The `-head` channels are bleeding-edge, the other ones are stable.

This is recommended if you need to install butler as part of a script, perhaps for continuous deployment

If you want to get the latest stable, for example, you could curl or wget the following URL:

  - <https://broth.itch.ovh/butler/linux-amd64/LATEST/archive/default>

You can substitute `linux-amd64` with any channel listed on broth.

## Adding butler to your path

Adding an executable to your path allows you to launch it from anywhere,
no matter which directory you're currently in.

### On Windows

[Follow this article](http://www.howtogeek.com/118594/how-to-edit-your-system-path-for-easy-command-line-access/) to add the folder where you downloaded butler.exe to your path.

*Alternatively, Windows will look into the current working directory when
looking for commands*

### On Linux

If you downloaded butler to a directory (let's say `~/bin`), you first need
to mark it as executable. From a terminal, run:

```sh
chmod +x ~/bin/butler
```

(Replacing `~/bin` with the folder you actually want to store butler into)

Then, edit the `~/.bashrc` file (`~` is your home directory) and add this line
at the end:

```sh
export PATH="$PATH:~/bin"
```

(Again, replacing `~/bin` as appropriate)

You'll need to close and start a new terminal to apply the changes. You should
now be able to move on to the `First run` section.

**Alternatively**, if you want to use the version installed by the itch app,
you can skip the chmod command and use this line in your `~/.bashrc` instead:

```sh
export PATH="$PATH:~/.config/itch/apps/butler"
```

### On macOS

Follow the Linux instructions, except:

  * On macOS, the `~/.bash_profile` file is used instead of `~/.bashrc`
  * If you want to use the itch app version, use this line in your `~/.bash_profile` instead:

```sh
export PATH="$PATH:~/Library/Application Support/itch/apps/butler"
```

*(don't forget the double-quotes, they're needed because there is a space in Application Support)*

As with Linux, don't forget to close and re-open your terminal to apply the changes.

## First run

To make sure butler is installed properly, open a terminal (`cmd.exe` on Windows),
and type the following command:

```bash
butler -V
```

*(that's a capital V, casing matters)*

It should print something like that:

```bash
head, built on Sep 13 2018 @ 10:59:39, ref 30fe1c38a9611d6b17dc61c7d4fb9582aa369d41
```

Or if you're using a stable version, `head` will be replaced by a [semantic version
number](http://semver.io/).

Here's how it looks on Windows:

![](images/butler-cmd-exe.png)

*Note: of course, you can also run butler from PowerShell. But if you know
about PowerShell you probably didn't need to read most of this page anyway.*

## Appendix: Finding butler

If you ever forget where you put your butler.exe, the `butler which` command
will print its complete path.

