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

&ast.Object{Kind:3, Name:"VersionGetParams", Decl:(*ast.TypeSpec)(0xc0420801b0), Data:interface {}(nil), Type:interface {}(nil)}

Comment:

(null doc)

Doc:

(null doc)

Parameters:

*empty*

Result:

Name | Type | Description
--- | --- | ---
**version** | `string` | Something short, like `v8.0.0` 
**versionString** | `string` | Something long, like `v8.0.0, built on Aug 27 2017 @ 01:13:55, ref d833cc0aeea81c236c81dffb27bc18b2b8d8b290` 

## GameFindUploads

&ast.Object{Kind:3, Name:"GameFindUploadsParams", Decl:(*ast.TypeSpec)(0xc042080270), Data:interface {}(nil), Type:interface {}(nil)}

Comment:

(null doc)

Doc:

(null doc)

Parameters:

Name | Type | Description
--- | --- | ---
**game** | `itchio.Game` | (null doc)
**credentials** | `GameCredentials` | (null doc)

Result:

Name | Type | Description
--- | --- | ---
**uploads** | `itchio.Upload[]` | (null doc)

## OperationStart

&ast.Object{Kind:3, Name:"OperationStartParams", Decl:(*ast.TypeSpec)(0xc042080390), Data:interface {}(nil), Type:interface {}(nil)}

Comment:

(null doc)

Doc:

(null doc)

Parameters:

Name | Type | Description
--- | --- | ---
**id** | `string` | (null doc)
**stagingFolder** | `string` | (null doc)
**operation** | `Operation` | (null doc)
**installParams** | `InstallParams` | this is more or less a union, the relevant field should be set depending on the 'Operation' type 
**uninstallParams** | `UninstallParams` | (null doc)

Result:

*empty*

## OperationCancel

&ast.Object{Kind:3, Name:"OperationCancelParams", Decl:(*ast.TypeSpec)(0xc0420803f0), Data:interface {}(nil), Type:interface {}(nil)}

Comment:

(null doc)

Doc:

(null doc)

Parameters:

Name | Type | Description
--- | --- | ---
**id** | `string` | (null doc)

Result:

*empty*

## Install

&ast.Object{Kind:3, Name:"InstallParams", Decl:(*ast.TypeSpec)(0xc0420804b0), Data:interface {}(nil), Type:interface {}(nil)}

Comment:

(null doc)

Doc:

(null doc)

Parameters:

Name | Type | Description
--- | --- | ---
**game** | `itchio.Game` | (null doc)
**installFolder** | `string` | (null doc)
**upload** | `itchio.Upload` | (null doc)
**build** | `itchio.Build` | (null doc)
**credentials** | `GameCredentials` | (null doc)
**ignoreInstallers** | `boolean` | (null doc)

Result:

Name | Type | Description
--- | --- | ---
**game** | `itchio.Game` | (null doc)
**upload** | `itchio.Upload` | (null doc)
**build** | `itchio.Build` | (null doc)

## Uninstall

&ast.Object{Kind:3, Name:"UninstallParams", Decl:(*ast.TypeSpec)(0xc042080510), Data:interface {}(nil), Type:interface {}(nil)}

Comment:

(null doc)

Doc:

(null doc)

Parameters:

Name | Type | Description
--- | --- | ---
**installFolder** | `string` | (null doc)

Result:

*empty*

## PickUpload

&ast.Object{Kind:3, Name:"PickUploadParams", Decl:(*ast.TypeSpec)(0xc0420805d0), Data:interface {}(nil), Type:interface {}(nil)}

Comment:

(null doc)

Doc:

(null doc)

Parameters:

Name | Type | Description
--- | --- | ---
**uploads** | `itchio.Upload[]` | (null doc)

Result:

Name | Type | Description
--- | --- | ---
**index** | `number` | (null doc)

## GetReceipt

&ast.Object{Kind:3, Name:"GetReceiptParams", Decl:(*ast.TypeSpec)(0xc0420806c0), Data:interface {}(nil), Type:interface {}(nil)}

Comment:

(null doc)

Doc:

(null doc)

Parameters:

*empty*

Result:

Name | Type | Description
--- | --- | ---
**receipt** | `bfs.Receipt` | (null doc)

## CheckUpdate

&ast.Object{Kind:3, Name:"CheckUpdateParams", Decl:(*ast.TypeSpec)(0xc0420809c0), Data:interface {}(nil), Type:interface {}(nil)}

Comment:

(null doc)

Doc:

(null doc)

Parameters:

Name | Type | Description
--- | --- | ---
**items** | `CheckUpdateItem[]` | (null doc)

Result:

Name | Type | Description
--- | --- | ---
**updates** | `GameUpdate[]` | (null doc)
**warnings** | `string[]` | (null doc)

## Launch

&ast.Object{Kind:3, Name:"LaunchParams", Decl:(*ast.TypeSpec)(0xc042080c30), Data:interface {}(nil), Type:interface {}(nil)}

Comment:

(null doc)

Doc:

(null doc)

Parameters:

Name | Type | Description
--- | --- | ---
**installFolder** | `string` | (null doc)
**game** | `itchio.Game` | (null doc)
**upload** | `itchio.Upload` | (null doc)
**build** | `itchio.Build` | (null doc)
**verdict** | `configurator.Verdict` | (null doc)
**prereqsDir** | `string` | (null doc)
**forcePrereqs** | `boolean` | (null doc)
**sandbox** | `boolean` | (null doc)
**credentials** | `GameCredentials` | Used for subkeying 

Result:

*empty*

## PickManifestAction

&ast.Object{Kind:3, Name:"PickManifestActionParams", Decl:(*ast.TypeSpec)(0xc042080db0), Data:interface {}(nil), Type:interface {}(nil)}

Comment:

(null doc)

Doc:

(null doc)

Parameters:

Name | Type | Description
--- | --- | ---
**actions** | `manifest.Action[]` | (null doc)

Result:

Name | Type | Description
--- | --- | ---
**name** | `string` | (null doc)

## ShellLaunch

&ast.Object{Kind:3, Name:"ShellLaunchParams", Decl:(*ast.TypeSpec)(0xc042080ea0), Data:interface {}(nil), Type:interface {}(nil)}

Comment:

(null doc)

Doc:

(null doc)

Parameters:

Name | Type | Description
--- | --- | ---
**itemPath** | `string` | (null doc)

Result:

*empty*

## HTMLLaunch

&ast.Object{Kind:3, Name:"HTMLLaunchParams", Decl:(*ast.TypeSpec)(0xc042080f60), Data:interface {}(nil), Type:interface {}(nil)}

Comment:

(null doc)

Doc:

(null doc)

Parameters:

Name | Type | Description
--- | --- | ---
**rootFolder** | `string` | (null doc)
**indexPath** | `string` | (null doc)
**args** | `string[]` | (null doc)
**env** | `Map<string, string>` | (null doc)

Result:

*empty*

## URLLaunch

&ast.Object{Kind:3, Name:"URLLaunchParams", Decl:(*ast.TypeSpec)(0xc042081080), Data:interface {}(nil), Type:interface {}(nil)}

Comment:

(null doc)

Doc:

(null doc)

Parameters:

Name | Type | Description
--- | --- | ---
**url** | `string` | (null doc)

Result:

*empty*

## SaveVerdict

&ast.Object{Kind:3, Name:"SaveVerdictParams", Decl:(*ast.TypeSpec)(0xc042081140), Data:interface {}(nil), Type:interface {}(nil)}

Comment:

(null doc)

Doc:

(null doc)

Parameters:

Name | Type | Description
--- | --- | ---
**verdict** | `configurator.Verdict` | (null doc)

Result:

*empty*

## AllowSandboxSetup

&ast.Object{Kind:3, Name:"AllowSandboxSetupParams", Decl:(*ast.TypeSpec)(0xc042081200), Data:interface {}(nil), Type:interface {}(nil)}

Comment:

(null doc)

Doc:

(null doc)

Parameters:

*empty*

Result:

*empty*

## PrereqsFailed

&ast.Object{Kind:3, Name:"PrereqsFailedParams", Decl:(*ast.TypeSpec)(0xc0420814a0), Data:interface {}(nil), Type:interface {}(nil)}

Comment:

(null doc)

Doc:

(null doc)

Parameters:

Name | Type | Description
--- | --- | ---
**error** | `string` | (null doc)
**errorStack** | `string` | (null doc)

Result:

Name | Type | Description
--- | --- | ---
**continue** | `boolean` | (null doc)

## CleanDownloadsSearch

&ast.Object{Kind:3, Name:"CleanDownloadsSearchParams", Decl:(*ast.TypeSpec)(0xc042081560), Data:interface {}(nil), Type:interface {}(nil)}

Comment:

(null doc)

Doc:

(null doc)

Parameters:

Name | Type | Description
--- | --- | ---
**roots** | `string[]` | A list of folders to scan for potential subfolders to clean up 
**whitelist** | `string[]` | A list of subfolders to not consider when cleaning (staging folders for in-progress downloads) 

Result:

Name | Type | Description
--- | --- | ---
**entries** | `CleanDownloadsEntry[]` | (null doc)

## CleanDownloadsApply

&ast.Object{Kind:3, Name:"CleanDownloadsApplyParams", Decl:(*ast.TypeSpec)(0xc042081710), Data:interface {}(nil), Type:interface {}(nil)}

Comment:

(null doc)

Doc:

(null doc)

Parameters:

Name | Type | Description
--- | --- | ---
**entries** | `CleanDownloadsEntry[]` | (null doc)

Result:

*empty*



# Notifications

Notifications

## OperationProgress


Payload:

Name | Type | Description
--- | --- | ---
**progress** | `number` | (null doc)
**eta** | `number` | (null doc)
**bps** | `number` | (null doc)

## TaskStarted


Payload:

Name | Type | Description
--- | --- | ---
**reason** | `TaskReason` | (null doc)
**type** | `TaskType` | (null doc)
**game** | `itchio.Game` | (null doc)
**upload** | `itchio.Upload` | (null doc)
**build** | `itchio.Build` | (null doc)
**totalSize** | `number` | (null doc)

## TaskSucceeded


Payload:

Name | Type | Description
--- | --- | ---
**type** | `TaskType` | (null doc)
**installResult** | `InstallResult` | If the task installed something, then this contains info about the game, upload, build that were installed 

## GameUpdateAvailable


Payload:

Name | Type | Description
--- | --- | ---
**update** | `GameUpdate` | (null doc)

## LaunchRunning


Payload:

*empty*

## LaunchExited


Payload:

*empty*

## PrereqsStarted


Payload:

Name | Type | Description
--- | --- | ---
**tasks** | `Map<string, PrereqTask>` | (null doc)

## PrereqsTaskState


Payload:

Name | Type | Description
--- | --- | ---
**name** | `string` | (null doc)
**status** | `PrereqStatus` | (null doc)
**progress** | `number` | (null doc)
**eta** | `number` | (null doc)
**bps** | `number` | (null doc)

## PrereqsEnded


Payload:

*empty*

## Log


Payload:

Name | Type | Description
--- | --- | ---
**level** | `string` | (null doc)
**message** | `string` | (null doc)


