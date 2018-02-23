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

### <em class="request-client-caller"></em>Version.Get
<p class='tags'>
<em>Client request</em>
<em>Offline</em>
</p>

Retrieves the version of the butler instance the client
is connected to.

This endpoint is meant to gather information when reporting
issues, rather than feature sniffing. Conforming clients should
automatically download new versions of butler, see [Updating](#updating).



**Parameters**: _none_


**Result**: 

Name | Type | Description
--- | --- | ---
`version` | [`string`](#string-type) | Something short, like `v8.0.0`
`versionString` | [`string`](#string-type) | Something long, like `v8.0.0, built on Aug 27 2017 @ 01:13:55, ref d833cc0aeea81c236c81dffb27bc18b2b8d8b290`


## Launch

### <em class="request-client-caller"></em>Launch
<p class='tags'>
<em>Client request</em>
</p>




**Parameters**: 

Name | Type | Description
--- | --- | ---
`installFolder` | [`string`](#string-type) | *undocumented*
`game` | [`itchio.Game`](#itchiogame-type) | *undocumented*
`upload` | [`itchio.Upload`](#itchioupload-type) | *undocumented*
`build` | [`itchio.Build`](#itchiobuild-type) | *undocumented*
`verdict` | [`configurator.Verdict`](#configuratorverdict-type) | *undocumented*
`prereqsDir` | [`string`](#string-type) | *undocumented*
`forcePrereqs` | [`boolean`](#boolean-type) | *undocumented*
`sandbox` | [`boolean`](#boolean-type) | *undocumented*
`credentials` | [`GameCredentials`](#gamecredentials-type) | Used for subkeying


**Result**: _none_

### <em class="notification"></em>LaunchRunning
<p class='tags'>
<em>Notification</em>
</p>

Sent when the game is configured, prerequisites are installed
sandbox is set up (if enabled), and the game is actually running.



**Payload**: _none_

### <em class="notification"></em>LaunchExited
<p class='tags'>
<em>Notification</em>
</p>

Sent when the game has actually exited.



**Payload**: _none_

### <em class="request-server-caller"></em>PickManifestAction
<p class='tags'>
<em>Server request</em>
<em>Dialogs</em>
</p>

Pick a manifest action to launch, see [itch app manifests](https://itch.io/docs/itch/integrating/manifest.html)



**Parameters**: 

Name | Type | Description
--- | --- | ---
`actions` | [`manifest.Action[]`](#manifestaction[]-type) | *undocumented*


**Result**: 

Name | Type | Description
--- | --- | ---
`name` | [`string`](#string-type) | *undocumented*

### <em class="request-server-caller"></em>ShellLaunch
<p class='tags'>
<em>Server request</em>
</p>

Ask the client to perform a shell launch, ie. open an item
with the operating system's default handler (File explorer)



**Parameters**: 

Name | Type | Description
--- | --- | ---
`itemPath` | [`string`](#string-type) | *undocumented*


**Result**: _none_

### <em class="request-server-caller"></em>HTMLLaunch
<p class='tags'>
<em>Server request</em>
</p>

Ask the client to perform an HTML launch, ie. open an HTML5
game, ideally in an embedded browser.



**Parameters**: 

Name | Type | Description
--- | --- | ---
`rootFolder` | [`string`](#string-type) | *undocumented*
`indexPath` | [`string`](#string-type) | *undocumented*
`args` | [`string[]`](#string[]-type) | *undocumented*
`env` | [`Map<string, string>`](#map<string, string>-type) | *undocumented*


**Result**: _none_

### <em class="request-server-caller"></em>URLLaunch
<p class='tags'>
<em>Server request</em>
</p>

Ask the client to perform an URL launch, ie. open an address
with the system browser or appropriate.



**Parameters**: 

Name | Type | Description
--- | --- | ---
`url` | [`string`](#string-type) | *undocumented*


**Result**: _none_

### <em class="request-server-caller"></em>SaveVerdict
<p class='tags'>
<em>Server request</em>
<em>Deprecated</em>
</p>

Ask the client to save verdict information after a reconfiguration.



**Parameters**: 

Name | Type | Description
--- | --- | ---
`verdict` | [`configurator.Verdict`](#configuratorverdict-type) | *undocumented*


**Result**: _none_

### <em class="request-server-caller"></em>AllowSandboxSetup
<p class='tags'>
<em>Server request</em>
<em>Dialogs</em>
</p>

Ask the user to allow sandbox setup. Will be followed by
a UAC prompt (on Windows) or a pkexec dialog (on Linux) if
the user allows.



**Parameters**: _none_


**Result**: 

Name | Type | Description
--- | --- | ---
`allow` | [`boolean`](#boolean-type) | *undocumented*

### <em class="notification"></em>PrereqsStarted
<p class='tags'>
<em>Notification</em>
</p>

Sent when some prerequisites are about to be installed.



**Payload**: 

Name | Type | Description
--- | --- | ---
`tasks` | [`Map<string, PrereqTask>`](#map<string, prereqtask>-type) | *undocumented*

### <em class="type"></em>PrereqTask
<p class='tags'>
<em>Type</em>
</p>




**Fields**: 

Name | Type | Description
--- | --- | ---
`fullName` | [`string`](#string-type) | *undocumented*
`order` | [`int`](#int-type) | *undocumented*

### <em class="notification"></em>PrereqsTaskState
<p class='tags'>
<em>Notification</em>
</p>




**Payload**: 

Name | Type | Description
--- | --- | ---
`name` | [`string`](#string-type) | *undocumented*
`status` | [`PrereqStatus`](#prereqstatus-type) | *undocumented*
`progress` | [`number`](#number-type) | *undocumented*
`eta` | [`number`](#number-type) | *undocumented*
`bps` | [`number`](#number-type) | *undocumented*

### <em class="notification"></em>PrereqsEnded
<p class='tags'>
<em>Notification</em>
</p>




**Payload**: _none_

### <em class="request-server-caller"></em>PrereqsFailed
<p class='tags'>
<em>Server request</em>
</p>




**Parameters**: 

Name | Type | Description
--- | --- | ---
`error` | [`string`](#string-type) | *undocumented*
`errorStack` | [`string`](#string-type) | *undocumented*


**Result**: 

Name | Type | Description
--- | --- | ---
`continue` | [`boolean`](#boolean-type) | *undocumented*


## Update

### <em class="request-client-caller"></em>CheckUpdate
<p class='tags'>
<em>Client request</em>
</p>

Looks for one or more game updates.



**Parameters**: 

Name | Type | Description
--- | --- | ---
`items` | [`CheckUpdateItem[]`](#checkupdateitem[]-type) | A list of items, each of it will be checked for updates


**Result**: 

Name | Type | Description
--- | --- | ---
`updates` | [`GameUpdate[]`](#gameupdate[]-type) | Any updates found (might be empty)
`warnings` | [`string[]`](#string[]-type) | Warnings messages logged while looking for updates

### <em class="type"></em>CheckUpdateItem
<p class='tags'>
<em>Type</em>
</p>




**Fields**: 

Name | Type | Description
--- | --- | ---
`itemId` | [`string`](#string-type) | An UUID generated by the client, which allows it to map back the results to its own items.
`installedAt` | [`string`](#string-type) | Timestamp of the last successful install operation
`game` | [`itchio.Game`](#itchiogame-type) | Game for which to look for an update
`upload` | [`itchio.Upload`](#itchioupload-type) | Currently installed upload
`build` | [`itchio.Build`](#itchiobuild-type) | Currently installed build
`credentials` | [`GameCredentials`](#gamecredentials-type) | Credentials to use to list uploads

### <em class="notification"></em>GameUpdateAvailable
<p class='tags'>
<em>Notification</em>
<em>Optional</em>
</p>

Sent while CheckUpdate is still running, every time butler
finds an update for a game. Can be safely ignored if displaying
updates as they are found is not a requirement for the client.



**Payload**: 

Name | Type | Description
--- | --- | ---
`update` | [`GameUpdate`](#gameupdate-type) | *undocumented*

### <em class="type"></em>GameUpdate
<p class='tags'>
<em>Type</em>
</p>

Describes an available update for a particular game install.



**Fields**: 

Name | Type | Description
--- | --- | ---
`itemId` | [`string`](#string-type) | Identifier originally passed in CheckUpdateItem
`game` | [`itchio.Game`](#itchiogame-type) | Game we found an update for
`upload` | [`itchio.Upload`](#itchioupload-type) | Upload to be installed
`build` | [`itchio.Build`](#itchiobuild-type) | Build to be installed (may be nil)


## General

### <em class="type"></em>GameCredentials
<p class='tags'>
<em>Type</em>
</p>

GameCredentials contains all the credentials required to make API requests
including the download key if any


**Fields**: 

Name | Type | Description
--- | --- | ---
`server` | [`string`](#string-type) | Defaults to `https://itch.io`
`apiKey` | [`string`](#string-type) | A valid itch.io API key
`downloadKey` | [`number`](#number-type) | A download key identifier, or 0 if no download key is available


## Install

### <em class="request-client-caller"></em>Game.FindUploads
<p class='tags'>
<em>Client request</em>
</p>

Finds uploads compatible with the current runtime, for a given game



**Parameters**: 

Name | Type | Description
--- | --- | ---
`game` | [`itchio.Game`](#itchiogame-type) | Which game to find uploads for
`credentials` | [`GameCredentials`](#gamecredentials-type) | The credentials to use to list uploads


**Result**: 

Name | Type | Description
--- | --- | ---
`uploads` | [`itchio.Upload[]`](#itchioupload[]-type) | A list of uploads that were found to be compatible.

### <em class="request-client-caller"></em>Operation.Start
<p class='tags'>
<em>Client request</em>
<em>Cancellable</em>
</p>

Start a new operation (installing or uninstalling).



**Parameters**: 

Name | Type | Description
--- | --- | ---
`id` | [`string`](#string-type) | A UUID, generated by the client, used for referring to the task when cancelling it, for instance.
`stagingFolder` | [`string`](#string-type) | A folder that butler can use to store temporary files, like partial downloads, checkpoint files, etc.
`operation` | [`Operation`](#operation-type) | Which operation to perform
`installParams` | [`InstallParams`](#installparams-type) | Must be set if Operation is `install`
`uninstallParams` | [`UninstallParams`](#uninstallparams-type) | Must be set if Operation is `uninstall`


**Result**: _none_

### <em class="request-client-caller"></em>Operation.Cancel
<p class='tags'>
<em>Client request</em>
</p>

Attempt to gracefully cancel an ongoing operation.



**Parameters**: 

Name | Type | Description
--- | --- | ---
`id` | [`string`](#string-type) | The UUID of the task to cancel, as passed to [Operation.Start](#operationstart-request)


**Result**: _none_

### <em class="type"></em>InstallParams
<p class='tags'>
<em>Type</em>
</p>

InstallParams contains all the parameters needed to perform
an installation for a game.


**Fields**: 

Name | Type | Description
--- | --- | ---
`game` | [`itchio.Game`](#itchiogame-type) | Which game to install
`installFolder` | [`string`](#string-type) | An absolute path where to install the game
`upload` | [`itchio.Upload`](#itchioupload-type) | Which upload to install @optional
`build` | [`itchio.Build`](#itchiobuild-type) | Which build to install @optional
`credentials` | [`GameCredentials`](#gamecredentials-type) | Which credentials to use to install the game
`ignoreInstallers` | [`boolean`](#boolean-type) | If true, do not run windows installers, just extract whatever to the install folder. @optional

### <em class="type"></em>UninstallParams
<p class='tags'>
<em>Type</em>
</p>

UninstallParams contains all the parameters needed to perform
an uninstallation for a game.


**Fields**: 

Name | Type | Description
--- | --- | ---
`installFolder` | [`string`](#string-type) | Absolute path of the folder butler should uninstall

### <em class="request-server-caller"></em>PickUpload
<p class='tags'>
<em>Server request</em>
<em>Dialog</em>
</p>

Asks the user to pick between multiple available uploads



**Parameters**: 

Name | Type | Description
--- | --- | ---
`uploads` | [`itchio.Upload[]`](#itchioupload[]-type) | An array of upload objects to choose from


**Result**: 

Name | Type | Description
--- | --- | ---
`index` | [`number`](#number-type) | The index (in the original array) of the upload that was picked, or a negative value to cancel.

### <em class="request-server-caller"></em>GetReceipt
<p class='tags'>
<em>Server request</em>
<em>Deprecated</em>
</p>

Retrieves existing receipt information for an install



**Parameters**: _none_


**Result**: 

Name | Type | Description
--- | --- | ---
`receipt` | [`bfs.Receipt`](#bfsreceipt-type) | *undocumented*

### <em class="notification"></em>Operation.Progress
<p class='tags'>
<em>Notification</em>
</p>

Sent periodically to inform on the current state an operation.



**Payload**: 

Name | Type | Description
--- | --- | ---
`progress` | [`number`](#number-type) | An overall progress value between 0 and 1
`eta` | [`number`](#number-type) | Estimated completion time for the operation, in seconds (floating)
`bps` | [`number`](#number-type) | Network bandwidth used, in bytes per second (floating)

### <em class="notification"></em>TaskStarted
<p class='tags'>
<em>Notification</em>
</p>

Each operation is made up of one or more tasks. This notification
is sent whenever a task starts for an operation.



**Payload**: 

Name | Type | Description
--- | --- | ---
`reason` | [`TaskReason`](#taskreason-type) | Why this task was started
`type` | [`TaskType`](#tasktype-type) | Is this task a download? An install?
`game` | [`itchio.Game`](#itchiogame-type) | The game this task is dealing with
`upload` | [`itchio.Upload`](#itchioupload-type) | The upload this task is dealing with
`build` | [`itchio.Build`](#itchiobuild-type) | The build this task is dealing with (if any)
`totalSize` | [`number`](#number-type) | Total size in bytes

### <em class="notification"></em>TaskSucceeded
<p class='tags'>
<em>Notification</em>
</p>

Sent whenever a task succeeds for an operation.



**Payload**: 

Name | Type | Description
--- | --- | ---
`type` | [`TaskType`](#tasktype-type) | *undocumented*
`installResult` | [`InstallResult`](#installresult-type) | If the task installed something, then this contains info about the game, upload, build that were installed

### <em class="type"></em>InstallResult
<p class='tags'>
<em>Type</em>
</p>




**Fields**: 

Name | Type | Description
--- | --- | ---
`game` | [`itchio.Game`](#itchiogame-type) | *undocumented*
`upload` | [`itchio.Upload`](#itchioupload-type) | *undocumented*
`build` | [`itchio.Build`](#itchiobuild-type) | *undocumented*


## Test

### <em class="request-client-caller"></em>Test.DoubleTwice
<p class='tags'>
<em>Client request</em>
</p>




**Parameters**: 

Name | Type | Description
--- | --- | ---
`number` | [`number`](#number-type) | *undocumented*


**Result**: 

Name | Type | Description
--- | --- | ---
`number` | [`number`](#number-type) | *undocumented*

### <em class="request-server-caller"></em>Test.Double
<p class='tags'>
<em>Server request</em>
</p>




**Parameters**: 

Name | Type | Description
--- | --- | ---
`number` | [`number`](#number-type) | *undocumented*


**Result**: 

Name | Type | Description
--- | --- | ---
`number` | [`number`](#number-type) | *undocumented*


## Clean Downloads

### <em class="request-client-caller"></em>CleanDownloads.Search
<p class='tags'>
<em>Client request</em>
</p>

Look for folders we can clean up in various download folders.
This finds anything that doesn't correspond to any current downloads
we know about.



**Parameters**: 

Name | Type | Description
--- | --- | ---
`roots` | [`string[]`](#string[]-type) | A list of folders to scan for potential subfolders to clean up
`whitelist` | [`string[]`](#string[]-type) | A list of subfolders to not consider when cleaning (staging folders for in-progress downloads)


**Result**: 

Name | Type | Description
--- | --- | ---
`entries` | [`CleanDownloadsEntry[]`](#cleandownloadsentry[]-type) | *undocumented*

### <em class="type"></em>CleanDownloadsEntry
<p class='tags'>
<em>Type</em>
</p>




**Fields**: 

Name | Type | Description
--- | --- | ---
`path` | [`string`](#string-type) | The complete path of the file or folder we intend to remove
`size` | [`number`](#number-type) | The size of the folder or file, in bytes

### <em class="request-client-caller"></em>CleanDownloads.Apply
<p class='tags'>
<em>Client request</em>
</p>

Remove the specified entries from disk, freeing up disk space.



**Parameters**: 

Name | Type | Description
--- | --- | ---
`entries` | [`CleanDownloadsEntry[]`](#cleandownloadsentry[]-type) | *undocumented*


**Result**: _none_


## Miscellaneous

### <em class="notification"></em>Log
<p class='tags'>
<em>Notification</em>
</p>

Sent any time butler needs to send a log message. The client should
relay them in their own stdout / stderr, and collect them so they
can be part of an issue report if something goes wrong.

Log


**Payload**: 

Name | Type | Description
--- | --- | ---
`level` | [`string`](#string-type) | *undocumented*
`message` | [`string`](#string-type) | *undocumented*

### <em class="type"></em>itchio.User
<p class='tags'>
<em>Type</em>
</p>

User represents an itch.io account, with basic profile info


**Fields**: 

Name | Type | Description
--- | --- | ---
`id` | [`number`](#number-type) | Site-wide unique identifier generated by itch.io
`username` | [`string`](#string-type) | The user's username (used for login)
`displayName` | [`string`](#string-type) | The user's display name: human-friendly, may contain spaces, unicode etc.
`developer` | [`boolean`](#boolean-type) | Has the user opted into creating games?
`pressUser` | [`boolean`](#boolean-type) | Is the user part of itch.io's press program?
`url` | [`string`](#string-type) | The address of the user's page on itch.io
`coverUrl` | [`string`](#string-type) | User's avatar, may be a GIF
`stillCoverUrl` | [`string`](#string-type) | Static version of user's avatar, only set if the main cover URL is a GIF

### <em class="type"></em>itchio.Game
<p class='tags'>
<em>Type</em>
</p>

Game represents a page on itch.io, it could be a game,
a tool, a comic, etc.


**Fields**: 

Name | Type | Description
--- | --- | ---
`id` | [`number`](#number-type) | Site-wide unique identifier generated by itch.io
`url` | [`string`](#string-type) | Canonical address of the game's page on itch.io
`title` | [`string`](#string-type) | Human-friendly title (may contain any character)
`shortText` | [`string`](#string-type) | Human-friendly short description
`type` | [`string`](#string-type) | Downloadable game, html game, etc.
`classification` | [`string`](#string-type) | Classification: game, tool, comic, etc.
`coverUrl` | [`string`](#string-type) | Cover url (might be a GIF)
`stillCoverUrl` | [`string`](#string-type) | Non-gif cover url, only set if main cover url is a GIF
`createdAt` | [`string`](#string-type) | Date the game was created
`publishedAt` | [`string`](#string-type) | Date the game was published, empty if not currently published
`minPrice` | [`number`](#number-type) | Price in cents of a dollar
`inPressSystem` | [`boolean`](#boolean-type) | Is this game downloadable by press users for free?
`hasDemo` | [`boolean`](#boolean-type) | Does this game have a demo that can be downloaded for free?
`pOsx` | [`boolean`](#boolean-type) | Does this game have an upload tagged with 'macOS compatible'? (creator-controlled)
`pLinux` | [`boolean`](#boolean-type) | Does this game have an upload tagged with 'Linux compatible'? (creator-controlled)
`pWindows` | [`boolean`](#boolean-type) | Does this game have an upload tagged with 'Windows compatible'? (creator-controlled)
`pAndroid` | [`boolean`](#boolean-type) | Does this game have an upload tagged with 'Android compatible'? (creator-controlled)

### <em class="type"></em>itchio.Upload
<p class='tags'>
<em>Type</em>
</p>

An Upload is a downloadable file. Some are wharf-enabled, which means
they're actually a "channel" that may contain multiple builds, pushed
with <https://github.com/itchio/butler>


**Fields**: 

Name | Type | Description
--- | --- | ---
`id` | [`number`](#number-type) | Site-wide unique identifier generated by itch.io
`filename` | [`string`](#string-type) | Original file name (example: `Overland_x64.zip`)
`displayName` | [`string`](#string-type) | Human-friendly name set by developer (example: `Overland for Windows 64-bit`)
`size` | [`number`](#number-type) | Size of upload in bytes. For wharf-enabled uploads, it's the archive size.
`channelName` | [`string`](#string-type) | Name of the wharf channel for this upload, if it's a wharf-enabled upload
`build` | [`Build`](#build-type) | Latest build for this upload, if it's a wharf-enabled upload
`demo` | [`boolean`](#boolean-type) | Is this upload a demo that can be downloaded for free?
`preorder` | [`boolean`](#boolean-type) | Is this upload a pre-order placeholder?
`type` | [`string`](#string-type) | Upload type: default, soundtrack, etc.
`pOsx` | [`boolean`](#boolean-type) | Is this upload tagged with 'macOS compatible'? (creator-controlled)
`pLinux` | [`boolean`](#boolean-type) | Is this upload tagged with 'Linux compatible'? (creator-controlled)
`pWindows` | [`boolean`](#boolean-type) | Is this upload tagged with 'Windows compatible'? (creator-controlled)
`pAndroid` | [`boolean`](#boolean-type) | Is this upload tagged with 'Android compatible'? (creator-controlled)
`createdAt` | [`string`](#string-type) | Date this upload was created at
`updatedAt` | [`string`](#string-type) | Date this upload was last updated at (order changed, display name set, etc.)

### <em class="type"></em>itchio.Collection
<p class='tags'>
<em>Type</em>
</p>

A Collection is a set of games, curated by humans.


**Fields**: 

Name | Type | Description
--- | --- | ---
`id` | [`number`](#number-type) | Site-wide unique identifier generated by itch.io
`title` | [`string`](#string-type) | Human-friendly title for collection, for example `Couch coop games`
`createdAt` | [`string`](#string-type) | Date this collection was created at
`updatedAt` | [`string`](#string-type) | Date this collection was last updated at (item added, title set, etc.)
`gamesCount` | [`number`](#number-type) | Number of games in the collection. This might not be accurate as some games might not be accessible to whoever is asking (project page deleted, visibility level changed, etc.)

### <em class="type"></em>itchio.DownloadKey
<p class='tags'>
<em>Type</em>
</p>

A download key is often generated when a purchase is made, it
allows downloading uploads for a game that are not available
for free.


**Fields**: 

Name | Type | Description
--- | --- | ---
`id` | [`number`](#number-type) | Site-wide unique identifier generated by itch.io
`gameId` | [`number`](#number-type) | Identifier of the game to which this download key grants access
`game` | [`Game`](#game-type) | *undocumented*
`createdAt` | [`string`](#string-type) | Date this key was created at (often coincides with purchase time)
`updatedAt` | [`string`](#string-type) | Date this key was last updated at
`ownerId` | [`number`](#number-type) | Identifier of the itch.io user to which this key belongs

### <em class="type"></em>itchio.Build
<p class='tags'>
<em>Type</em>
</p>

Build contains information about a specific build


**Fields**: 

Name | Type | Description
--- | --- | ---
`id` | [`number`](#number-type) | Site-wide unique identifier generated by itch.io
`parentBuildId` | [`number`](#number-type) | Identifier of the build before this one on the same channel, or 0 if this is the initial build.
`state` | [`BuildState`](#buildstate-type) | State of the build: started, processing, etc.
`version` | [`number`](#number-type) | Automatically-incremented version number, starting with 1
`userVersion` | [`string`](#string-type) | Value specified by developer with `--userversion` when pushing a build Might not be unique across builds of a given channel.
`files` | [`BuildFile[]`](#buildfile[]-type) | Files associated with this build - often at least an archive, a signature, and a patch. Some might be missing while the build is still processing or if processing has failed.
`user` | [`User`](#user-type) | *undocumented*
`createdAt` | [`string`](#string-type) | *undocumented*
`updatedAt` | [`string`](#string-type) | *undocumented*


