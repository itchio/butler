# Syncing builds from Steam

If your game is already on Steam, `butler steam-sync` copies its builds to
itch.io. It downloads the depots of a Steam app, groups them into one
directory per platform, and pushes each directory to an itch.io channel with
`butler push`. You don't need to rebuild or re-upload anything yourself.

The commands in this section are still experimental. They don't show up in
`butler --help`, but they are available in every butler release that has them.

## Overview

A sync goes through these steps:

  1. `butler steam-login` connects a Steam account. Steam only serves depot
     content to a logged-in account.
  2. `butler steam-key` stores a publisher Web API key. It proves which apps
     your partner account controls. butler only syncs those.
  3. `butler steam-sync APPID user/game` maps the depots of that app to itch.io
     channels, downloads them, and pushes. Run it with `--dry-run` first to see
     the plan without downloading anything.

After that, run `butler steam-sync` again after each Steam release. Channels
that already have the current Steam build are skipped.

## Logging in to Steam

```bash
butler steam-login
```

By default this shows a QR code. Scan it with the Steam mobile app and approve
the login there. Your password never passes through butler this way. If you'd
rather type your credentials:

```bash
butler steam-login --password
butler steam-login --password --user myaccount
```

You'll be prompted for the password and, if the account has Steam Guard
enabled, for the code or for approval in the mobile app.

### What logging in grants

Steam has no scoped API for downloading a game's files, so butler logs in as
your Steam account the same way the Steam client does. The token Steam hands
back grants full access to that account, not just to downloads. Treat it like
a password.

Everything happens on your computer. butler talks to Steam directly, and the
token is never sent to itch.io or any other server. It is saved as
`steam_creds.json` next to butler's itch.io credentials, so the `-i` /
`--identity` flag moves both together. Your password itself is not stored.

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
    but without a channel. Channels are chosen per platform, see below.

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
them.

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

`--password` is only needed for password-protected branches.

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
the next run. Use one cache directory per app.

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

Steam refresh tokens eventually expire, and are revoked when you sign out of
all devices from your Steam account. When that happens the sync fails and
asks you to log in again. Run `butler steam-login --no-save` once more and
update the secret.

## Command reference

```
butler steam-login [--password] [--user NAME] [--no-save]
butler steam-logout
butler steam-key [KEY]
butler steam-apps [--owned]
butler steam-sync APPID user/game [flags]
```

Flags for `steam-sync`:

  * `--branch NAME`: Steam branch to sync, default `public`
  * `--password PASS`: password for a private branch
  * `--map DEPOTID=CHANNEL`: send a depot to a specific channel, repeatable
  * `--skip DEPOTID`: leave a depot out, repeatable
  * `--dry-run`: print the plan and exit
  * `--cache-dir DIR`: keep downloads here between syncs
  * `--no-push`: download and assemble, then stop. Requires `--cache-dir`
  * `--force`: push even if the channel already has this Steam build
  * `--hidden`: create new channels as hidden

All of these commands support `--json` for machine-readable JSON-lines
output, like the rest of butler. With it, `butler steam-sync --dry-run` emits
the full plan as a `result` message.
