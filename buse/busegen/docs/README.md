# Caution: this is a draft

!> This document is a draft! It should not be used yet for implementing
   clients for buse. The API and recommendations are still subject to change.

# Overview

buse is butler's JSON-RPC 2.0 service

## Starting the service

To start butler service, run:

```bash
butler service --json
```

The output will be a single line of JSON:

```json
{"time":1519235834,"type":"result","value":{"address":"127.0.0.1:52919","type":"server-listening"}}
```

Contrary to most JSON-RPC services, it's not recommended to keep a single
instance of butler running and make all requests to it (like a server).

Instead, start a new butler instance for each individual task you want to
achieve, like logging in, performing a search, or cleaning downloads.

## Transport

Requests, results, and notifications are sent over TCP, separated by
a newline (`\n`) character.

The format of each line conforms to the
[JSON-RPC 2.0 Specification](http://www.jsonrpc.org/specification),
with the following exceptions:

  * Request `id`s are always numbers
  * Batch requests are not supported

### Why TCP?

We need a connection where either peer can send any number of
messages to the other.

HTTP 1.x implementations of JSON-RPC 2.0 typically allow only
one request/reply, and HTTP 2.0, while awesome, seemed like
overkill for a protocol that is typically used for IPC.

## Updating

Clients are responsible for regularly checking for butler updates, and
installing them.

### HTTP endpoints

Use the following HTTP endpoint to check for a newer version:

  * <https://dl.itch.ovh/butler/windows-amd64/LATEST>

Where `windows-amd64` is one of:

  * `windows-386` - 32-bit Windows
  * `windows-amd64` - 64-bit Windows
  * `linux-amd64` - 64-bit Linux
  * `darwin-amd64` - 64-bit macOS

`LATEST` is a text file that contains a version number.

For example, if the contents of `LATEST` is `v11.1.0`, then
the latest version of butler can be downloaded via:

  * <https://dl.itch.ovh/butler/windows-amd64/v11.1.0/butler.gz>

For the `windows` platform, `butler.gz` should be decompressed to `butler.exe`.
On other platforms, it should be decompressed to just `butler`, and the
executable bit needs to be set.

### Friendly update deployment

See <https://github.com/itchio/itch/issues/1721>

## Requests

Requests are essentially procedure calls: they're made asynchronously, and
a result is sent asynchronously. They may also fail, in which case
you get an error back, with details.

Some requests may complete almost instantly, and have an empty result
Still, waiting for the result lets you know that the peer has received
the request and processed it successfully.

Some requests are made by the client to butler (like CheckUpdate),
others are made from butler to the client (like AllowSandboxSetup)

## Notifications

Notifications are messages that can be sent at any time, in any direction.

There is no way to check that a notification was delivered, only that it was
sent (but the other peer may fail to process it before it exits).


# Messages


## Utilities

### Version.Get <em class="request">Request</em>
<p class='tags'>
<em>Offline</em>
</p>

Retrieves the version of the butler instance the client
is connected to.

This endpoint is meant to gather information when reporting
issues, rather than feature sniffing. Conforming clients should
automatically download new versions of butler, see [Updating](#updating).



Result: 

  * `version` [`string`](#string-type)  
    Something short, like `v8.0.0`
  * `versionString` [`string`](#string-type)  
    Something long, like `v8.0.0, built on Aug 27 2017 @ 01:13:55, ref d833cc0aeea81c236c81dffb27bc18b2b8d8b290`


## Miscellaneous

### InstallResult <em class="type">Type</em>




Fields: 

  * `game` [`itchio.Game`](#itchiogame-type)  
    *undocumented*
  * `upload` [`itchio.Upload`](#itchioupload-type)  
    *undocumented*
  * `build` [`itchio.Build`](#itchiobuild-type)  
    *undocumented*

### Log <em class="notification">Notification</em>

Sent any time butler needs to send a log message. The client should
relay them in their own stdout / stderr, and collect them so they
can be part of an issue report if something goes wrong.

Log


Payload: 

  * `level` [`string`](#string-type)  
    *undocumented*
  * `message` [`string`](#string-type)  
    *undocumented*


## Install

### Game.FindUploads <em class="request">Request</em>

Finds uploads compatible with the current runtime, for a given game



Parameters: 

  * `game` [`itchio.Game`](#itchiogame-type)  
    Which game to find uploads for
  * `credentials` [`GameCredentials`](#gamecredentials-type)  
    The credentials to use to list uploads


Result: 

  * `uploads` [`itchio.Upload[]`](#itchioupload[]-type)  
    A list of uploads that were found to be compatible.

### Operation.Start <em class="request">Request</em>
<p class='tags'>
<em>Cancellable</em>
</p>

Start a new operation (installing or uninstalling).



Parameters: 

  * `id` [`string`](#string-type)  
    A UUID, generated by the client, used for referring to the task when cancelling it, for instance.
  * `stagingFolder` [`string`](#string-type)  
    A folder that butler can use to store temporary files, like partial downloads, checkpoint files, etc.
  * `operation` [`Operation`](#operation-type)  
    Which operation to perform
  * `installParams` [`InstallParams`](#installparams-type)  
    Must be set if Operation is `install`
  * `uninstallParams` [`UninstallParams`](#uninstallparams-type)  
    Must be set if Operation is `uninstall`

### Operation.Cancel <em class="request">Request</em>

Attempt to gracefully cancel an ongoing operation.



Parameters: 

  * `id` [`string`](#string-type)  
    The UUID of the task to cancel, as passed to [Operation.Start](#operationstart-request)

### InstallParams <em class="type">Type</em>

InstallParams contains all the parameters needed to perform
an installation for a game.


Fields: 

  * `game` [`itchio.Game`](#itchiogame-type)  
    Which game to install
  * `installFolder` [`string`](#string-type)  
    An absolute path where to install the game
  * `upload` [`itchio.Upload`](#itchioupload-type)  
    Which upload to install @optional
  * `build` [`itchio.Build`](#itchiobuild-type)  
    Which build to install @optional
  * `credentials` [`GameCredentials`](#gamecredentials-type)  
    Which credentials to use to install the game
  * `ignoreInstallers` [`boolean`](#boolean-type)  
    If true, do not run windows installers, just extract whatever to the install folder. @optional

### UninstallParams <em class="type">Type</em>

UninstallParams contains all the parameters needed to perform
an uninstallation for a game.


Fields: 

  * `installFolder` [`string`](#string-type)  
    Absolute path of the folder butler should uninstall

### PickUpload <em class="request">Request</em>
<p class='tags'>
<em>Dialog</em>
</p>

Asks the user to pick between multiple available uploads



Parameters: 

  * `uploads` [`itchio.Upload[]`](#itchioupload[]-type)  
    An array of upload objects to choose from


Result: 

  * `index` [`number`](#number-type)  
    The index (in the original array) of the upload that was picked, or a negative value to cancel.

### GetReceipt <em class="request">Request</em>
<p class='tags'>
<em>Deprecated</em>
</p>

Retrieves existing receipt information for an install



Result: 

  * `receipt` [`bfs.Receipt`](#bfsreceipt-type)  
    *undocumented*

### Operation.Progress <em class="notification">Notification</em>

Sent periodically to inform on the current state an operation.



Payload: 

  * `progress` [`number`](#number-type)  
    An overall progress value between 0 and 1
  * `eta` [`number`](#number-type)  
    Estimated completion time for the operation, in seconds (floating)
  * `bps` [`number`](#number-type)  
    Network bandwidth used, in bytes per second (floating)

### TaskStarted <em class="notification">Notification</em>

Each operation is made up of one or more tasks. This notification
is sent whenever a task starts for an operation.



Payload: 

  * `reason` [`TaskReason`](#taskreason-type)  
    Why this task was started
  * `type` [`TaskType`](#tasktype-type)  
    Is this task a download? An install?
  * `game` [`itchio.Game`](#itchiogame-type)  
    The game this task is dealing with
  * `upload` [`itchio.Upload`](#itchioupload-type)  
    The upload this task is dealing with
  * `build` [`itchio.Build`](#itchiobuild-type)  
    The build this task is dealing with (if any)
  * `totalSize` [`number`](#number-type)  
    Total size in bytes

### TaskSucceeded <em class="notification">Notification</em>

Sent whenever a task succeeds for an operation.



Payload: 

  * `type` [`TaskType`](#tasktype-type)  
    *undocumented*
  * `installResult` [`InstallResult`](#installresult-type)  
    If the task installed something, then this contains info about the game, upload, build that were installed


## General

### GameCredentials <em class="type">Type</em>

GameCredentials contains all the credentials required to make API requests
including the download key if any


Fields: 

  * `server` [`string`](#string-type)  
    Defaults to `https://itch.io`
  * `apiKey` [`string`](#string-type)  
    A valid itch.io API key
  * `downloadKey` [`number`](#number-type)  
    A download key identifier, or 0 if no download key is available


## Update

### CheckUpdate <em class="request">Request</em>

Looks for one or more game updates.



Parameters: 

  * `items` [`CheckUpdateItem[]`](#checkupdateitem[]-type)  
    A list of items, each of it will be checked for updates


Result: 

  * `updates` [`GameUpdate[]`](#gameupdate[]-type)  
    Any updates found (might be empty)
  * `warnings` [`string[]`](#string[]-type)  
    Warnings messages logged while looking for updates

### CheckUpdateItem <em class="type">Type</em>




Fields: 

  * `itemId` [`string`](#string-type)  
    An UUID generated by the client, which allows it to map back the results to its own items.
  * `installedAt` [`string`](#string-type)  
    Timestamp of the last successful install operation
  * `game` [`itchio.Game`](#itchiogame-type)  
    Game for which to look for an update
  * `upload` [`itchio.Upload`](#itchioupload-type)  
    Currently installed upload
  * `build` [`itchio.Build`](#itchiobuild-type)  
    Currently installed build
  * `credentials` [`GameCredentials`](#gamecredentials-type)  
    Credentials to use to list uploads

### GameUpdateAvailable <em class="notification">Notification</em>
<p class='tags'>
<em>Optional</em>
</p>

Sent while CheckUpdate is still running, every time butler
finds an update for a game. Can be safely ignored if displaying
updates as they are found is not a requirement for the client.



Payload: 

  * `update` [`GameUpdate`](#gameupdate-type)  
    *undocumented*

### GameUpdate <em class="type">Type</em>

Describes an available update for a particular game install.



Fields: 

  * `itemId` [`string`](#string-type)  
    Identifier originally passed in CheckUpdateItem
  * `game` [`itchio.Game`](#itchiogame-type)  
    Game we found an update for
  * `upload` [`itchio.Upload`](#itchioupload-type)  
    Upload to be installed
  * `build` [`itchio.Build`](#itchiobuild-type)  
    Build to be installed (may be nil)


## Launch

### Launch <em class="request">Request</em>




Parameters: 

  * `installFolder` [`string`](#string-type)  
    *undocumented*
  * `game` [`itchio.Game`](#itchiogame-type)  
    *undocumented*
  * `upload` [`itchio.Upload`](#itchioupload-type)  
    *undocumented*
  * `build` [`itchio.Build`](#itchiobuild-type)  
    *undocumented*
  * `verdict` [`configurator.Verdict`](#configuratorverdict-type)  
    *undocumented*
  * `prereqsDir` [`string`](#string-type)  
    *undocumented*
  * `forcePrereqs` [`boolean`](#boolean-type)  
    *undocumented*
  * `sandbox` [`boolean`](#boolean-type)  
    *undocumented*
  * `credentials` [`GameCredentials`](#gamecredentials-type)  
    Used for subkeying

### LaunchRunning <em class="notification">Notification</em>

Sent when the game is configured, prerequisites are installed
sandbox is set up (if enabled), and the game is actually running.


### LaunchExited <em class="notification">Notification</em>

Sent when the game has actually exited.


### PickManifestAction <em class="request">Request</em>
<p class='tags'>
<em>Dialogs</em>
</p>

Pick a manifest action to launch, see [itch app manifests](https://itch.io/docs/itch/integrating/manifest.html)



Parameters: 

  * `actions` [`manifest.Action[]`](#manifestaction[]-type)  
    *undocumented*


Result: 

  * `name` [`string`](#string-type)  
    *undocumented*

### ShellLaunch <em class="request">Request</em>

Ask the client to perform a shell launch, ie. open an item
with the operating system's default handler (File explorer)



Parameters: 

  * `itemPath` [`string`](#string-type)  
    *undocumented*

### HTMLLaunch <em class="request">Request</em>

Ask the client to perform an HTML launch, ie. open an HTML5
game, ideally in an embedded browser.



Parameters: 

  * `rootFolder` [`string`](#string-type)  
    *undocumented*
  * `indexPath` [`string`](#string-type)  
    *undocumented*
  * `args` [`string[]`](#string[]-type)  
    *undocumented*
  * `env` [`Map<string, string>`](#map<string, string>-type)  
    *undocumented*

### URLLaunch <em class="request">Request</em>

Ask the client to perform an URL launch, ie. open an address
with the system browser or appropriate.



Parameters: 

  * `url` [`string`](#string-type)  
    *undocumented*

### SaveVerdict <em class="request">Request</em>
<p class='tags'>
<em>Deprecated</em>
</p>

Ask the client to save verdict information after a reconfiguration.



Parameters: 

  * `verdict` [`configurator.Verdict`](#configuratorverdict-type)  
    *undocumented*

### AllowSandboxSetup <em class="request">Request</em>
<p class='tags'>
<em>Dialogs</em>
</p>

Ask the user to allow sandbox setup. Will be followed by
a UAC prompt (on Windows) or a pkexec dialog (on Linux) if
the user allows.



Result: 

  * `allow` [`boolean`](#boolean-type)  
    *undocumented*

### PrereqsStarted <em class="notification">Notification</em>

Sent when some prerequisites are about to be installed.



Payload: 

  * `tasks` [`Map<string, PrereqTask>`](#map<string, prereqtask>-type)  
    *undocumented*

### PrereqTask <em class="type">Type</em>




Fields: 

  * `fullName` [`string`](#string-type)  
    *undocumented*
  * `order` [`int`](#int-type)  
    *undocumented*

### PrereqsTaskState <em class="notification">Notification</em>




Payload: 

  * `name` [`string`](#string-type)  
    *undocumented*
  * `status` [`PrereqStatus`](#prereqstatus-type)  
    *undocumented*
  * `progress` [`number`](#number-type)  
    *undocumented*
  * `eta` [`number`](#number-type)  
    *undocumented*
  * `bps` [`number`](#number-type)  
    *undocumented*

### PrereqsEnded <em class="notification">Notification</em>



### PrereqsFailed <em class="request">Request</em>




Parameters: 

  * `error` [`string`](#string-type)  
    *undocumented*
  * `errorStack` [`string`](#string-type)  
    *undocumented*


Result: 

  * `continue` [`boolean`](#boolean-type)  
    *undocumented*


## Clean Downloads

### CleanDownloads.Search <em class="request">Request</em>

Look for folders we can clean up in various download folders.
This finds anything that doesn't correspond to any current downloads
we know about.



Parameters: 

  * `roots` [`string[]`](#string[]-type)  
    A list of folders to scan for potential subfolders to clean up
  * `whitelist` [`string[]`](#string[]-type)  
    A list of subfolders to not consider when cleaning (staging folders for in-progress downloads)


Result: 

  * `entries` [`CleanDownloadsEntry[]`](#cleandownloadsentry[]-type)  
    *undocumented*

### CleanDownloadsEntry <em class="type">Type</em>




Fields: 

  * `path` [`string`](#string-type)  
    The complete path of the file or folder we intend to remove
  * `size` [`number`](#number-type)  
    The size of the folder or file, in bytes

### CleanDownloads.Apply <em class="request">Request</em>

Remove the specified entries from disk, freeing up disk space.



Parameters: 

  * `entries` [`CleanDownloadsEntry[]`](#cleandownloadsentry[]-type)  
    *undocumented*


## Test

### Test.DoubleTwice <em class="type">Type</em>




Fields: 

  * `number` [`number`](#number-type)  
    *undocumented*

### Test.Double <em class="type">Type</em>




Fields: 

  * `number` [`number`](#number-type)  
    *undocumented*


