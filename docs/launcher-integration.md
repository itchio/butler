
# Building your own itch.io app launcher

butler provides a daemon mode called `butlerd` that exposes everything butler
can do as a JSON-RPC 2.0 service. If you're building your own itch.io game
launcher or any client app that needs to log users in, browse their library,
install games, and run them, talking to butlerd is the supported way to do it.

This page walks through the lifecycle of a launcher end-to-end. It is
language-agnostic: butlerd is just a TCP server speaking JSON, so you can
integrate from any language that can open a socket. Where examples are useful
they're shown as raw JSON-RPC messages.

> The full reference for every method, parameter, result, and notification
> butlerd exposes lives at <https://itchio.github.io/butler/butlerd/>. Keep that
> open while you read this page. This guide explains the *flow*; the spec
> documents the *shape* of every call.

If you're writing a launcher in JavaScript or TypeScript, there is an
official client library on npm called [`@itchio/butlerd`][butlerd-npm] that
handles process supervision, the handshake, and request/notification routing
for you. Everything below applies whether or not you use it.

[butlerd-npm]: https://www.npmjs.com/package/@itchio/butlerd

## What you're working with

There are two pieces:

  * **butler** is the binary. The same binary that uploads builds, runs
    `butler push`, generates patches, and so on.
  * **butlerd** is what you get when you run that binary as `butler daemon …`.
    It's a long-lived process that listens on a local TCP port and accepts
    JSON-RPC 2.0 requests. Every install, fetch, login, and launch goes through
    this daemon. Your launcher never invokes `butler` subcommands directly.

butlerd also keeps its own SQLite database. That database is where saved
logins, the list of installed games (called *caves*), install locations, and
cached metadata live. **The same `--dbpath` reused across runs is what gives
your users persistent profiles and a working library.**

## 1. Provide a butler binary with your launcher

You need a butler binary on disk before you can do anything. There are three
reasonable approaches:

  * **Bundle a pinned butler binary** with your launcher's installer. Simple,
    but you are responsible for updating it. **Updating is critical to ensure
    that your butler binary is compatible with the current live version of the
    itch.io API.**
  * **Download butler at runtime** from the [broth](https://broth.itch.zone/) distribution channel, the
    same one [the itch.io app][itch-app] uses to keep its bundled butler up to
    date. The latest version string for a platform lives at
    `https://broth.itch.zone/butler/<goos>-<goarch>/LATEST`, and the matching
    archive at
    `https://broth.itch.zone/butler/<goos>-<goarch>/<version>/archive/default`.
    The platform slug uses Go's OS/arch naming, so `linux-amd64`,
    `windows-amd64`, `darwin-arm64`, and so on.
  * **Ask the user to install butler themselves** and let them point your
    launcher at the path. Suitable for power-users or developers who already
    have butler installed.

[itch-app]: https://itch.io/app

## 2. Spawn butlerd

Start the daemon as a subprocess of your launcher:

```
butler daemon --json --transport tcp --keep-alive \
    --dbpath /path/to/your-launcher/butler.db \
    --address https://itch.io \
    --user-agent "your-launcher/1.2.3" \
    --destiny-pid <your-launcher-pid>
```

The flags worth knowing:

  * `--json` is **required**. Without it the daemon prints a friendly error and
    exits; the daemon doesn't have a human-friendly mode.
  * `--transport tcp` listens on a random local port. (`stdio` is also
    available if you'd rather pipe over the subprocess's stdin/stdout; use it
    when you can't open extra sockets.)
  * `--keep-alive` lets the same daemon accept multiple TCP connections during
    its lifetime instead of exiting after the first one disconnects. You almost
    always want this.
  * `--dbpath` points at the SQLite file butlerd uses to store *everything*
    persistent. Pick a stable per-user location and reuse it forever. Back it
    up if your launcher cares about losing the install library.
  * `--address` is the itch.io site to talk to. Use `https://itch.io` in
    production.
  * `--user-agent` is included on every HTTP request butler makes to itch.io.
    Set it to your launcher's name + version so traffic is attributable.
  * `--destiny-pid` ties the daemon's lifetime to a process ID. When the
    process with that PID exits, butlerd shuts itself down. Pass your
    launcher's own PID and you'll never leak orphan daemons.
  * `--log` (optional) writes every JSON-RPC request to stderr. Very useful
    while developing your client.

As soon as butlerd starts up it will print **one line of JSON to stdout** that
tells you where to connect and what secret to use:

```json
{"type":"butlerd/listen-notification","secret":"…uuid-quad…","tcp":{"address":"127.0.0.1:54321"}}
```

Capture that line, parse it, and you're ready to connect. Your launcher should
read butlerd's stdout until it sees a `butlerd/listen-notification` object.
Other status objects may appear before it.

## 3. Connect, authenticate, and call methods

The wire protocol is **JSON-RPC 2.0 over a raw TCP socket**, with each message
**terminated by a single `\n` (newline)**. There are no Content-Length headers
and no HTTP framing, just a stream of JSON objects, one per line, in both
directions.

That means a connected client looks roughly like:

  1. Open TCP socket to the address from the listen-notification.
  2. Write a JSON-RPC request followed by `\n`.
  3. Read newline-terminated JSON objects from the socket and dispatch them by
     `id` (responses to your requests) or by `method` (notifications and
     server-initiated requests).

### The handshake

The very first request you send on a new connection **must** be
`Meta.Authenticate`, with the secret from the listen-notification:

```json
{"jsonrpc":"2.0","id":1,"method":"Meta.Authenticate","params":{"secret":"…uuid-quad…"}}
```

The daemon replies with `{"result":{"ok":true}}` and from that point on the
connection is good for any other call. No subsequent request needs to repeat
the secret.

The secret is regenerated every time butlerd starts. If your daemon restarts,
re-read the listen-notification and re-authenticate.

### Requests, responses, and notifications

Once authenticated, you make calls the normal JSON-RPC way: send a request
with an `id`, get a response with the same `id` back. Notifications (no `id`)
flow from butlerd to you to report progress on long-running operations.

A few methods butlerd offers also go *the other way*: while you're in the
middle of a call (a "conversation"), butlerd may send **a request to you** and
expect a response. This is how interactive prompts work: an upload picker
when a game has more than one compatible build, a manifest action the user
needs to pick, a license to accept. Your client must be willing to receive a
request from the server in the middle of its own outstanding call and send
back a response.

Your client needs three things:

  1. A way to send a request and `await` its matching response by `id`.
  2. A dispatcher for incoming notifications, keyed by method name.
  3. A dispatcher for incoming server-to-client requests, also keyed by method
     name, that produces a response object.

The official TypeScript client wraps these in a `Conversation` object you can
attach `onNotification` and `onRequest` handlers to before making a call. If
you're writing your own client in another language, model it the same way.

### The message catalog

Every method, every params type, every result type, and every notification is
listed in the [butlerd specification][spec]. Method names are namespaced:
`Meta.*`, `Profile.*`, `Fetch.*`, `Install.*`, `Launch`, `CheckUpdate`,
`Uninstall.*`, `Downloads.*`, `System.*`, and so on. Whenever this guide
mentions a method like `Profile.LoginWithOAuthCode`, look it up there for the
exact field names.

[spec]: https://itchio.github.io/butler/butlerd/

## 4. Log the user in

butlerd handles credential storage on your behalf. Your launcher never has to
persist tokens. You just remember the **profile ID** (an integer; the user's
itch.io user ID) and ask butlerd to use it on subsequent runs.

The supported login flow for a public launcher is **OAuth 2.0 with PKCE**.

### Register an OAuth application

Register an OAuth application for your launcher through your itch.io account
settings:

  * <https://itch.io/user/settings/oauth-apps>

You'll receive a client ID. You'll also need to choose a **redirect URI** that
your launcher controls. The standard pattern for a desktop launcher is to
register a custom URL scheme with the operating system, something like
`my-launcher://oauth-callback`, so that when itch.io redirects the user's
browser to that URL, the OS hands the URL back to your running launcher
process.

Each platform has its own way of registering custom URL schemes; that's a
detail of your launcher framework, not of butlerd.

### The OAuth + PKCE flow

The flow has four steps:

**1. Generate PKCE values.** A random code verifier (32 bytes is fine), and a
code challenge that is the base64url-encoded SHA-256 hash of the verifier.
Generate a random `state` value too, for CSRF protection.

**2. Open the user's browser** to itch.io's authorize endpoint:

```
https://itch.io/user/oauth?
  client_id=<your-client-id>
  &scope=itch
  &redirect_uri=<your-redirect-uri>
  &response_type=code
  &state=<random-state>
  &code_challenge=<code-challenge>
  &code_challenge_method=S256
```

The user signs into itch.io (if they aren't already) and approves your app.
itch.io then redirects them to your redirect URI with `?code=…&state=…`
appended.

**3. Receive the callback** in your launcher (via your custom URL scheme
handler). Verify the returned `state` matches the one you generated. Drop the
flow with an error if it doesn't.

**4. Exchange the code via butlerd.** Send a `Profile.LoginWithOAuthCode`
request with the code, the original code verifier, your redirect URI, and your
client ID. The daemon performs the token exchange and stores the resulting
credentials.

```json
{"jsonrpc":"2.0","id":42,"method":"Profile.LoginWithOAuthCode","params":{
  "code":"…","codeVerifier":"…","redirectUri":"my-launcher://oauth-callback","clientId":"…"
}}
```

The result contains a `profile` object. Save `profile.id` somewhere your
launcher will see on next startup. That's all you need to log them back in
later.

### API key login (alternative)

For headless tools, CI pipelines, or development utilities where opening a
browser doesn't make sense, butlerd also supports `Profile.LoginWithAPIKey`.
The user generates an API key from their itch.io
[API keys settings](https://itch.io/user/settings/api-keys) and pastes it into
your tool, which forwards it to butlerd. The whole exchange is
non-interactive: no browser, no prompts.

Use it for power-user or CLI tools. Consumer-facing launchers should stick
to OAuth.

## 5. Saved profiles and auto-login

After any successful login, butlerd has the credentials stashed in its
SQLite database keyed by profile ID. On every subsequent launcher startup:

  1. Call `Profile.List`. This returns the array of remembered profiles, each
     with `id`, the timestamp of `lastConnected`, and a snapshot of the
     `user` (username, display name, avatar URL).
  2. Pick one (or let the user pick if there are several) and call
     `Profile.UseSavedLogin` with its `profileId`. The daemon validates the
     stored credentials against itch.io and returns the freshened `profile`.

Two notes:

  * **Time-box the call.** `Profile.UseSavedLogin` makes a network round-trip.
    The official itch.io app gives it a five-second budget so the launcher
    still opens promptly when the user is offline; on timeout, it falls
    through to a "you can keep using cached data" mode and lets the user
    retry.
  * **Logging out** is `Profile.Forget` with the profile ID. The credentials
    are wiped from the database and the profile no longer appears in
    `Profile.List`.

If you need somewhere to stash per-profile launcher state (last opened
collection, UI preferences, etc.), butlerd exposes
`Profile.Data.Put` / `Profile.Data.Get` as a string key/value store scoped to
each profile.

## 6. Fetch the user's library

Once a profile is active, the `Fetch.*` family of methods gets you everything
you need to build a library UI.

A pattern shared by every `Fetch.*` call: by default it returns whatever's in
butler's local cache *immediately*, with `stale: true` set on the response if
the data hasn't been refreshed from itch.io recently. To force a network round
trip, pass `fresh: true` in the params. The recommended UI pattern is to call
once with `fresh: false` to paint UI from the cache, then call again with
`fresh: true` to refresh, or use the cached results if `stale` is false.

The methods you'll reach for most:

  * `Fetch.ProfileOwnedKeys`: every download key the user owns. Each key
    carries the associated game.
  * `Fetch.ProfileCollections` and `Fetch.CollectionGames`: the user's
    collections and what's in them.
  * `Fetch.ProfileGames`: games the user has authored (relevant for
    creator-facing launchers).
  * `Fetch.Game`: fetch a single game by ID, e.g. for a detail page.
  * `Fetch.Caves` and `Fetch.Cave`: the user's *installed* games. (See below.)
  * `Fetch.Commons`: flat ID-only summaries of every key, every cave, and
    every install location. Cheap; use it for "do I own this? is it
    installed?" lookups in bulk UI.
  * `Fetch.Search.Games`: search.

### Caves: the installed-game record

butlerd represents every install with a **cave**. A cave has a UUID (`caveId`),
the game and upload it was installed from, optional build metadata if it's a
wharf-channel install, install location and folder, size on disk, last-played
timestamp, total seconds run, and so on. One game can have multiple caves
(rare, but supported). Most launcher UIs end up doing a
`Fetch.Caves` filtered by `gameId` to badge games with their installed state.

## 7. Install a game

Installs go in three steps: **queue**, **plan**, **perform**.

**Queue** (`Install.Queue`) tells butlerd *what* you want to install. You pass
the chosen `game`, the chosen `upload` (and optionally `build` for wharf
channels), an `installLocationId`, and a `reason` (`install`, `update`,
`reinstall`, or `version-switch`). butlerd allocates a task UUID for it and
returns the staging folder it'll work in. Pass `queueDownload: true` to have
butlerd manage the actual download.

**Plan** (`Install.PlanUpload`, optional but recommended) computes how much
disk space is needed and detects the installer type. Use the result to show a
"this will take 1.2 GB" confirmation step before committing.

**Perform** (`Install.Perform`) is the long-running call that downloads and
installs. Pass it the task ID from the queue step. While the call is in
flight, butlerd will stream you notifications:

  * `Progress`: periodic `{progress, eta, bps}` updates.
  * `TaskStarted`: fired at the start of each sub-task (download, install,
    update, heal). Includes total size.
  * `TaskSucceeded`: fired when each sub-task finishes successfully.

You **subscribe to those notifications by registering handlers on the
conversation before issuing the `Install.Perform` request**. They arrive on
the same connection while the call is outstanding.

### Picking an upload

If the user only knows they want to install "this game", you'll need to look
up which uploads (downloadable files) are available for it. `Fetch.GameUploads`
with `compatible: true` returns only uploads that match the user's current OS
and architecture, which is normally what you want. Show the user a picker if
there's more than one.

### Cancelling

If the user changes their mind mid-install, `Install.Cancel` with the task ID
aborts cleanly.

### Install locations and on-disk layout

A launcher needs at least one install location: a directory butlerd is allowed
to install games into. Locations are managed via
`Install.Locations.List`, `Install.Locations.Add`, `Install.Locations.Remove`,
and `Install.Locations.GetByID`. butlerd has no built-in default; on first
run, prompt the user for a path (or pick a sensible per-user one yourself,
like `<userData>/games`), call `Install.Locations.Add` with it, and remember
the returned ID.

Within an install location, the on-disk layout looks like this:

```
/path/to/install-location/
  overland/                  # one folder per cave, named from the game's URL slug
    .itch/
      receipt.json.gz        # what's installed here, written by butler
    Overland.exe
    data/
    ...
  game-12345/                # fallback folder name when no slug can be derived
  overland 2/                # auto-incremented suffix on name collisions
  downloads/
    quick-fox-jumps/         # staging folder for an in-progress install
    happy-cat-runs/
```

butlerd derives the per-cave folder name from the game's itch.io URL slug:
`https://finji.itch.io/overland` becomes `overland`. If there's no parseable
slug it falls back to `game-<id>`. If the chosen name already exists on disk,
butlerd appends ` 2`, ` 3`, and so on until it finds a free name (up to 200
tries). The collision check looks at the **filesystem**, not the database, so
it works even if some other tool put a folder there.

Staging folders live under `downloads/` inside the install location, named
with a random three-word petname like `quick-fox-jumps`. Partial downloads,
patch checkpoints, and other temporary state go there. They're cleaned up
when an install succeeds, but a long-running launcher should periodically
call `CleanDownloads.Search` and `CleanDownloads.Apply` to reclaim space
from cancelled or failed jobs.

The cave-to-folder mapping lives **only in butler.db**. Each cave row stores
its install-location ID and folder name, plus all the associated metadata
(game, upload, build, last-played, seconds run, and so on). butler does drop
a `.itch/receipt.json.gz` inside each install folder describing what was
installed there, and `Install.Locations.Scan` will walk an install location
and rebuild caves from those receipts if your database is lost. Recovery is
possible but not free, so back up `--dbpath`.

### Multiple launchers and shared install locations

Each butler.db is independent. If two launchers each have their own
`--dbpath` but point at the **same** install-location path on disk, neither
one's database can see the other's caves.

That means:

  * **Folder collisions are avoided automatically.** When launcher A installs
    Overland into `/games/overland`, launcher B installing the same game
    finds that folder taken and goes to `/games/overland 2`, because butler's
    name picker checks the filesystem rather than its own database.
  * **Each launcher downloads its own copy.** Bytes aren't deduplicated; the
    same game installed through two launchers takes twice the disk.
  * **Each launcher's `Uninstall.Perform` only removes its own caves.**
    Folders the other launcher created stay put.
  * **Staging folders use random petnames**, so concurrent downloads from
    different launchers won't collide under `downloads/`.

If you want your launcher to **share** installed games with other
butlerd-based launchers (including the official itch.io app), the only
clean way is to share the same `--dbpath` *and* the same install location
paths. Running two daemons against one `--dbpath` simultaneously is **not**
supported: butlerd assumes single-tenant access and SQLite write contention
will produce errors. So sharing means cooperating on which launcher's
daemon is running at any given moment, which is rarely worth it. Most
third-party launchers should keep their own `--dbpath` and their own
install-location paths and accept that installs are private to the
launcher.

### Installing without a cave

`Install.Queue` accepts `noCave: true` plus an explicit `installFolder`
(absolute path). butler installs directly into that folder and writes
nothing to its database. No cave record is created, so the install is
invisible to `Fetch.Caves`, `Launch` can't run it, and `CheckUpdate` can't
update it. This mode is useful for one-shot extracts, portable bundles, or
launchers that just want to use butler as a download/unpack engine without
adopting its state model.

### Driving multiple downloads

If you'd rather queue up a set of games and let butlerd serialize the
downloads, open a long-lived `Downloads.Drive` conversation. butlerd will emit
`DownloadsDriveStarted` / `DownloadsDriveProgress` / `DownloadsDriveFinished`
notifications as the queue advances.

## 8. Launch a game

`Launch` takes a `caveId`, a `prereqsDir` (a writable directory butler can use
to download prerequisite installers into), and optional sandboxing flags.
The call blocks for the full lifetime of the game.

Prerequisites (DirectX, .NET runtimes, Visual C++ redistributables, and so on)
are detected from the upload's manifest and installed automatically before the
game starts. While that's happening, butlerd streams:

  * `PrereqsStarted`: the list of prereqs about to be installed.
  * `PrereqsTaskState`: per-prereq progress (`pending`, `downloading`,
    `ready`, `installing`, `done`).
  * `LaunchRunning`: the game's actual process is up.
  * `LaunchExited`: the game has quit.

`Launch` may also send your launcher *interactive requests* mid-call. The
common ones:

  * `PickManifestAction`: the upload's manifest declared multiple actions
    (e.g. "Play", "Edit Levels"); ask the user which to run.
  * `AcceptLicense`: show a EULA, return whether the user accepted.
  * `ShellLaunch`: butlerd is handing off a file or folder for your launcher
    to open via the OS shell.
  * `HTMLLaunch`: the upload is an HTML5 game; butlerd needs your launcher
    to serve it locally and open a window.

Each of these arrives as a JSON-RPC request from the server during the
`Launch` call. Your client responds with the appropriate result object.

Sandboxing is opt-in and currently only meaningful on Linux. See the spec for
`SandboxOptions`.

## 9. Updates and uninstalls

To check for updates, call `CheckUpdate`. Pass an array of cave IDs to scope
it to specific games, or omit them to check everything installed. The result
is an array of `GameUpdate` records, each carrying the cave ID, the new
upload(s) to choose from, and whether the update is "direct" (i.e. on the same
channel the cave was originally installed from).

To apply an update, queue it through the same `Install.Queue` →
`Install.Perform` flow you'd use for a fresh install: pass the existing
`caveId` and `reason: "update"`, and butlerd will use wharf patching when
both ends have build metadata. (Patches mean a 30 MB game update typically
transfers a few hundred kilobytes. You don't have to do anything special to
get it; butlerd picks the right strategy on its own.)

`SnoozeCave` lets a user dismiss updates for a specific game until something
newer than the currently-installed upload appears.

Uninstalling is a single call: `Uninstall.Perform` with a `caveId`. Files come
off disk; the cave record is deleted. There is no undo, so confirm in your
UI.

## A typical launcher lifecycle

A typical launcher session looks like this:

```
on startup:
  ensure butler binary present
  spawn butler daemon, parse listen-notification
  open TCP, send Meta.Authenticate
  Profile.List
    -> if a remembered profile: Profile.UseSavedLogin (timeout: ~5s)
    -> else: run OAuth+PKCE flow, then Profile.LoginWithOAuthCode

main window:
  Fetch.ProfileOwnedKeys (fresh:false; refetch with fresh:true)
  Fetch.Caves to badge installed games
  CheckUpdate in the background

on user clicks install:
  Fetch.GameUploads (compatible:true)
  Install.Queue -> Install.PlanUpload -> user confirms -> Install.Perform
  (subscribe to Progress / TaskStarted / TaskSucceeded notifications)

on user clicks play:
  Launch(caveId, prereqsDir)
  (subscribe to PrereqsStarted / PrereqsTaskState / LaunchRunning / LaunchExited;
   answer PickManifestAction / AcceptLicense / ShellLaunch / HTMLLaunch as needed)

on user clicks update:
  Install.Queue (reason:"update", caveId:…) -> Install.Perform

on user clicks uninstall:
  Uninstall.Perform(caveId)

on shutdown:
  close the TCP connection; --destiny-pid will reap the daemon
```

## Things to know going in

A handful of practical details that aren't obvious from the spec but will
bite you if you skip them:

  * **Notifications can arrive after the call resolves.** Don't tear down
    your conversation handlers the instant the response comes back; drain a
    bit further. The lifetime of notification handlers is tied to the
    surrounding call, not the response message.
  * **Errors come back with structured data.** A failed call's error object
    includes a `data.apiError` with `statusCode` and a `messages` array of
    human-readable strings. Show those messages directly in your UI;
    they're already user-facing.
  * **Be mindful of butler updates.** The itch.io API can change over time and
    you should ensure you are using a modern version of butler. Don't vendor a
    butler binary in your app with no intent to ever updated it.
  * **butlerd is single-tenant.** It expects one client. If your launcher has
    multiple windows or worker processes, multiplex inside your launcher and
    keep one connection (or one daemon) for the whole app. Don't spawn
    multiple daemons against the same `--dbpath`.
  * **Cancellation is cooperative.** Long-running calls accept an `id` you
    generate; passing the same `id` to the matching `*.Cancel` method aborts
    the operation.

## Where to go from here

  * **The full butlerd specification**, with field-level documentation for
    every method, every type, and every notification:
    <https://itchio.github.io/butler/butlerd/>
  * **The TypeScript client library** for JavaScript/TypeScript launchers:
    <https://www.npmjs.com/package/@itchio/butlerd>
  * **The itch.io app** is the largest open-source consumer of butlerd and a
    useful reference for hard cases (download driver, launch UI, manifest
    handling): <https://github.com/itchio/itch>

## Appendix: a simple example in bash

To make the wire protocol concrete, here's a hands-on shell session that
spawns butlerd, authenticates, and calls `Version.Get`. Every step is
something you run yourself, so you can see exactly what goes over the
wire. You only need `butler` and `nc` (netcat).

**1. Start butlerd in the background**, with its JSON output going to a log
file you can read:

```bash
butler daemon --json --transport tcp --keep-alive \
    --dbpath ./butler-demo.db --address https://itch.io \
    --user-agent "butlerd-demo/0.1" > butlerd.log &
```

bash prints the background job's PID, e.g. `[1] 23847`. Note that number;
you'll use it to kill the daemon at the end.

**2. Read the listen notification** out of the log file:

```bash
cat butlerd.log
```

In the long you will see the listen notification with the secret and the TCP address:

```json
{"type":"butlerd/listen-notification","secret":"abc-def-…","tcp":{"address":"127.0.0.1:54321"}}
```

**3. Open a TCP connection with netcat:**

Use the address from the listen notification to connect to butlerd:

```bash
nc 127.0.0.1 54321
```

netcat now sits there waiting for input. Paste the authenticate request
on a single line, substituting the secret you copied:

```json
{"jsonrpc":"2.0","id":1,"method":"Meta.Authenticate","params":{"secret":"abc-def-…"}}
```

butlerd replies on the next line:

```json
{"jsonrpc":"2.0","id":1,"result":{"ok":true}}
```

Now send a real call:

```json
{"jsonrpc":"2.0","id":2,"method":"Version.Get","params":{}}
```

```json
{"jsonrpc":"2.0","id":2,"result":{"version":"15.21.0","versionString":"…"}}
```

Press Ctrl-C to close the netcat session.

**4. Kill the daemon** using the PID from step 1:

```bash
kill 23847
```

A few things to notice:

  * Every JSON-RPC message is one line of JSON terminated by a newline.
    No headers, no other framing.
  * `Meta.Authenticate` must be the **first** request on the connection.
    If you start with `Version.Get`, the daemon rejects it.
  * Anything more involved than this (long-running calls that emit
    notifications, interactive flows where butlerd sends requests back to
    you mid-call) needs a client that dispatches messages by id and
    method. netcat is fine for poking at the protocol by hand; for a real
    launcher you'll want a socket library in your language of choice.
