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

<p>
<p>Retrieves the version of the butler instance the client
is connected to.</p>

<p>This endpoint is meant to gather information when reporting
issues, rather than feature sniffing. Conforming clients should
automatically download new versions of butler, see <a href="#updating">Updating</a>.</p>

</p>

<p>
<strong>Parameters</strong>: <em>none</em>
</p>



<p>
<strong>Result</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>version</code></td>
<td><code class="typename">string</code></td>
<td><p>Something short, like <code>v8.0.0</code></p>
</td>
</tr>
<tr>
<td><code>versionString</code></td>
<td><code class="typename">string</code></td>
<td><p>Something long, like <code>v8.0.0, built on Aug 27 2017 @ 01:13:55, ref d833cc0aeea81c236c81dffb27bc18b2b8d8b290</code></p>
</td>
</tr>
</table>


## Launch

### <em class="request-client-caller"></em>Launch

<p class='tags'>
<em>Client request</em>
</p>


<p>
<strong>Parameters</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>installFolder</code></td>
<td><code class="typename">string</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#Upload__TypeHint">Upload</span></code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#Build__TypeHint">Build</span></code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>verdict</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#Verdict__TypeHint">Verdict</span></code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>prereqsDir</code></td>
<td><code class="typename">string</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>forcePrereqs</code></td>
<td><code class="typename">boolean</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>sandbox</code></td>
<td><code class="typename">boolean</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>credentials</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#GameCredentials__TypeHint">GameCredentials</span></code></td>
<td><p>Used for subkeying</p>
</td>
</tr>
</table>



<p>
<strong>Result</strong>: <em>none</em>
</p>

### <em class="notification"></em>LaunchRunning

<p class='tags'>
<em>Notification</em>
</p>

<p>
<p>Sent when the game is configured, prerequisites are installed
sandbox is set up (if enabled), and the game is actually running.</p>

</p>

<p>
<strong>Payload</strong>: <em>none</em>
</p>

### <em class="notification"></em>LaunchExited

<p class='tags'>
<em>Notification</em>
</p>

<p>
<p>Sent when the game has actually exited.</p>

</p>

<p>
<strong>Payload</strong>: <em>none</em>
</p>

### <em class="request-server-caller"></em>PickManifestAction

<p class='tags'>
<em>Server request</em>
<em>Dialogs</em>
</p>

<p>
<p>Pick a manifest action to launch, see <a href="https://itch.io/docs/itch/integrating/manifest.html">itch app manifests</a></p>

</p>

<p>
<strong>Parameters</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>actions</code></td>
<td><code class="typename">Action[]</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
</table>



<p>
<strong>Result</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>name</code></td>
<td><code class="typename">string</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
</table>

### <em class="request-server-caller"></em>ShellLaunch

<p class='tags'>
<em>Server request</em>
</p>

<p>
<p>Ask the client to perform a shell launch, ie. open an item
with the operating system&rsquo;s default handler (File explorer)</p>

</p>

<p>
<strong>Parameters</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>itemPath</code></td>
<td><code class="typename">string</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
</table>



<p>
<strong>Result</strong>: <em>none</em>
</p>

### <em class="request-server-caller"></em>HTMLLaunch

<p class='tags'>
<em>Server request</em>
</p>

<p>
<p>Ask the client to perform an HTML launch, ie. open an HTML5
game, ideally in an embedded browser.</p>

</p>

<p>
<strong>Parameters</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>rootFolder</code></td>
<td><code class="typename">string</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>indexPath</code></td>
<td><code class="typename">string</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>args</code></td>
<td><code class="typename">string[]</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>env</code></td>
<td><code class="typename">Map<string, string></code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
</table>



<p>
<strong>Result</strong>: <em>none</em>
</p>

### <em class="request-server-caller"></em>URLLaunch

<p class='tags'>
<em>Server request</em>
</p>

<p>
<p>Ask the client to perform an URL launch, ie. open an address
with the system browser or appropriate.</p>

</p>

<p>
<strong>Parameters</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>url</code></td>
<td><code class="typename">string</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
</table>



<p>
<strong>Result</strong>: <em>none</em>
</p>

### <em class="request-server-caller"></em>SaveVerdict

<p class='tags'>
<em>Server request</em>
<em>Deprecated</em>
</p>

<p>
<p>Ask the client to save verdict information after a reconfiguration.</p>

</p>

<p>
<strong>Parameters</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>verdict</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#Verdict__TypeHint">Verdict</span></code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
</table>



<p>
<strong>Result</strong>: <em>none</em>
</p>

### <em class="request-server-caller"></em>AllowSandboxSetup

<p class='tags'>
<em>Server request</em>
<em>Dialogs</em>
</p>

<p>
<p>Ask the user to allow sandbox setup. Will be followed by
a UAC prompt (on Windows) or a pkexec dialog (on Linux) if
the user allows.</p>

</p>

<p>
<strong>Parameters</strong>: <em>none</em>
</p>



<p>
<strong>Result</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>allow</code></td>
<td><code class="typename">boolean</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
</table>

### <em class="notification"></em>PrereqsStarted

<p class='tags'>
<em>Notification</em>
</p>

<p>
<p>Sent when some prerequisites are about to be installed.</p>

</p>

<p>
<strong>Payload</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>tasks</code></td>
<td><code class="typename">Map<string, PrereqTask></code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
</table>

### <em class="type"></em>PrereqTask

<p class='tags'>
<em>Type</em>
</p>


<p>
<strong>Fields</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>fullName</code></td>
<td><code class="typename">string</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>order</code></td>
<td><code class="typename">int</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
</table>


<div id="PrereqTask__TypeHint" style="display: none;" class="tip-content">
<p>PrereqTask <a href="#/?id=prereqtask">(Go to definition)</a></p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
</tr>
<tr>
<td><code>fullName</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>order</code></td>
<td><code class="typename">int</code></td>
</tr>
</table>

</div>

### <em class="notification"></em>PrereqsTaskState

<p class='tags'>
<em>Notification</em>
</p>


<p>
<strong>Payload</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>name</code></td>
<td><code class="typename">string</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>status</code></td>
<td><code class="typename">PrereqStatus</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>progress</code></td>
<td><code class="typename">number</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>eta</code></td>
<td><code class="typename">number</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>bps</code></td>
<td><code class="typename">number</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
</table>

### <em class="notification"></em>PrereqsEnded

<p class='tags'>
<em>Notification</em>
</p>


<p>
<strong>Payload</strong>: <em>none</em>
</p>

### <em class="request-server-caller"></em>PrereqsFailed

<p class='tags'>
<em>Server request</em>
</p>


<p>
<strong>Parameters</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>error</code></td>
<td><code class="typename">string</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>errorStack</code></td>
<td><code class="typename">string</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
</table>



<p>
<strong>Result</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>continue</code></td>
<td><code class="typename">boolean</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
</table>


## Update

### <em class="request-client-caller"></em>CheckUpdate

<p class='tags'>
<em>Client request</em>
</p>

<p>
<p>Looks for one or more game updates.</p>

</p>

<p>
<strong>Parameters</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>items</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#CheckUpdateItem__TypeHint">CheckUpdateItem</span>[]</code></td>
<td><p>A list of items, each of it will be checked for updates</p>
</td>
</tr>
</table>



<p>
<strong>Result</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>updates</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#GameUpdate__TypeHint">GameUpdate</span>[]</code></td>
<td><p>Any updates found (might be empty)</p>
</td>
</tr>
<tr>
<td><code>warnings</code></td>
<td><code class="typename">string[]</code></td>
<td><p>Warnings messages logged while looking for updates</p>
</td>
</tr>
</table>

### <em class="type"></em>CheckUpdateItem

<p class='tags'>
<em>Type</em>
</p>


<p>
<strong>Fields</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>itemId</code></td>
<td><code class="typename">string</code></td>
<td><p>An UUID generated by the client, which allows it to map back the results to its own items.</p>
</td>
</tr>
<tr>
<td><code>installedAt</code></td>
<td><code class="typename">string</code></td>
<td><p>Timestamp of the last successful install operation</p>
</td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td><p>Game for which to look for an update</p>
</td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#Upload__TypeHint">Upload</span></code></td>
<td><p>Currently installed upload</p>
</td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#Build__TypeHint">Build</span></code></td>
<td><p>Currently installed build</p>
</td>
</tr>
<tr>
<td><code>credentials</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#GameCredentials__TypeHint">GameCredentials</span></code></td>
<td><p>Credentials to use to list uploads</p>
</td>
</tr>
</table>


<div id="CheckUpdateItem__TypeHint" style="display: none;" class="tip-content">
<p>CheckUpdateItem <a href="#/?id=checkupdateitem">(Go to definition)</a></p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
</tr>
<tr>
<td><code>itemId</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>installedAt</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="struct-type">Game</span></code></td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="struct-type">Upload</span></code></td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="struct-type">Build</span></code></td>
</tr>
<tr>
<td><code>credentials</code></td>
<td><code class="typename"><span class="struct-type">GameCredentials</span></code></td>
</tr>
</table>

</div>

### <em class="notification"></em>GameUpdateAvailable

<p class='tags'>
<em>Notification</em>
<em>Optional</em>
</p>

<p>
<p>Sent while CheckUpdate is still running, every time butler
finds an update for a game. Can be safely ignored if displaying
updates as they are found is not a requirement for the client.</p>

</p>

<p>
<strong>Payload</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>update</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#GameUpdate__TypeHint">GameUpdate</span></code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
</table>

### <em class="type"></em>GameUpdate

<p class='tags'>
<em>Type</em>
</p>

<p>
<p>Describes an available update for a particular game install.</p>

</p>

<p>
<strong>Fields</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>itemId</code></td>
<td><code class="typename">string</code></td>
<td><p>Identifier originally passed in CheckUpdateItem</p>
</td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td><p>Game we found an update for</p>
</td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#Upload__TypeHint">Upload</span></code></td>
<td><p>Upload to be installed</p>
</td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#Build__TypeHint">Build</span></code></td>
<td><p>Build to be installed (may be nil)</p>
</td>
</tr>
</table>


<div id="GameUpdate__TypeHint" style="display: none;" class="tip-content">
<p>GameUpdate <a href="#/?id=gameupdate">(Go to definition)</a></p>

<p>
<p>Describes an available update for a particular game install.</p>

</p>

<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
</tr>
<tr>
<td><code>itemId</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="struct-type">Game</span></code></td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="struct-type">Upload</span></code></td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="struct-type">Build</span></code></td>
</tr>
</table>

</div>


## General

### <em class="type"></em>GameCredentials

<p class='tags'>
<em>Type</em>
</p>

<p>
<p>GameCredentials contains all the credentials required to make API requests
including the download key if any</p>

</p>

<p>
<strong>Fields</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>server</code></td>
<td><code class="typename">string</code></td>
<td><p>Defaults to <code>https://itch.io</code></p>
</td>
</tr>
<tr>
<td><code>apiKey</code></td>
<td><code class="typename">string</code></td>
<td><p>A valid itch.io API key</p>
</td>
</tr>
<tr>
<td><code>downloadKey</code></td>
<td><code class="typename">number</code></td>
<td><p>A download key identifier, or 0 if no download key is available</p>
</td>
</tr>
</table>


<div id="GameCredentials__TypeHint" style="display: none;" class="tip-content">
<p>GameCredentials <a href="#/?id=gamecredentials">(Go to definition)</a></p>

<p>
<p>GameCredentials contains all the credentials required to make API requests
including the download key if any</p>

</p>

<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
</tr>
<tr>
<td><code>server</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>apiKey</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>downloadKey</code></td>
<td><code class="typename">number</code></td>
</tr>
</table>

</div>


## Install

### <em class="request-client-caller"></em>Game.FindUploads

<p class='tags'>
<em>Client request</em>
</p>

<p>
<p>Finds uploads compatible with the current runtime, for a given game</p>

</p>

<p>
<strong>Parameters</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td><p>Which game to find uploads for</p>
</td>
</tr>
<tr>
<td><code>credentials</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#GameCredentials__TypeHint">GameCredentials</span></code></td>
<td><p>The credentials to use to list uploads</p>
</td>
</tr>
</table>



<p>
<strong>Result</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>uploads</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#Upload__TypeHint">Upload</span>[]</code></td>
<td><p>A list of uploads that were found to be compatible.</p>
</td>
</tr>
</table>

### <em class="request-client-caller"></em>Operation.Start

<p class='tags'>
<em>Client request</em>
<em>Cancellable</em>
</p>

<p>
<p>Start a new operation (installing or uninstalling).</p>

</p>

<p>
<strong>Parameters</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>id</code></td>
<td><code class="typename">string</code></td>
<td><p>A UUID, generated by the client, used for referring to the task when cancelling it, for instance.</p>
</td>
</tr>
<tr>
<td><code>stagingFolder</code></td>
<td><code class="typename">string</code></td>
<td><p>A folder that butler can use to store temporary files, like partial downloads, checkpoint files, etc.</p>
</td>
</tr>
<tr>
<td><code>operation</code></td>
<td><code class="typename">Operation</code></td>
<td><p>Which operation to perform</p>
</td>
</tr>
<tr>
<td><code>installParams</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#InstallParams__TypeHint">InstallParams</span></code></td>
<td><p>Must be set if Operation is <code>install</code></p>
</td>
</tr>
<tr>
<td><code>uninstallParams</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#UninstallParams__TypeHint">UninstallParams</span></code></td>
<td><p>Must be set if Operation is <code>uninstall</code></p>
</td>
</tr>
</table>



<p>
<strong>Result</strong>: <em>none</em>
</p>

### <em class="request-client-caller"></em>Operation.Cancel

<p class='tags'>
<em>Client request</em>
</p>

<p>
<p>Attempt to gracefully cancel an ongoing operation.</p>

</p>

<p>
<strong>Parameters</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>id</code></td>
<td><code class="typename">string</code></td>
<td><p>The UUID of the task to cancel, as passed to <a href="#operationstart-request">Operation.Start</a></p>
</td>
</tr>
</table>



<p>
<strong>Result</strong>: <em>none</em>
</p>

### <em class="type"></em>InstallParams

<p class='tags'>
<em>Type</em>
</p>

<p>
<p>InstallParams contains all the parameters needed to perform
an installation for a game.</p>

</p>

<p>
<strong>Fields</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td><p>Which game to install</p>
</td>
</tr>
<tr>
<td><code>installFolder</code></td>
<td><code class="typename">string</code></td>
<td><p>An absolute path where to install the game</p>
</td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#Upload__TypeHint">Upload</span></code></td>
<td><p>Which upload to install @optional</p>
</td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#Build__TypeHint">Build</span></code></td>
<td><p>Which build to install @optional</p>
</td>
</tr>
<tr>
<td><code>credentials</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#GameCredentials__TypeHint">GameCredentials</span></code></td>
<td><p>Which credentials to use to install the game</p>
</td>
</tr>
<tr>
<td><code>ignoreInstallers</code></td>
<td><code class="typename">boolean</code></td>
<td><p>If true, do not run windows installers, just extract whatever to the install folder. @optional</p>
</td>
</tr>
</table>


<div id="InstallParams__TypeHint" style="display: none;" class="tip-content">
<p>InstallParams <a href="#/?id=installparams">(Go to definition)</a></p>

<p>
<p>InstallParams contains all the parameters needed to perform
an installation for a game.</p>

</p>

<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="struct-type">Game</span></code></td>
</tr>
<tr>
<td><code>installFolder</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="struct-type">Upload</span></code></td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="struct-type">Build</span></code></td>
</tr>
<tr>
<td><code>credentials</code></td>
<td><code class="typename"><span class="struct-type">GameCredentials</span></code></td>
</tr>
<tr>
<td><code>ignoreInstallers</code></td>
<td><code class="typename">boolean</code></td>
</tr>
</table>

</div>

### <em class="type"></em>UninstallParams

<p class='tags'>
<em>Type</em>
</p>

<p>
<p>UninstallParams contains all the parameters needed to perform
an uninstallation for a game.</p>

</p>

<p>
<strong>Fields</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>installFolder</code></td>
<td><code class="typename">string</code></td>
<td><p>Absolute path of the folder butler should uninstall</p>
</td>
</tr>
</table>


<div id="UninstallParams__TypeHint" style="display: none;" class="tip-content">
<p>UninstallParams <a href="#/?id=uninstallparams">(Go to definition)</a></p>

<p>
<p>UninstallParams contains all the parameters needed to perform
an uninstallation for a game.</p>

</p>

<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
</tr>
<tr>
<td><code>installFolder</code></td>
<td><code class="typename">string</code></td>
</tr>
</table>

</div>

### <em class="request-server-caller"></em>PickUpload

<p class='tags'>
<em>Server request</em>
<em>Dialog</em>
</p>

<p>
<p>Asks the user to pick between multiple available uploads</p>

</p>

<p>
<strong>Parameters</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>uploads</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#Upload__TypeHint">Upload</span>[]</code></td>
<td><p>An array of upload objects to choose from</p>
</td>
</tr>
</table>



<p>
<strong>Result</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>index</code></td>
<td><code class="typename">number</code></td>
<td><p>The index (in the original array) of the upload that was picked, or a negative value to cancel.</p>
</td>
</tr>
</table>

### <em class="request-server-caller"></em>GetReceipt

<p class='tags'>
<em>Server request</em>
<em>Deprecated</em>
</p>

<p>
<p>Retrieves existing receipt information for an install</p>

</p>

<p>
<strong>Parameters</strong>: <em>none</em>
</p>



<p>
<strong>Result</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>receipt</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#Receipt__TypeHint">Receipt</span></code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
</table>

### <em class="notification"></em>Operation.Progress

<p class='tags'>
<em>Notification</em>
</p>

<p>
<p>Sent periodically to inform on the current state an operation.</p>

</p>

<p>
<strong>Payload</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>progress</code></td>
<td><code class="typename">number</code></td>
<td><p>An overall progress value between 0 and 1</p>
</td>
</tr>
<tr>
<td><code>eta</code></td>
<td><code class="typename">number</code></td>
<td><p>Estimated completion time for the operation, in seconds (floating)</p>
</td>
</tr>
<tr>
<td><code>bps</code></td>
<td><code class="typename">number</code></td>
<td><p>Network bandwidth used, in bytes per second (floating)</p>
</td>
</tr>
</table>

### <em class="notification"></em>TaskStarted

<p class='tags'>
<em>Notification</em>
</p>

<p>
<p>Each operation is made up of one or more tasks. This notification
is sent whenever a task starts for an operation.</p>

</p>

<p>
<strong>Payload</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>reason</code></td>
<td><code class="typename">TaskReason</code></td>
<td><p>Why this task was started</p>
</td>
</tr>
<tr>
<td><code>type</code></td>
<td><code class="typename">TaskType</code></td>
<td><p>Is this task a download? An install?</p>
</td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td><p>The game this task is dealing with</p>
</td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#Upload__TypeHint">Upload</span></code></td>
<td><p>The upload this task is dealing with</p>
</td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#Build__TypeHint">Build</span></code></td>
<td><p>The build this task is dealing with (if any)</p>
</td>
</tr>
<tr>
<td><code>totalSize</code></td>
<td><code class="typename">number</code></td>
<td><p>Total size in bytes</p>
</td>
</tr>
</table>

### <em class="notification"></em>TaskSucceeded

<p class='tags'>
<em>Notification</em>
</p>

<p>
<p>Sent whenever a task succeeds for an operation.</p>

</p>

<p>
<strong>Payload</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>type</code></td>
<td><code class="typename">TaskType</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>installResult</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#InstallResult__TypeHint">InstallResult</span></code></td>
<td><p>If the task installed something, then this contains info about the game, upload, build that were installed</p>
</td>
</tr>
</table>

### <em class="type"></em>InstallResult

<p class='tags'>
<em>Type</em>
</p>


<p>
<strong>Fields</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#Upload__TypeHint">Upload</span></code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#Build__TypeHint">Build</span></code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
</table>


<div id="InstallResult__TypeHint" style="display: none;" class="tip-content">
<p>InstallResult <a href="#/?id=installresult">(Go to definition)</a></p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="struct-type">Game</span></code></td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="struct-type">Upload</span></code></td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="struct-type">Build</span></code></td>
</tr>
</table>

</div>


## Test

### <em class="request-client-caller"></em>Test.DoubleTwice

<p class='tags'>
<em>Client request</em>
</p>


<p>
<strong>Parameters</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>number</code></td>
<td><code class="typename">number</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
</table>



<p>
<strong>Result</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>number</code></td>
<td><code class="typename">number</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
</table>

### <em class="request-server-caller"></em>Test.Double

<p class='tags'>
<em>Server request</em>
</p>


<p>
<strong>Parameters</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>number</code></td>
<td><code class="typename">number</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
</table>


<p>
<p>Result for Test.Double</p>

</p>

<p>
<strong>Result</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>number</code></td>
<td><code class="typename">number</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
</table>


## Clean Downloads

### <em class="request-client-caller"></em>CleanDownloads.Search

<p class='tags'>
<em>Client request</em>
</p>

<p>
<p>Look for folders we can clean up in various download folders.
This finds anything that doesn&rsquo;t correspond to any current downloads
we know about.</p>

</p>

<p>
<strong>Parameters</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>roots</code></td>
<td><code class="typename">string[]</code></td>
<td><p>A list of folders to scan for potential subfolders to clean up</p>
</td>
</tr>
<tr>
<td><code>whitelist</code></td>
<td><code class="typename">string[]</code></td>
<td><p>A list of subfolders to not consider when cleaning (staging folders for in-progress downloads)</p>
</td>
</tr>
</table>



<p>
<strong>Result</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>entries</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#CleanDownloadsEntry__TypeHint">CleanDownloadsEntry</span>[]</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
</table>

### <em class="type"></em>CleanDownloadsEntry

<p class='tags'>
<em>Type</em>
</p>


<p>
<strong>Fields</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>path</code></td>
<td><code class="typename">string</code></td>
<td><p>The complete path of the file or folder we intend to remove</p>
</td>
</tr>
<tr>
<td><code>size</code></td>
<td><code class="typename">number</code></td>
<td><p>The size of the folder or file, in bytes</p>
</td>
</tr>
</table>


<div id="CleanDownloadsEntry__TypeHint" style="display: none;" class="tip-content">
<p>CleanDownloadsEntry <a href="#/?id=cleandownloadsentry">(Go to definition)</a></p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
</tr>
<tr>
<td><code>path</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>size</code></td>
<td><code class="typename">number</code></td>
</tr>
</table>

</div>

### <em class="request-client-caller"></em>CleanDownloads.Apply

<p class='tags'>
<em>Client request</em>
</p>

<p>
<p>Remove the specified entries from disk, freeing up disk space.</p>

</p>

<p>
<strong>Parameters</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>entries</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#CleanDownloadsEntry__TypeHint">CleanDownloadsEntry</span>[]</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
</table>



<p>
<strong>Result</strong>: <em>none</em>
</p>


## Miscellaneous

### <em class="notification"></em>Log

<p class='tags'>
<em>Notification</em>
</p>

<p>
<p>Sent any time butler needs to send a log message. The client should
relay them in their own stdout / stderr, and collect them so they
can be part of an issue report if something goes wrong.</p>

<p>Log</p>

</p>

<p>
<strong>Payload</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>level</code></td>
<td><code class="typename">string</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>message</code></td>
<td><code class="typename">string</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
</table>

### <em class="type"></em>User

<p class='tags'>
<em>Type</em>
</p>

<p>
<p>User represents an itch.io account, with basic profile info</p>

</p>

<p>
<strong>Fields</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>id</code></td>
<td><code class="typename">number</code></td>
<td><p>Site-wide unique identifier generated by itch.io</p>
</td>
</tr>
<tr>
<td><code>username</code></td>
<td><code class="typename">string</code></td>
<td><p>The user&rsquo;s username (used for login)</p>
</td>
</tr>
<tr>
<td><code>displayName</code></td>
<td><code class="typename">string</code></td>
<td><p>The user&rsquo;s display name: human-friendly, may contain spaces, unicode etc.</p>
</td>
</tr>
<tr>
<td><code>developer</code></td>
<td><code class="typename">boolean</code></td>
<td><p>Has the user opted into creating games?</p>
</td>
</tr>
<tr>
<td><code>pressUser</code></td>
<td><code class="typename">boolean</code></td>
<td><p>Is the user part of itch.io&rsquo;s press program?</p>
</td>
</tr>
<tr>
<td><code>url</code></td>
<td><code class="typename">string</code></td>
<td><p>The address of the user&rsquo;s page on itch.io</p>
</td>
</tr>
<tr>
<td><code>coverUrl</code></td>
<td><code class="typename">string</code></td>
<td><p>User&rsquo;s avatar, may be a GIF</p>
</td>
</tr>
<tr>
<td><code>stillCoverUrl</code></td>
<td><code class="typename">string</code></td>
<td><p>Static version of user&rsquo;s avatar, only set if the main cover URL is a GIF</p>
</td>
</tr>
</table>


<div id="User__TypeHint" style="display: none;" class="tip-content">
<p>User <a href="#/?id=user">(Go to definition)</a></p>

<p>
<p>User represents an itch.io account, with basic profile info</p>

</p>

<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
</tr>
<tr>
<td><code>id</code></td>
<td><code class="typename">number</code></td>
</tr>
<tr>
<td><code>username</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>displayName</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>developer</code></td>
<td><code class="typename">boolean</code></td>
</tr>
<tr>
<td><code>pressUser</code></td>
<td><code class="typename">boolean</code></td>
</tr>
<tr>
<td><code>url</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>coverUrl</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>stillCoverUrl</code></td>
<td><code class="typename">string</code></td>
</tr>
</table>

</div>

### <em class="type"></em>Game

<p class='tags'>
<em>Type</em>
</p>

<p>
<p>Game represents a page on itch.io, it could be a game,
a tool, a comic, etc.</p>

</p>

<p>
<strong>Fields</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>id</code></td>
<td><code class="typename">number</code></td>
<td><p>Site-wide unique identifier generated by itch.io</p>
</td>
</tr>
<tr>
<td><code>url</code></td>
<td><code class="typename">string</code></td>
<td><p>Canonical address of the game&rsquo;s page on itch.io</p>
</td>
</tr>
<tr>
<td><code>title</code></td>
<td><code class="typename">string</code></td>
<td><p>Human-friendly title (may contain any character)</p>
</td>
</tr>
<tr>
<td><code>shortText</code></td>
<td><code class="typename">string</code></td>
<td><p>Human-friendly short description</p>
</td>
</tr>
<tr>
<td><code>type</code></td>
<td><code class="typename">string</code></td>
<td><p>Downloadable game, html game, etc.</p>
</td>
</tr>
<tr>
<td><code>classification</code></td>
<td><code class="typename">string</code></td>
<td><p>Classification: game, tool, comic, etc.</p>
</td>
</tr>
<tr>
<td><code>coverUrl</code></td>
<td><code class="typename">string</code></td>
<td><p>Cover url (might be a GIF)</p>
</td>
</tr>
<tr>
<td><code>stillCoverUrl</code></td>
<td><code class="typename">string</code></td>
<td><p>Non-gif cover url, only set if main cover url is a GIF</p>
</td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename">string</code></td>
<td><p>Date the game was created</p>
</td>
</tr>
<tr>
<td><code>publishedAt</code></td>
<td><code class="typename">string</code></td>
<td><p>Date the game was published, empty if not currently published</p>
</td>
</tr>
<tr>
<td><code>minPrice</code></td>
<td><code class="typename">number</code></td>
<td><p>Price in cents of a dollar</p>
</td>
</tr>
<tr>
<td><code>inPressSystem</code></td>
<td><code class="typename">boolean</code></td>
<td><p>Is this game downloadable by press users for free?</p>
</td>
</tr>
<tr>
<td><code>hasDemo</code></td>
<td><code class="typename">boolean</code></td>
<td><p>Does this game have a demo that can be downloaded for free?</p>
</td>
</tr>
<tr>
<td><code>pOsx</code></td>
<td><code class="typename">boolean</code></td>
<td><p>Does this game have an upload tagged with &lsquo;macOS compatible&rsquo;? (creator-controlled)</p>
</td>
</tr>
<tr>
<td><code>pLinux</code></td>
<td><code class="typename">boolean</code></td>
<td><p>Does this game have an upload tagged with &lsquo;Linux compatible&rsquo;? (creator-controlled)</p>
</td>
</tr>
<tr>
<td><code>pWindows</code></td>
<td><code class="typename">boolean</code></td>
<td><p>Does this game have an upload tagged with &lsquo;Windows compatible&rsquo;? (creator-controlled)</p>
</td>
</tr>
<tr>
<td><code>pAndroid</code></td>
<td><code class="typename">boolean</code></td>
<td><p>Does this game have an upload tagged with &lsquo;Android compatible&rsquo;? (creator-controlled)</p>
</td>
</tr>
</table>


<div id="Game__TypeHint" style="display: none;" class="tip-content">
<p>Game <a href="#/?id=game">(Go to definition)</a></p>

<p>
<p>Game represents a page on itch.io, it could be a game,
a tool, a comic, etc.</p>

</p>

<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
</tr>
<tr>
<td><code>id</code></td>
<td><code class="typename">number</code></td>
</tr>
<tr>
<td><code>url</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>title</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>shortText</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>type</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>classification</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>coverUrl</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>stillCoverUrl</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>publishedAt</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>minPrice</code></td>
<td><code class="typename">number</code></td>
</tr>
<tr>
<td><code>inPressSystem</code></td>
<td><code class="typename">boolean</code></td>
</tr>
<tr>
<td><code>hasDemo</code></td>
<td><code class="typename">boolean</code></td>
</tr>
<tr>
<td><code>pOsx</code></td>
<td><code class="typename">boolean</code></td>
</tr>
<tr>
<td><code>pLinux</code></td>
<td><code class="typename">boolean</code></td>
</tr>
<tr>
<td><code>pWindows</code></td>
<td><code class="typename">boolean</code></td>
</tr>
<tr>
<td><code>pAndroid</code></td>
<td><code class="typename">boolean</code></td>
</tr>
</table>

</div>

### <em class="type"></em>Upload

<p class='tags'>
<em>Type</em>
</p>

<p>
<p>An Upload is a downloadable file. Some are wharf-enabled, which means
they&rsquo;re actually a &ldquo;channel&rdquo; that may contain multiple builds, pushed
with <a href="https://github.com/itchio/butler">https://github.com/itchio/butler</a></p>

</p>

<p>
<strong>Fields</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>id</code></td>
<td><code class="typename">number</code></td>
<td><p>Site-wide unique identifier generated by itch.io</p>
</td>
</tr>
<tr>
<td><code>filename</code></td>
<td><code class="typename">string</code></td>
<td><p>Original file name (example: <code>Overland_x64.zip</code>)</p>
</td>
</tr>
<tr>
<td><code>displayName</code></td>
<td><code class="typename">string</code></td>
<td><p>Human-friendly name set by developer (example: <code>Overland for Windows 64-bit</code>)</p>
</td>
</tr>
<tr>
<td><code>size</code></td>
<td><code class="typename">number</code></td>
<td><p>Size of upload in bytes. For wharf-enabled uploads, it&rsquo;s the archive size.</p>
</td>
</tr>
<tr>
<td><code>channelName</code></td>
<td><code class="typename">string</code></td>
<td><p>Name of the wharf channel for this upload, if it&rsquo;s a wharf-enabled upload</p>
</td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#Build__TypeHint">Build</span></code></td>
<td><p>Latest build for this upload, if it&rsquo;s a wharf-enabled upload</p>
</td>
</tr>
<tr>
<td><code>demo</code></td>
<td><code class="typename">boolean</code></td>
<td><p>Is this upload a demo that can be downloaded for free?</p>
</td>
</tr>
<tr>
<td><code>preorder</code></td>
<td><code class="typename">boolean</code></td>
<td><p>Is this upload a pre-order placeholder?</p>
</td>
</tr>
<tr>
<td><code>type</code></td>
<td><code class="typename">string</code></td>
<td><p>Upload type: default, soundtrack, etc.</p>
</td>
</tr>
<tr>
<td><code>pOsx</code></td>
<td><code class="typename">boolean</code></td>
<td><p>Is this upload tagged with &lsquo;macOS compatible&rsquo;? (creator-controlled)</p>
</td>
</tr>
<tr>
<td><code>pLinux</code></td>
<td><code class="typename">boolean</code></td>
<td><p>Is this upload tagged with &lsquo;Linux compatible&rsquo;? (creator-controlled)</p>
</td>
</tr>
<tr>
<td><code>pWindows</code></td>
<td><code class="typename">boolean</code></td>
<td><p>Is this upload tagged with &lsquo;Windows compatible&rsquo;? (creator-controlled)</p>
</td>
</tr>
<tr>
<td><code>pAndroid</code></td>
<td><code class="typename">boolean</code></td>
<td><p>Is this upload tagged with &lsquo;Android compatible&rsquo;? (creator-controlled)</p>
</td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename">string</code></td>
<td><p>Date this upload was created at</p>
</td>
</tr>
<tr>
<td><code>updatedAt</code></td>
<td><code class="typename">string</code></td>
<td><p>Date this upload was last updated at (order changed, display name set, etc.)</p>
</td>
</tr>
</table>


<div id="Upload__TypeHint" style="display: none;" class="tip-content">
<p>Upload <a href="#/?id=upload">(Go to definition)</a></p>

<p>
<p>An Upload is a downloadable file. Some are wharf-enabled, which means
they&rsquo;re actually a &ldquo;channel&rdquo; that may contain multiple builds, pushed
with <a href="https://github.com/itchio/butler">https://github.com/itchio/butler</a></p>

</p>

<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
</tr>
<tr>
<td><code>id</code></td>
<td><code class="typename">number</code></td>
</tr>
<tr>
<td><code>filename</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>displayName</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>size</code></td>
<td><code class="typename">number</code></td>
</tr>
<tr>
<td><code>channelName</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="struct-type">Build</span></code></td>
</tr>
<tr>
<td><code>demo</code></td>
<td><code class="typename">boolean</code></td>
</tr>
<tr>
<td><code>preorder</code></td>
<td><code class="typename">boolean</code></td>
</tr>
<tr>
<td><code>type</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>pOsx</code></td>
<td><code class="typename">boolean</code></td>
</tr>
<tr>
<td><code>pLinux</code></td>
<td><code class="typename">boolean</code></td>
</tr>
<tr>
<td><code>pWindows</code></td>
<td><code class="typename">boolean</code></td>
</tr>
<tr>
<td><code>pAndroid</code></td>
<td><code class="typename">boolean</code></td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>updatedAt</code></td>
<td><code class="typename">string</code></td>
</tr>
</table>

</div>

### <em class="type"></em>Collection

<p class='tags'>
<em>Type</em>
</p>

<p>
<p>A Collection is a set of games, curated by humans.</p>

</p>

<p>
<strong>Fields</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>id</code></td>
<td><code class="typename">number</code></td>
<td><p>Site-wide unique identifier generated by itch.io</p>
</td>
</tr>
<tr>
<td><code>title</code></td>
<td><code class="typename">string</code></td>
<td><p>Human-friendly title for collection, for example <code>Couch coop games</code></p>
</td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename">string</code></td>
<td><p>Date this collection was created at</p>
</td>
</tr>
<tr>
<td><code>updatedAt</code></td>
<td><code class="typename">string</code></td>
<td><p>Date this collection was last updated at (item added, title set, etc.)</p>
</td>
</tr>
<tr>
<td><code>gamesCount</code></td>
<td><code class="typename">number</code></td>
<td><p>Number of games in the collection. This might not be accurate as some games might not be accessible to whoever is asking (project page deleted, visibility level changed, etc.)</p>
</td>
</tr>
</table>


<div id="Collection__TypeHint" style="display: none;" class="tip-content">
<p>Collection <a href="#/?id=collection">(Go to definition)</a></p>

<p>
<p>A Collection is a set of games, curated by humans.</p>

</p>

<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
</tr>
<tr>
<td><code>id</code></td>
<td><code class="typename">number</code></td>
</tr>
<tr>
<td><code>title</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>updatedAt</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>gamesCount</code></td>
<td><code class="typename">number</code></td>
</tr>
</table>

</div>

### <em class="type"></em>DownloadKey

<p class='tags'>
<em>Type</em>
</p>

<p>
<p>A download key is often generated when a purchase is made, it
allows downloading uploads for a game that are not available
for free.</p>

</p>

<p>
<strong>Fields</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>id</code></td>
<td><code class="typename">number</code></td>
<td><p>Site-wide unique identifier generated by itch.io</p>
</td>
</tr>
<tr>
<td><code>gameId</code></td>
<td><code class="typename">number</code></td>
<td><p>Identifier of the game to which this download key grants access</p>
</td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename">string</code></td>
<td><p>Date this key was created at (often coincides with purchase time)</p>
</td>
</tr>
<tr>
<td><code>updatedAt</code></td>
<td><code class="typename">string</code></td>
<td><p>Date this key was last updated at</p>
</td>
</tr>
<tr>
<td><code>ownerId</code></td>
<td><code class="typename">number</code></td>
<td><p>Identifier of the itch.io user to which this key belongs</p>
</td>
</tr>
</table>


<div id="DownloadKey__TypeHint" style="display: none;" class="tip-content">
<p>DownloadKey <a href="#/?id=downloadkey">(Go to definition)</a></p>

<p>
<p>A download key is often generated when a purchase is made, it
allows downloading uploads for a game that are not available
for free.</p>

</p>

<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
</tr>
<tr>
<td><code>id</code></td>
<td><code class="typename">number</code></td>
</tr>
<tr>
<td><code>gameId</code></td>
<td><code class="typename">number</code></td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="struct-type">Game</span></code></td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>updatedAt</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>ownerId</code></td>
<td><code class="typename">number</code></td>
</tr>
</table>

</div>

### <em class="type"></em>Build

<p class='tags'>
<em>Type</em>
</p>

<p>
<p>Build contains information about a specific build</p>

</p>

<p>
<strong>Fields</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>id</code></td>
<td><code class="typename">number</code></td>
<td><p>Site-wide unique identifier generated by itch.io</p>
</td>
</tr>
<tr>
<td><code>parentBuildId</code></td>
<td><code class="typename">number</code></td>
<td><p>Identifier of the build before this one on the same channel, or 0 if this is the initial build.</p>
</td>
</tr>
<tr>
<td><code>state</code></td>
<td><code class="typename">BuildState</code></td>
<td><p>State of the build: started, processing, etc.</p>
</td>
</tr>
<tr>
<td><code>version</code></td>
<td><code class="typename">number</code></td>
<td><p>Automatically-incremented version number, starting with 1</p>
</td>
</tr>
<tr>
<td><code>userVersion</code></td>
<td><code class="typename">string</code></td>
<td><p>Value specified by developer with <code>--userversion</code> when pushing a build Might not be unique across builds of a given channel.</p>
</td>
</tr>
<tr>
<td><code>files</code></td>
<td><code class="typename">BuildFile[]</code></td>
<td><p>Files associated with this build - often at least an archive, a signature, and a patch. Some might be missing while the build is still processing or if processing has failed.</p>
</td>
</tr>
<tr>
<td><code>user</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#User__TypeHint">User</span></code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename">string</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>updatedAt</code></td>
<td><code class="typename">string</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
</table>


<div id="Build__TypeHint" style="display: none;" class="tip-content">
<p>Build <a href="#/?id=build">(Go to definition)</a></p>

<p>
<p>Build contains information about a specific build</p>

</p>

<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
</tr>
<tr>
<td><code>id</code></td>
<td><code class="typename">number</code></td>
</tr>
<tr>
<td><code>parentBuildId</code></td>
<td><code class="typename">number</code></td>
</tr>
<tr>
<td><code>state</code></td>
<td><code class="typename">BuildState</code></td>
</tr>
<tr>
<td><code>version</code></td>
<td><code class="typename">number</code></td>
</tr>
<tr>
<td><code>userVersion</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>files</code></td>
<td><code class="typename">BuildFile[]</code></td>
</tr>
<tr>
<td><code>user</code></td>
<td><code class="typename"><span class="struct-type">User</span></code></td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>updatedAt</code></td>
<td><code class="typename">string</code></td>
</tr>
</table>

</div>

### <em class="type"></em>Verdict

<p class='tags'>
<em>Type</em>
</p>


<p>
<strong>Fields</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>basePath</code></td>
<td><code class="typename">string</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>totalSize</code></td>
<td><code class="typename">number</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>candidates</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#Candidate__TypeHint">Candidate</span>[]</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
</table>


<div id="Verdict__TypeHint" style="display: none;" class="tip-content">
<p>Verdict <a href="#/?id=verdict">(Go to definition)</a></p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
</tr>
<tr>
<td><code>basePath</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>totalSize</code></td>
<td><code class="typename">number</code></td>
</tr>
<tr>
<td><code>candidates</code></td>
<td><code class="typename"><span class="struct-type">Candidate</span>[]</code></td>
</tr>
</table>

</div>

### <em class="type"></em>Candidate

<p class='tags'>
<em>Type</em>
</p>

<p>
<p>Candidate indicates what&rsquo;s interesting about a file</p>

</p>

<p>
<strong>Fields</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>path</code></td>
<td><code class="typename">string</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>mode</code></td>
<td><code class="typename">uint32</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>depth</code></td>
<td><code class="typename">int</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>flavor</code></td>
<td><code class="typename">Flavor</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>arch</code></td>
<td><code class="typename">Arch</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>size</code></td>
<td><code class="typename">number</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>spell</code></td>
<td><code class="typename">string[]</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>windowsInfo</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#WindowsInfo__TypeHint">WindowsInfo</span></code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>linuxInfo</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#LinuxInfo__TypeHint">LinuxInfo</span></code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>macosInfo</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#MacosInfo__TypeHint">MacosInfo</span></code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>loveInfo</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#LoveInfo__TypeHint">LoveInfo</span></code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>scriptInfo</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#ScriptInfo__TypeHint">ScriptInfo</span></code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>jarInfo</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#JarInfo__TypeHint">JarInfo</span></code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
</table>


<div id="Candidate__TypeHint" style="display: none;" class="tip-content">
<p>Candidate <a href="#/?id=candidate">(Go to definition)</a></p>

<p>
<p>Candidate indicates what&rsquo;s interesting about a file</p>

</p>

<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
</tr>
<tr>
<td><code>path</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>mode</code></td>
<td><code class="typename">uint32</code></td>
</tr>
<tr>
<td><code>depth</code></td>
<td><code class="typename">int</code></td>
</tr>
<tr>
<td><code>flavor</code></td>
<td><code class="typename">Flavor</code></td>
</tr>
<tr>
<td><code>arch</code></td>
<td><code class="typename">Arch</code></td>
</tr>
<tr>
<td><code>size</code></td>
<td><code class="typename">number</code></td>
</tr>
<tr>
<td><code>spell</code></td>
<td><code class="typename">string[]</code></td>
</tr>
<tr>
<td><code>windowsInfo</code></td>
<td><code class="typename"><span class="struct-type">WindowsInfo</span></code></td>
</tr>
<tr>
<td><code>linuxInfo</code></td>
<td><code class="typename"><span class="struct-type">LinuxInfo</span></code></td>
</tr>
<tr>
<td><code>macosInfo</code></td>
<td><code class="typename"><span class="struct-type">MacosInfo</span></code></td>
</tr>
<tr>
<td><code>loveInfo</code></td>
<td><code class="typename"><span class="struct-type">LoveInfo</span></code></td>
</tr>
<tr>
<td><code>scriptInfo</code></td>
<td><code class="typename"><span class="struct-type">ScriptInfo</span></code></td>
</tr>
<tr>
<td><code>jarInfo</code></td>
<td><code class="typename"><span class="struct-type">JarInfo</span></code></td>
</tr>
</table>

</div>

### <em class="type"></em>WindowsInfo

<p class='tags'>
<em>Type</em>
</p>


<p>
<strong>Fields</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>installerType</code></td>
<td><code class="typename">WindowsInstallerType</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>uninstaller</code></td>
<td><code class="typename">boolean</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>gui</code></td>
<td><code class="typename">boolean</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>dotNet</code></td>
<td><code class="typename">boolean</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
</table>


<div id="WindowsInfo__TypeHint" style="display: none;" class="tip-content">
<p>WindowsInfo <a href="#/?id=windowsinfo">(Go to definition)</a></p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
</tr>
<tr>
<td><code>installerType</code></td>
<td><code class="typename">WindowsInstallerType</code></td>
</tr>
<tr>
<td><code>uninstaller</code></td>
<td><code class="typename">boolean</code></td>
</tr>
<tr>
<td><code>gui</code></td>
<td><code class="typename">boolean</code></td>
</tr>
<tr>
<td><code>dotNet</code></td>
<td><code class="typename">boolean</code></td>
</tr>
</table>

</div>

### <em class="type"></em>MacosInfo

<p class='tags'>
<em>Type</em>
</p>


<p>
<strong>Fields</strong>: <em>none</em>
</p>


<div id="MacosInfo__TypeHint" style="display: none;" class="tip-content">
<p>MacosInfo <a href="#/?id=macosinfo">(Go to definition)</a></p>

</div>

### <em class="type"></em>LinuxInfo

<p class='tags'>
<em>Type</em>
</p>


<p>
<strong>Fields</strong>: <em>none</em>
</p>


<div id="LinuxInfo__TypeHint" style="display: none;" class="tip-content">
<p>LinuxInfo <a href="#/?id=linuxinfo">(Go to definition)</a></p>

</div>

### <em class="type"></em>LoveInfo

<p class='tags'>
<em>Type</em>
</p>


<p>
<strong>Fields</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>version</code></td>
<td><code class="typename">string</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
</table>


<div id="LoveInfo__TypeHint" style="display: none;" class="tip-content">
<p>LoveInfo <a href="#/?id=loveinfo">(Go to definition)</a></p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
</tr>
<tr>
<td><code>version</code></td>
<td><code class="typename">string</code></td>
</tr>
</table>

</div>

### <em class="type"></em>ScriptInfo

<p class='tags'>
<em>Type</em>
</p>


<p>
<strong>Fields</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>interpreter</code></td>
<td><code class="typename">string</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
</table>


<div id="ScriptInfo__TypeHint" style="display: none;" class="tip-content">
<p>ScriptInfo <a href="#/?id=scriptinfo">(Go to definition)</a></p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
</tr>
<tr>
<td><code>interpreter</code></td>
<td><code class="typename">string</code></td>
</tr>
</table>

</div>

### <em class="type"></em>JarInfo

<p class='tags'>
<em>Type</em>
</p>


<p>
<strong>Fields</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>mainClass</code></td>
<td><code class="typename">string</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
</table>


<div id="JarInfo__TypeHint" style="display: none;" class="tip-content">
<p>JarInfo <a href="#/?id=jarinfo">(Go to definition)</a></p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
</tr>
<tr>
<td><code>mainClass</code></td>
<td><code class="typename">string</code></td>
</tr>
</table>

</div>

### <em class="type"></em>Receipt

<p class='tags'>
<em>Type</em>
</p>


<p>
<strong>Fields</strong>: 
</p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
<th>Description</th>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#Upload__TypeHint">Upload</span></code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="struct-type" data-tip-selector="#Build__TypeHint">Build</span></code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>files</code></td>
<td><code class="typename">string[]</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>installerName</code></td>
<td><code class="typename">string</code></td>
<td><p><em>undocumented</em></p>
</td>
</tr>
<tr>
<td><code>msiProductCode</code></td>
<td><code class="typename">string</code></td>
<td><p>optional, installer-specific fields</p>
</td>
</tr>
</table>


<div id="Receipt__TypeHint" style="display: none;" class="tip-content">
<p>Receipt <a href="#/?id=receipt">(Go to definition)</a></p>


<table class="field-table">
<tr>
<th>Name</th>
<th>Type</th>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="struct-type">Game</span></code></td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="struct-type">Upload</span></code></td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="struct-type">Build</span></code></td>
</tr>
<tr>
<td><code>files</code></td>
<td><code class="typename">string[]</code></td>
</tr>
<tr>
<td><code>installerName</code></td>
<td><code class="typename">string</code></td>
</tr>
<tr>
<td><code>msiProductCode</code></td>
<td><code class="typename">string</code></td>
</tr>
</table>

</div>


