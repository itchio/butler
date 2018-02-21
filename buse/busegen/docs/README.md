# buse

> butler's JSON-RPC 2.0 service documentation

# Requests

Requests are essentially procedure calls: they're made asynchronously, and
a result is sent asynchronously. They may also fail, in which case
you get an error back, with details.

Some requests may complete almost instantly, and have an empty result
Still, waiting for the result lets you know that the peer has received
the request and processed it successfully.

Some requests are made by the client to butler (like CheckUpdate),
others are made from butler to the client (like AllowSandboxSetup)
## AllowSandboxSetup


Parameters:

*empty*

Result:

*empty*

## CheckUpdate


Parameters:

Name | Type | JSON Tag
--- | --- | ---
Items | &{%!s(token.Pos=4646) <nil> %!s(*ast.StarExpr=&{4648 0xc0420a8ba0})} | `json:"items"`

Result:

Name | Type | JSON Tag
--- | --- | ---
Updates | &{%!s(token.Pos=5051) <nil> %!s(*ast.StarExpr=&{5053 0xc0420a9040})} | `json:"updates"`
Warnings | &{%!s(token.Pos=5092) <nil> string} | `json:"warnings"`

## CleanDownloadsApply


Parameters:

Name | Type | JSON Tag
--- | --- | ---
Entries | &{%!s(token.Pos=8605) <nil> %!s(*ast.StarExpr=&{8607 0xc0420af080})} | `json:"entries"`

Result:

*empty*

## CleanDownloadsSearch


Parameters:

Name | Type | JSON Tag
--- | --- | ---
Roots | &{%!s(token.Pos=8203) <nil> string} | `json:"roots"`
Whitelist | &{%!s(token.Pos=8341) <nil> string} | `json:"whitelist"`

Result:

Name | Type | JSON Tag
--- | --- | ---
Entries | &{%!s(token.Pos=8422) <nil> %!s(*ast.StarExpr=&{8424 0xc0420aeec0})} | `json:"entries"`

## GameFindUploads


Parameters:

Name | Type | JSON Tag
--- | --- | ---
Game | &{%!s(token.Pos=1016) %!s(*ast.SelectorExpr=&{0xc04204aa40 0xc04204aa80})} | `json:"game"`
Credentials | &{%!s(token.Pos=1060) GameCredentials} | `json:"credentials"`

Result:

Name | Type | JSON Tag
--- | --- | ---
Uploads | &{%!s(token.Pos=1146) <nil> %!s(*ast.StarExpr=&{1148 0xc04204ac20})} | `json:"uploads"`

## GetReceipt


Parameters:

*empty*

Result:

Name | Type | JSON Tag
--- | --- | ---
Receipt | &{%!s(token.Pos=3029) %!s(*ast.SelectorExpr=&{0xc04204bc00 0xc04204bc20})} | `json:"receipt"`

## HTMLLaunch


Parameters:

Name | Type | JSON Tag
--- | --- | ---
RootFolder | string | `json:"rootFolder"`
IndexPath | string | `json:"indexPath"`
Args | &{%!s(token.Pos=6549) <nil> string} | `json:"args"`
Env | &{%!s(token.Pos=6587) string string} | `json:"env"`

Result:

*empty*

## Install


Parameters:

Name | Type | JSON Tag
--- | --- | ---
Game | &{%!s(token.Pos=2144) %!s(*ast.SelectorExpr=&{0xc04204b340 0xc04204b360})} | `json:"game"`
InstallFolder | string | `json:"installFolder"`
Upload | &{%!s(token.Pos=2245) %!s(*ast.SelectorExpr=&{0xc04204b460 0xc04204b480})} | `json:"upload"`
Build | &{%!s(token.Pos=2293) %!s(*ast.SelectorExpr=&{0xc04204b540 0xc04204b560})} | `json:"build"`
Credentials | &{%!s(token.Pos=2340) GameCredentials} | `json:"credentials"`
IgnoreInstallers | bool | `json:"ignoreInstallers,omitempty"`

Result:

Name | Type | JSON Tag
--- | --- | ---
Game | &{%!s(token.Pos=4316) %!s(*ast.SelectorExpr=&{0xc0420a8820 0xc0420a8840})} | `json:"game"`
Upload | &{%!s(token.Pos=4353) %!s(*ast.SelectorExpr=&{0xc0420a88e0 0xc0420a8900})} | `json:"upload"`
Build | &{%!s(token.Pos=4392) %!s(*ast.SelectorExpr=&{0xc0420a89a0 0xc0420a89c0})} | `json:"build"`

## Launch


Parameters:

Name | Type | JSON Tag
--- | --- | ---
InstallFolder | string | `json:"installFolder"`
Game | &{%!s(token.Pos=5652) %!s(*ast.SelectorExpr=&{0xc0420a9620 0xc0420a9640})} | `json:"game"`
Upload | &{%!s(token.Pos=5703) %!s(*ast.SelectorExpr=&{0xc0420a96e0 0xc0420a9700})} | `json:"upload"`
Build | &{%!s(token.Pos=5756) %!s(*ast.SelectorExpr=&{0xc0420a97c0 0xc0420a97e0})} | `json:"build"`
Verdict | &{%!s(token.Pos=5808) %!s(*ast.SelectorExpr=&{0xc0420a9880 0xc0420a98a0})} | `json:"verdict"`
PrereqsDir | string | `json:"prereqsDir"`
ForcePrereqs | bool | `json:"forcePrereqs,omitempty"`
Sandbox | bool | `json:"sandbox,omitempty"`
Credentials | &{%!s(token.Pos=6021) GameCredentials} | `json:"credentials"`

Result:

*empty*

## OperationCancel


Parameters:

Name | Type | JSON Tag
--- | --- | ---
ID | string | `json:"id"`

Result:

*empty*

## OperationStart


Parameters:

Name | Type | JSON Tag
--- | --- | ---
ID | string | `json:"id"`
StagingFolder | string | `json:"stagingFolder"`
Operation | Operation | `json:"operation"`
InstallParams | &{%!s(token.Pos=1767) InstallParams} | `json:"installParams,omitempty"`
UninstallParams | &{%!s(token.Pos=1834) UninstallParams} | `json:"uninstallParams,omitempty"`

Result:

*empty*

## PickManifestAction


Parameters:

Name | Type | JSON Tag
--- | --- | ---
Actions | &{%!s(token.Pos=6221) <nil> %!s(*ast.StarExpr=&{6223 0xc0420a9c60})} | `json:"actions"`

Result:

Name | Type | JSON Tag
--- | --- | ---
Name | string | `json:"name"`

## PickUpload


Parameters:

Name | Type | JSON Tag
--- | --- | ---
Uploads | &{%!s(token.Pos=2845) <nil> %!s(*ast.StarExpr=&{2847 0xc04204ba20})} | `json:"uploads"`

Result:

Name | Type | JSON Tag
--- | --- | ---
Index | int64 | `json:"index"`

## PrereqsFailed


Parameters:

Name | Type | JSON Tag
--- | --- | ---
Error | string | `json:"error"`
ErrorStack | string | `json:"errorStack"`

Result:

Name | Type | JSON Tag
--- | --- | ---
Continue | bool | `json:"continue"`

## SaveVerdict


Parameters:

Name | Type | JSON Tag
--- | --- | ---
Verdict | &{%!s(token.Pos=6785) %!s(*ast.SelectorExpr=&{0xc0420ae1c0 0xc0420ae1e0})} | `json:"verdict"`

Result:

*empty*

## ShellLaunch


Parameters:

Name | Type | JSON Tag
--- | --- | ---
ItemPath | string | `json:"itemPath"`

Result:

*empty*

## URLLaunch


Parameters:

Name | Type | JSON Tag
--- | --- | ---
URL | string | `json:"url"`

Result:

*empty*

## Uninstall


Parameters:

Name | Type | JSON Tag
--- | --- | ---
InstallFolder | string | `json:"installFolder"`

Result:

*empty*

## VersionGet


Parameters:

*empty*

Result:

Name | Type | JSON Tag
--- | --- | ---
Version | string | `json:"version"`
VersionString | string | `json:"versionString"`


# Notifications

## GameUpdateAvailable


Payload:

Name | Type | JSON Tag
--- | --- | ---
Update | &{%!s(token.Pos=5181) GameUpdate} | `json:"update"`

## LaunchExited


Payload:

*empty*

## LaunchRunning


Payload:

*empty*

## Log


Payload:

Name | Type | JSON Tag
--- | --- | ---
Level | string | `json:"level"`
Message | string | `json:"message"`

## OperationProgress


Payload:

Name | Type | JSON Tag
--- | --- | ---
Progress | float64 | `json:"progress"`
ETA | float64 | `json:"eta"`
BPS | float64 | `json:"bps"`

## PrereqsEnded


Payload:

*empty*

## PrereqsStarted


Payload:

Name | Type | JSON Tag
--- | --- | ---
Tasks | &{%!s(token.Pos=7016) string %!s(*ast.StarExpr=&{7027 0xc0420ae420})} | `json:"tasks"`

## PrereqsTaskState


Payload:

Name | Type | JSON Tag
--- | --- | ---
Name | string | `json:"name"`
Status | PrereqStatus | `json:"status"`
Progress | float64 | `json:"progress"`
ETA | float64 | `json:"eta"`
BPS | float64 | `json:"bps"`

## TaskStarted


Payload:

Name | Type | JSON Tag
--- | --- | ---
Reason | TaskReason | `json:"reason"`
Type | TaskType | `json:"type"`
Game | &{%!s(token.Pos=3782) %!s(*ast.SelectorExpr=&{0xc0420a8300 0xc0420a8320})} | `json:"game"`
Upload | &{%!s(token.Pos=3822) %!s(*ast.SelectorExpr=&{0xc0420a83e0 0xc0420a8400})} | `json:"upload"`
Build | &{%!s(token.Pos=3864) %!s(*ast.SelectorExpr=&{0xc0420a84a0 0xc0420a84c0})} | `json:"build,omitempty"`
TotalSize | int64 | `json:"totalSize,omitempty"`

## TaskSucceeded


Payload:

Name | Type | JSON Tag
--- | --- | ---
Type | TaskType | `json:"type"`
InstallResult | &{%!s(token.Pos=4161) InstallResult} | `json:"installResult,omitempty"`

