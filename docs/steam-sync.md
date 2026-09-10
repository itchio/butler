# Syncing builds from Steam

If your game is already on Steam, `butler steam-sync` copies its builds to
itch.io. It downloads the depots of a Steam app, groups them into one
directory per platform, and pushes each directory to an itch.io channel with
`butler push`.

The depot downloader supports a cache directory so that subsequent syncs can be
downloaded as patches from Steam without having to redownload your whole build,
and then `butler` supports patch uploads, so you only upload what has changed
since your last sync.

A config file can be written to manage your entire sync pipeline, so you can
simply run `butler steam-sync --from-config` periodically to keep your builds
synchronized into itch.io.

This feature is experimental. The commands don't show up in `butler --help`
yet, and if you hit a problem, let us know.

## Overview

A sync goes through these steps:

  1. `butler steam-login` connects a Steam account. Steam only serves depot
     content to a logged-in account.
  2. `butler steam-key` stores a publisher Web API key. It's used to prove
     which apps your partner account controls. butler only supports syncing
     those.
  3. `butler steam-sync APPID user/game` maps the depots of that app to itch.io
     channels, downloads them, and pushes. Run it with `--dry-run` first to see
     the plan without downloading anything. Like `butler push` it will either
     create a new channel on itch.io, or push a patch to the existing channel
     if you've already run a sync.

## Logging in to Steam

```bash
butler steam-login
```

By default this shows a QR code. Scan it with the Steam mobile app and approve
the login there. Your password never passes through butler this way. If this
doesn't work for you, or you'd rather type your credentials:

```bash
butler steam-login --password
butler steam-login --password --user myaccount
```

You'll be prompted for the password and, if the account has Steam Guard
enabled, for the code or for approval in the mobile app. The username and
password are not stored, they are only used during authentication to exchange for
an authorization code.

### What logging in grants

Unfortunately, Steam has no scoped API for downloading a game's files, so
butler logs in as your Steam account the same way the Steam client does. The
token Steam hands back grants full access to that account, not just to
downloads.

> Done using this feature? Log out to delete the stored credential instead of
> leaving it around on your system.

Everything involving Steam happens locally on your computer. butler talks to
Steam directly, and the token is never sent to itch.io or any other server. It
is saved as `steam_creds.json` next to butler's itch.io credentials, so the
`-i` / `--identity` flag moves both together. Your password is never stored.

If you'd rather not keep a token on disk, see `--no-save` under
[Running from CI](#running-from-ci).

When you stop using this feature, delete the saved credentials:

```bash
butler steam-logout
```

This only deletes the local file, and Steam still considers the token valid.
To revoke it, sign out of other devices from your Steam account settings.

## Storing the publisher key

Steam can't tell a developer from a player when serving downloads. To make
sure builds only move for apps you publish, butler refuses to sync an app
unless a publisher Web API key confirms your partner group controls it.
Create one at <https://partner.steamgames.com/pub/groups/> under your
publisher group, then:

```bash
butler steam-key
```

You'll be prompted for the key. It can also be passed as an argument, but
that leaves it in your shell history:

```bash
butler steam-key ABCDEF0123456789...
```

butler verifies the key against Steam before saving it and reports how many
apps it controls. butler uses the key only to list those apps and to refuse
anything else. The key can read other partner data for your group though, so
create one just for butler and delete it from the partner site when you're
done. To see the apps it controls:

```bash
butler steam-apps
```

`butler steam-apps --owned` instead lists the apps the logged-in Steam account
holds a license for. Owning a license isn't enough to sync. The publisher key
must control the app as well.

## Running a sync

```bash
butler steam-sync 123456 user/game --dry-run
```

Where:

  * `123456` is the Steam app id
  * `user/game` is the itch.io project, the same target you'd give to `butler push`
    but without a channel. The page URL, `user.itch.io/game`, works too.
    Channels are chosen per platform, see below. A target with a channel
    such as `user/game:windows` is rejected.

A dry run prints the plan and stops:

```
Half-Life 3 (app 123456), branch public, build 987654

  user/game:linux  (1.2 GB on disk, 1.2 GB to download)
    depot 123457    Linux Content                     1.1 GB  manifest 8823...
    depot 123459    Shared Content                    100 MB  manifest 1029...  [all platforms]

  user/game:windows  (1.3 GB on disk, 1.3 GB to download)
    depot 123458    Windows Content                   1.2 GB  manifest 5561...
    depot 123459    Shared Content                    100 MB  manifest 1029...  [all platforms]

  skipped:
    depot 123460    Soundtrack DLC                    DLC for app 123470
    depot 123461    French                            french language pack

Each channel is pushed with --userversion 987654.
```

When the plan looks right, drop `--dry-run`. butler then:

  * Logs in to itch.io (or prompts you to, like `butler push`)
  * Checks each channel's latest build and skips channels whose
    [version number](pushing.md#specifying-your-own-version-number) already
    matches the Steam build id
  * Downloads each remaining depot once, even if several channels share it
  * Assembles one directory per channel and pushes it, with the Steam build
    id as the user version

Since the Steam build id is stored as the version of every itch.io build,
syncing the same Steam build twice does nothing. Pass `--force` to push
anyway.

Because the tool downloads the entire depot to your disk before pushing with
butler, you need to ensure you have enough disk space available. When running
without `--cache-dir`, a temporary cache dir is created to store downloads, and
is cleaned up after execution. If you have a larger game and intend to sync
often, we highly recommend using `--cache-dir` to avoid redownloading your
depots on every sync.

### Using a config file

A config file can be used to fully describe your sync process so you can simply
run `butler steam-sync --from-config config.toml` to synchronize all the
specified apps in a single command. The config file uses the TOML format:

```toml
cache_dir = ".steam-sync-cache"

[[sync]]
app = 123456
target = "user/game"

[[sync]]
app = 234567
target = "user/other-game"
branch = "beta"
skip = [234570]
cache_dir = "/mnt/big/other-game-cache"

[sync.map]
234568 = "win-64"
```

Run `butler steam-sync --from-config path/to/config.toml`. Each entry is synced
in order. A failed entry is reported and the remaining entries still run; the
exit status is non-zero if any failed.

At the top level, before the first `[[sync]]`:

  * `cache_dir`: keep downloads between syncs for every entry. Each app gets
    its own subdirectory named by app id, so the example above stages app
    123456 in `.steam-sync-cache/123456`. A relative path is relative to the
    config file. Without it, entries that set no `cache_dir` of their own are
    downloaded into a temporary directory and removed after the push.

Each `[[sync]]` entry takes:

  * `app` and `target`: required, the same two arguments as the command line
  * `branch`: default `public`
  * `skip`: depot ids to leave out
  * `map`: depot id to channel name, in a `[sync.map]` table
  * `cache_dir`: keep this entry's downloads between syncs, overriding the top
    level `cache_dir`. It is used as is, with no app id subdirectory, so give
    each entry its own. A relative path is relative to the config file.
  * `hidden`: create new channels as hidden

There is no password field. A private branch's password is given with
`--password` or the `BUTLER_STEAM_BRANCH_PASSWORD` environment variable.

The file and the command line are not combined. `--from-config` cannot be
given together with an app id, and with it the per-app flags (`--branch`,
`--map`, `--skip`, `--hidden`) are an error. `--cache-dir` is the exception:
with `--from-config` it sets the top level `cache_dir`, taking precedence
over one in the file, and is relative to the working directory.
`--dry-run`, `--force` and `--no-push` apply in both cases. `--dry-run` with
a config file prints the plan for every entry.

### How depots become channels

Each depot on Steam declares which operating systems it targets, and
optionally an architecture. butler turns that into itch.io channel names:

  * One channel per platform: `windows`, `linux`, `mac`
  * If any depot of a platform declares an architecture, one channel per
    architecture instead, for example `windows-64` and `windows-32`. Depots
    of that platform without an architecture are copied into each.
  * Depots with no platform at all are copied into every channel
  * DLC depots, depots shared from another app, and language packs other
    than English are left out

itch.io sets platform tags from the channel name, see
[Channel names](pushing.md#channel-names).

To override the automatic placement, send a depot to a channel of your
choosing. The flag is repeatable:

```bash
butler steam-sync 123456 user/game --map 123461=french --map 123459=windows
```

A mapped depot goes only where you sent it. Depots mapped by hand aren't
copied into the platform channels, and platform detection doesn't apply to
them. This is also how you keep a depot the automatic placement would skip,
such as a language pack.

When two depots in the same channel ship the same file path, the depot with
the higher id wins. That matches how Steam mounts them.

To leave a depot out entirely:

```bash
butler steam-sync 123456 user/game --skip 123460
```

If no depot declares a platform, everything ends up in a channel called
`all` with a warning. Use `--map` to give it a proper name.

### Branches

The `public` branch is synced by default. To sync another branch:

```bash
butler steam-sync 123456 user/game --branch beta
butler steam-sync 123456 user/game --branch playtest --password hunter2
```

`--password` is only needed for password-protected branches. Steam hides
newer private branches completely until the password is given, so one may
be missing from the branch list butler shows and still sync fine with
`--password`.

Steam has two kinds of password branch. The older kind lists an encrypted
manifest per depot in the app info, and butler decrypts it with the
password. The newer kind withholds the depot section entirely and is not
supported yet: the sync fails with "has no manifest on branch". To see
which kind a branch is, without a password or a sync:

```bash
butler steam-info 123456
```

It prints every branch and, per depot, whether each branch has a manifest,
an encrypted one, or none. Include its output when reporting a problem with
a private branch.

Channel names don't change with the branch. If you want a beta branch to
land in `windows-beta` rather than `windows`, use `--map` for each depot.

### Keeping downloads between syncs

By default butler downloads into a temporary directory and removes it when the
push is done. Every sync then downloads the full build again. To download only
what changed on Steam since the last sync, keep a cache directory:

```bash
butler steam-sync 123456 user/game --cache-dir ~/steam-sync/mygame
```

The directory holds one download per depot plus the assembled channel
directories, which are hardlinked from the depot downloads so they cost no
extra disk space. An interrupted download resumes from where it stopped on
the next run. Use one cache directory per app. With a config file, a single
top level `cache_dir` does this for you by giving each app a subdirectory,
see the config file section above.

To look at what would be pushed without pushing it:

```bash
butler steam-sync 123456 user/game --cache-dir ~/steam-sync/mygame --no-push
```

butler prints where each channel directory is and stops. `--no-push` needs
`--cache-dir`, since the temporary directory would be removed on exit.

### New channels

Pushing to a channel that doesn't exist yet creates a download on your itch.io
page. Pass `--hidden` to create new channels as
[hidden](pushing.md#appendix-f-pushing-to-a-hidden-channel), so you can check
the build before players see it. Channels that already exist are not
affected.

### Steamworks SDK warning

If a channel directory contains files such as `steam_api64.dll`,
`libsteam_api.so` or `steam_appid.txt`, butler prints a notice after
assembling it. A build that initializes Steam at startup may not run for
itch.io players who don't have Steam. The sync still goes ahead. If this is a
problem, publish a Steam branch with Steam integration disabled and sync that
branch instead.

## Running from CI

Instead of a saved login, credentials can be passed to `steam-sync`,
`steam-apps` and `steam-key` through flags or environment variables:

| Flag                    | Environment variable          |
|-------------------------|-------------------------------|
| `--steam-refresh-token` | `BUTLER_STEAM_REFRESH_TOKEN`  |
| `--steam-account-name`  | `BUTLER_STEAM_ACCOUNT_NAME`   |
| `--steam-publisher-key` | `BUTLER_STEAM_PUBLISHER_KEY`  |

A refresh token needs the account name that goes with it. Flags win over
environment variables, and both win over the saved file. Credentials given
this way are never written to disk.

To obtain a token without saving it on the machine you're on:

```bash
butler steam-login --no-save
```

This prints the refresh token to stdout after you complete the login. Store
it, along with the account name and publisher key, as secrets in your CI
system alongside `BUTLER_API_KEY` (see [Logging in](login.md#running-butler-from-ci-builds-github-actions-gitlab-ci-etc)).

A sync job then looks like this:

```bash
butler steam-sync 123456 user/game --cache-dir "$CACHE_DIR"
```

or, with a config file:

```bash
butler steam-sync --from-config sync.toml
```

Steam refresh tokens eventually expire, and are revoked when you sign out of
all devices from your Steam account. When that happens the sync fails and
asks you to log in again. Run `butler steam-login --no-save` once more and
update the secret.

## Troubleshooting

Since this feature is new, you may run into problems not listed here. Run
with `-v` for verbose output when reporting one, and include the `--dry-run`
plan.

### "getting decryption key for depot"

Steam hands out depot decryption keys freely for released apps, but for some
unreleased apps it only does so when the logged-in account owns the game.
Owning it through the partner site isn't always enough. Grant the Steam
account you logged in with a license for the app, through a Steamworks key or
by adding it to the app's package, and run the sync again.

### A channel is skipped but the build on itch.io is wrong

butler skips a channel when its newest build, including one still processing,
carries the current Steam build id as its version. If a push was interrupted
or a build failed processing on itch.io, that build still matches and the
sync does nothing. Run with `--force` to push it again.

### Running out of disk space

Without `--cache-dir`, downloads are staged in a `steam-sync` directory next
to butler's credentials file, not in the system temp directory, and removed
when the sync finishes. A large game needs that much free space on that
drive. Pass `--cache-dir` to stage somewhere with room, which also makes the
next sync faster.

If a sync was killed, its staging directory may be left behind. butler
removes leftovers older than a day on its next run. To clean up by hand,
delete the `tmp-*` directories under:

  * Linux: `~/.config/itch/steam-sync/`
  * Mac: `~/Library/Application Support/itch/steam-sync/`
  * Windows: `%USERPROFILE%\.config\itch\steam-sync\`

If you pass `-i` to point butler at a different credentials file, the
`steam-sync` directory sits next to that file instead.

### "has no manifest on branch"

The branch exists but Steam did not include the depot's manifest for it in
the app info. For a password branch this means the newer private-branch
mechanism, which butler does not support yet. Run `butler steam-info APPID`
and report the output.

### "app ... has no branch"

The error lists the branches Steam reports for the app. Branch names are
matched exactly as Steam spells them. A private branch is not reported at
all until its password is given, so if you know the branch exists, retry
with `--password`.

### "not in the list of apps your Steam publisher key controls"

The publisher key belongs to a partner group that doesn't own this app. Check
`butler steam-apps` for the list it does control, and store a key from the
right group with `butler steam-key`.

## Command reference

```
butler steam-login [--password] [--user NAME] [--no-save]
butler steam-logout
butler steam-key [KEY]
butler steam-apps [--owned]
butler steam-info APPID
butler steam-sync APPID user/game [flags]
butler steam-sync --from-config FILE [flags]
```

Flags for `steam-sync`, per app (command line only):

  * `--branch NAME`: Steam branch to sync, default `public`
  * `--map DEPOTID=CHANNEL`: send a depot to a specific channel, repeatable
  * `--skip DEPOTID`: leave a depot out, repeatable
  * `--cache-dir DIR`: keep downloads here between syncs. Also allowed with
    `--from-config`, where it is the top level `cache_dir` and each app
    stages in `DIR/<app id>`
  * `--hidden`: create new channels as hidden

Flags that apply to the run, with or without a config file:

  * `--from-config FILE`: run every entry of a sync config file instead of
    taking an app id and target
  * `--password PASS`: password for a private branch, also read from
    `BUTLER_STEAM_BRANCH_PASSWORD`
  * `--dry-run`: print the plan and exit
  * `--no-push`: download and assemble, then stop. Every entry needs a cache
    directory
  * `--force`: push even if the channel already has this Steam build

All of these commands support `--json` for machine-readable JSON-lines
output, like the rest of butler. With it, `butler steam-sync --dry-run` emits
the full plan as a `result` message.
