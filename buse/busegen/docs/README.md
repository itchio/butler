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

!> Contrary to most JSON-RPC services, it's not recommended
   to keep a single instance of butler running and make all requests
   to it (like a server). Instead, start a new butler instance for each
   individual task you want to achieve, like logging in, performing a search,
   or cleaning downloads.

## Protocol

Requests, results, and notifications are sent over TCP, separated by
a newline (`\n`) character.

# Requests

Requests are essentially procedure calls: they're made asynchronously, and
a result is sent asynchronously. They may also fail, in which case
you get an error back, with details.

Some requests may complete almost instantly, and have an empty result
Still, waiting for the result lets you know that the peer has received
the request and processed it successfully.

Some requests are made by the client to butler (like CheckUpdate),
others are made from butler to the client (like AllowSandboxSetup)

## VersionGet

Version.Get


Parameters:

*empty*

Result:

Name | Type | Description
--- | --- | ---
**version** | `string` | Something short, like `v8.0.0` 
**versionString** | `string` | Something long, like `v8.0.0, built on Aug 27 2017 @ 01:13:55, ref d833cc0aeea81c236c81dffb27bc18b2b8d8b290` 

## GameFindUploads

Game.FindUploads


Parameters:

Name | Type | Description
--- | --- | ---
**game** | `itchio.Game` | *undocumented*
**credentials** | `GameCredentials` | *undocumented*

Result:

Name | Type | Description
--- | --- | ---
**uploads** | `itchio.Upload[]` | *undocumented*

## OperationStart

Operation.Start


Parameters:

Name | Type | Description
--- | --- | ---
**id** | `string` | *undocumented*
**stagingFolder** | `string` | *undocumented*
**operation** | `Operation` | *undocumented*
**installParams** | `InstallParams` | this is more or less a union, the relevant field should be set depending on the 'Operation' type 
**uninstallParams** | `UninstallParams` | *undocumented*

Result:

*empty*

## OperationCancel

Operation.Cancel


Parameters:

Name | Type | Description
--- | --- | ---
**id** | `string` | *undocumented*

Result:

*empty*

## Install

InstallParams contains all the parameters needed to perform
an installation for a game


Parameters:

Name | Type | Description
--- | --- | ---
**game** | `itchio.Game` | *undocumented*
**installFolder** | `string` | *undocumented*
**upload** | `itchio.Upload` | *undocumented*
**build** | `itchio.Build` | *undocumented*
**credentials** | `GameCredentials` | *undocumented*
**ignoreInstallers** | `boolean` | *undocumented*

Result:

Name | Type | Description
--- | --- | ---
**game** | `itchio.Game` | *undocumented*
**upload** | `itchio.Upload` | *undocumented*
**build** | `itchio.Build` | *undocumented*

## Uninstall

*undocumented*

Parameters:

Name | Type | Description
--- | --- | ---
**installFolder** | `string` | *undocumented*

Result:

*empty*

## PickUpload

*undocumented*

Parameters:

Name | Type | Description
--- | --- | ---
**uploads** | `itchio.Upload[]` | *undocumented*

Result:

Name | Type | Description
--- | --- | ---
**index** | `number` | *undocumented*

## GetReceipt

*undocumented*

Parameters:

*empty*

Result:

Name | Type | Description
--- | --- | ---
**receipt** | `bfs.Receipt` | *undocumented*

## CheckUpdate

*undocumented*

Parameters:

Name | Type | Description
--- | --- | ---
**items** | `CheckUpdateItem[]` | *undocumented*

Result:

Name | Type | Description
--- | --- | ---
**updates** | `GameUpdate[]` | *undocumented*
**warnings** | `string[]` | *undocumented*

## Launch

*undocumented*

Parameters:

Name | Type | Description
--- | --- | ---
**installFolder** | `string` | *undocumented*
**game** | `itchio.Game` | *undocumented*
**upload** | `itchio.Upload` | *undocumented*
**build** | `itchio.Build` | *undocumented*
**verdict** | `configurator.Verdict` | *undocumented*
**prereqsDir** | `string` | *undocumented*
**forcePrereqs** | `boolean` | *undocumented*
**sandbox** | `boolean` | *undocumented*
**credentials** | `GameCredentials` | Used for subkeying 

Result:

*empty*

## PickManifestAction

*undocumented*

Parameters:

Name | Type | Description
--- | --- | ---
**actions** | `manifest.Action[]` | *undocumented*

Result:

Name | Type | Description
--- | --- | ---
**name** | `string` | *undocumented*

## ShellLaunch

*undocumented*

Parameters:

Name | Type | Description
--- | --- | ---
**itemPath** | `string` | *undocumented*

Result:

*empty*

## HTMLLaunch

*undocumented*

Parameters:

Name | Type | Description
--- | --- | ---
**rootFolder** | `string` | *undocumented*
**indexPath** | `string` | *undocumented*
**args** | `string[]` | *undocumented*
**env** | `Map<string, string>` | *undocumented*

Result:

*empty*

## URLLaunch

*undocumented*

Parameters:

Name | Type | Description
--- | --- | ---
**url** | `string` | *undocumented*

Result:

*empty*

## SaveVerdict

*undocumented*

Parameters:

Name | Type | Description
--- | --- | ---
**verdict** | `configurator.Verdict` | *undocumented*

Result:

*empty*

## AllowSandboxSetup

*undocumented*

Parameters:

*empty*

Result:

*empty*

## PrereqsFailed

*undocumented*

Parameters:

Name | Type | Description
--- | --- | ---
**error** | `string` | *undocumented*
**errorStack** | `string` | *undocumented*

Result:

Name | Type | Description
--- | --- | ---
**continue** | `boolean` | *undocumented*

## CleanDownloadsSearch

CleanDownloads.Search


Parameters:

Name | Type | Description
--- | --- | ---
**roots** | `string[]` | A list of folders to scan for potential subfolders to clean up 
**whitelist** | `string[]` | A list of subfolders to not consider when cleaning (staging folders for in-progress downloads) 

Result:

Name | Type | Description
--- | --- | ---
**entries** | `CleanDownloadsEntry[]` | *undocumented*

## CleanDownloadsApply

*undocumented*

Parameters:

Name | Type | Description
--- | --- | ---
**entries** | `CleanDownloadsEntry[]` | *undocumented*

Result:

*empty*



# Notifications

Notifications are messages that can be sent at any time, in any direction.

There is no way to check that a notification was delivered, only that it was
sent (but the other peer may fail to process it before it exits).

## OperationProgress

Operation.Progress
Sent periodically to inform on the current state an operation


Payload:

Name | Type | Description
--- | --- | ---
**progress** | `number` | *undocumented*
**eta** | `number` | *undocumented*
**bps** | `number` | *undocumented*

## TaskStarted

*undocumented*

Payload:

Name | Type | Description
--- | --- | ---
**reason** | `TaskReason` | *undocumented*
**type** | `TaskType` | *undocumented*
**game** | `itchio.Game` | *undocumented*
**upload** | `itchio.Upload` | *undocumented*
**build** | `itchio.Build` | *undocumented*
**totalSize** | `number` | *undocumented*

## TaskSucceeded

*undocumented*

Payload:

Name | Type | Description
--- | --- | ---
**type** | `TaskType` | *undocumented*
**installResult** | `InstallResult` | If the task installed something, then this contains info about the game, upload, build that were installed 

## GameUpdateAvailable

*undocumented*

Payload:

Name | Type | Description
--- | --- | ---
**update** | `GameUpdate` | *undocumented*

## LaunchRunning

*undocumented*

Payload:

*empty*

## LaunchExited

*undocumented*

Payload:

*empty*

## PrereqsStarted

*undocumented*

Payload:

Name | Type | Description
--- | --- | ---
**tasks** | `Map<string, PrereqTask>` | *undocumented*

## PrereqsTaskState

*undocumented*

Payload:

Name | Type | Description
--- | --- | ---
**name** | `string` | *undocumented*
**status** | `PrereqStatus` | *undocumented*
**progress** | `number` | *undocumented*
**eta** | `number` | *undocumented*
**bps** | `number` | *undocumented*

## PrereqsEnded

*undocumented*

Payload:

*empty*

## Log

Log


Payload:

Name | Type | Description
--- | --- | ---
**level** | `string` | *undocumented*
**message** | `string` | *undocumented*



# Types

These are some types that are used throughout the API:

## GameCredentials

GameCredentials contains all the credentials required to make API requests
including the download key if any


Fields:

Name | Type | Description
--- | --- | ---
**server** | `string` | *undocumented*
**apiKey** | `string` | *undocumented*
**downloadKey** | `number` | *undocumented*

## CheckUpdateItem

*undocumented*

Fields:

Name | Type | Description
--- | --- | ---
**itemId** | `string` | *undocumented*
**installedAt** | `string` | *undocumented*
**game** | `itchio.Game` | *undocumented*
**upload** | `itchio.Upload` | *undocumented*
**build** | `itchio.Build` | *undocumented*
**credentials** | `GameCredentials` | *undocumented*

## GameUpdate

*undocumented*

Fields:

Name | Type | Description
--- | --- | ---
**itemId** | `string` | *undocumented*
**game** | `itchio.Game` | *undocumented*
**upload** | `itchio.Upload` | *undocumented*
**build** | `itchio.Build` | *undocumented*

## AllowSandboxSetupResponse

*undocumented*

Fields:

Name | Type | Description
--- | --- | ---
**allow** | `boolean` | *undocumented*

## PrereqTask

*undocumented*

Fields:

Name | Type | Description
--- | --- | ---
**fullName** | `string` | *undocumented*
**order** | `int` | *undocumented*

## CleanDownloadsEntry

*undocumented*

Fields:

Name | Type | Description
--- | --- | ---
**path** | `string` | *undocumented*
**size** | `number` | *undocumented*

## TestDoubleTwiceRequest

Test.DoubleTwice


Fields:

Name | Type | Description
--- | --- | ---
**number** | `number` | *undocumented*

## TestDoubleRequest

Test.Double


Fields:

Name | Type | Description
--- | --- | ---
**number** | `number` | *undocumented*


