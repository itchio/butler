# Caution: this is a draft

!> This document is a draft! It should not be used yet for implementing
   clients for buse. The API and recommendations are still subject to change.

# Overview

buse is butler's JSON-RPC 2.0 service. It is used heavily by the [itch.io app](https://itch.io/app)
as of version v25.

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


<p>
<p>Retrieves the version of the butler instance the client
is connected to.</p>

<p>This endpoint is meant to gather information when reporting
issues, rather than feature sniffing. Conforming clients should
automatically download new versions of butler, see the <strong>Updating</strong> section.</p>

</p>

<p>
<span class="header">Parameters</span> <em>none</em>
</p>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>version</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Something short, like <code>v8.0.0</code></p>
</td>
</tr>
<tr>
<td><code>versionString</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Something long, like <code>v8.0.0, built on Aug 27 2017 @ 01:13:55, ref d833cc0aeea81c236c81dffb27bc18b2b8d8b290</code></p>
</td>
</tr>
</table>


<div id="VersionGetParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-client-caller"></em>Version.Get <a href="#/?id=versionget">(Go to definition)</a></p>

<p>
<p>Retrieves the version of the butler instance the client
is connected to.</p>

<p>This endpoint is meant to gather information when reporting
issues, rather than feature sniffing. Conforming clients should
automatically download new versions of butler, see the <strong>Updating</strong> section.</p>

</p>
</div>


## Profile

### <em class="request-client-caller"></em>Profile.List


<p>
<p>Lists remembered profiles</p>

</p>

<p>
<span class="header">Parameters</span> <em>none</em>
</p>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>profiles</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Profile__TypeHint">Profile</span>[]</code></td>
<td><p>A list of remembered profiles</p>
</td>
</tr>
</table>


<div id="ProfileListParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-client-caller"></em>Profile.List <a href="#/?id=profilelist">(Go to definition)</a></p>

<p>
<p>Lists remembered profiles</p>

</p>
</div>

### <em class="request-client-caller"></em>Profile.LoginWithPassword


<p>
<p>Add a new profile by password login</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>username</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>The username (or e-mail) to use for login</p>
</td>
</tr>
<tr>
<td><code>password</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>The password to use</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>profile</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Profile__TypeHint">Profile</span></code></td>
<td><p>Information for the new profile, now remembered</p>
</td>
</tr>
<tr>
<td><code>cookie</code></td>
<td><code class="typename">Map&lt;<span class="type builtin-type">string</span>, <span class="type builtin-type">string</span>&gt;</code></td>
<td><p>Profile cookie for website</p>
</td>
</tr>
</table>


<div id="ProfileLoginWithPasswordParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-client-caller"></em>Profile.LoginWithPassword <a href="#/?id=profileloginwithpassword">(Go to definition)</a></p>

<p>
<p>Add a new profile by password login</p>

</p>

<table class="field-table">
<tr>
<td><code>username</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>password</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### <em class="request-server-caller"></em>Profile.RequestCaptcha


<p>
<p>Ask the user to solve a captcha challenge
Sent during <code class="typename"><span class="type request-client-caller" data-tip-selector="#ProfileLoginWithPasswordParams__TypeHint">Profile.LoginWithPassword</span></code> if certain
conditions are met.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>recaptchaUrl</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Address of page containing a recaptcha widget</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>recaptchaResponse</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>The response given by recaptcha after it&rsquo;s been filled</p>
</td>
</tr>
</table>


<div id="ProfileRequestCaptchaParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-server-caller"></em>Profile.RequestCaptcha <a href="#/?id=profilerequestcaptcha">(Go to definition)</a></p>

<p>
<p>Ask the user to solve a captcha challenge
Sent during <code class="typename"><span class="type request-client-caller">Profile.LoginWithPassword</span></code> if certain
conditions are met.</p>

</p>

<table class="field-table">
<tr>
<td><code>recaptchaUrl</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### <em class="request-server-caller"></em>Profile.RequestTOTP


<p>
<p>Ask the user to provide a TOTP token.
Sent during <code class="typename"><span class="type request-client-caller" data-tip-selector="#ProfileLoginWithPasswordParams__TypeHint">Profile.LoginWithPassword</span></code> if the user has
two-factor authentication enabled.</p>

</p>

<p>
<span class="header">Parameters</span> <em>none</em>
</p>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>code</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>The TOTP code entered by the user</p>
</td>
</tr>
</table>


<div id="ProfileRequestTOTPParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-server-caller"></em>Profile.RequestTOTP <a href="#/?id=profilerequesttotp">(Go to definition)</a></p>

<p>
<p>Ask the user to provide a TOTP token.
Sent during <code class="typename"><span class="type request-client-caller">Profile.LoginWithPassword</span></code> if the user has
two-factor authentication enabled.</p>

</p>
</div>

### <em class="request-client-caller"></em>Profile.UseSavedLogin


<p>
<p>Use saved login credentials to validate a profile.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>profileId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>profile</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Profile__TypeHint">Profile</span></code></td>
<td><p>Information for the now validated profile</p>
</td>
</tr>
</table>


<div id="ProfileUseSavedLoginParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-client-caller"></em>Profile.UseSavedLogin <a href="#/?id=profileusesavedlogin">(Go to definition)</a></p>

<p>
<p>Use saved login credentials to validate a profile.</p>

</p>

<table class="field-table">
<tr>
<td><code>profileId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### <em class="request-client-caller"></em>Profile.Forget


<p>
<p>Forgets a remembered profile - it won&rsquo;t appear in the
<code class="typename"><span class="type request-client-caller" data-tip-selector="#ProfileListParams__TypeHint">Profile.List</span></code> results anymore.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>profileId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>success</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>True if the profile did exist (and was successfully forgotten)</p>
</td>
</tr>
</table>


<div id="ProfileForgetParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-client-caller"></em>Profile.Forget <a href="#/?id=profileforget">(Go to definition)</a></p>

<p>
<p>Forgets a remembered profile - it won&rsquo;t appear in the
<code class="typename"><span class="type request-client-caller">Profile.List</span></code> results anymore.</p>

</p>

<table class="field-table">
<tr>
<td><code>profileId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>


## Fetch

### <em class="request-client-caller"></em>Fetch.Game


<p>
<p>Fetches information for an itch.io game.</p>

<p>Sends <code class="typename"><span class="type notification" data-tip-selector="#FetchGameYieldNotification__TypeHint">Fetch.Game.Yield</span></code> twice at most: first from cache,
second from API if we&rsquo;re online.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>profileId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Profile to use to fetch game</p>
</td>
</tr>
<tr>
<td><code>gameId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Identifier of game to look for</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> <em>none</em>
</p>


<div id="FetchGameParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-client-caller"></em>Fetch.Game <a href="#/?id=fetchgame">(Go to definition)</a></p>

<p>
<p>Fetches information for an itch.io game.</p>

<p>Sends <code class="typename"><span class="type notification">Fetch.Game.Yield</span></code> twice at most: first from cache,
second from API if we&rsquo;re online.</p>

</p>

<table class="field-table">
<tr>
<td><code>profileId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>gameId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### <em class="notification"></em>Fetch.Game.Yield


<p>
<p>Sent during <code class="typename"><span class="type request-client-caller" data-tip-selector="#FetchGameParams__TypeHint">Fetch.Game</span></code> whenever a result is
available.</p>

</p>

<p>
<span class="header">Payload</span> 
</p>


<table class="field-table">
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td><p>Current result for game fetching (from local DB, or API, etc.)</p>
</td>
</tr>
</table>


<div id="FetchGameYieldNotification__TypeHint" style="display: none;" class="tip-content">
<p><em class="notification"></em>Fetch.Game.Yield <a href="#/?id=fetchgameyield">(Go to definition)</a></p>

<p>
<p>Sent during <code class="typename"><span class="type request-client-caller">Fetch.Game</span></code> whenever a result is
available.</p>

</p>

<table class="field-table">
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type struct-type">Game</span></code></td>
</tr>
</table>

</div>

### <em class="request-client-caller"></em>Fetch.Collection


<p>
<p>Fetches information about a collection and the games it
contains.</p>

<p>Sends <code class="typename"><span class="type notification" data-tip-selector="#FetchCollectionYieldNotification__TypeHint">Fetch.Collection.Yield</span></code>.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>profileId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Profile to use to fetch game</p>
</td>
</tr>
<tr>
<td><code>collectionId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Identifier of the collection to look for</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> <em>none</em>
</p>


<div id="FetchCollectionParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-client-caller"></em>Fetch.Collection <a href="#/?id=fetchcollection">(Go to definition)</a></p>

<p>
<p>Fetches information about a collection and the games it
contains.</p>

<p>Sends <code class="typename"><span class="type notification">Fetch.Collection.Yield</span></code>.</p>

</p>

<table class="field-table">
<tr>
<td><code>profileId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>collectionId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### <em class="notification"></em>Fetch.Collection.Yield


<p>
<p>Contains general info about a collection</p>

</p>

<p>
<span class="header">Payload</span> 
</p>


<table class="field-table">
<tr>
<td><code>collection</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Collection__TypeHint">Collection</span></code></td>
<td></td>
</tr>
</table>


<div id="FetchCollectionYieldNotification__TypeHint" style="display: none;" class="tip-content">
<p><em class="notification"></em>Fetch.Collection.Yield <a href="#/?id=fetchcollectionyield">(Go to definition)</a></p>

<p>
<p>Contains general info about a collection</p>

</p>

<table class="field-table">
<tr>
<td><code>collection</code></td>
<td><code class="typename"><span class="type struct-type">Collection</span></code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>CollectionGame


<p>
<p>Association between a <code class="typename"><span class="type struct-type" data-tip-selector="#Game__TypeHint">Game</span></code> and a <code class="typename"><span class="type struct-type" data-tip-selector="#Collection__TypeHint">Collection</span></code></p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>position</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Position in collection, use if you want to display them in the
canonical itch.io order</p>
</td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td></td>
</tr>
</table>


<div id="CollectionGame__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>CollectionGame <a href="#/?id=collectiongame">(Go to definition)</a></p>

<p>
<p>Association between a <code class="typename"><span class="type struct-type">Game</span></code> and a <code class="typename"><span class="type struct-type">Collection</span></code></p>

</p>

<table class="field-table">
<tr>
<td><code>position</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type struct-type">Game</span></code></td>
</tr>
</table>

</div>

### <em class="request-client-caller"></em>Fetch.ProfileCollections



<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>profileId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Profile to use to fetch game</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> <em>none</em>
</p>


<div id="FetchProfileCollectionsParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-client-caller"></em>Fetch.ProfileCollections <a href="#/?id=fetchprofilecollections">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>profileId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### <em class="notification"></em>Fetch.ProfileCollections.Yield


<p>
<p>Sent during <code class="typename"><span class="type request-client-caller" data-tip-selector="#FetchProfileCollectionsParams__TypeHint">Fetch.ProfileCollections</span></code> whenever new info is
available.</p>

</p>

<p>
<span class="header">Payload</span> 
</p>


<table class="field-table">
<tr>
<td><code>offset</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>total</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>items</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Collection__TypeHint">Collection</span>[]</code></td>
<td></td>
</tr>
</table>


<div id="FetchProfileCollectionsYieldNotification__TypeHint" style="display: none;" class="tip-content">
<p><em class="notification"></em>Fetch.ProfileCollections.Yield <a href="#/?id=fetchprofilecollectionsyield">(Go to definition)</a></p>

<p>
<p>Sent during <code class="typename"><span class="type request-client-caller">Fetch.ProfileCollections</span></code> whenever new info is
available.</p>

</p>

<table class="field-table">
<tr>
<td><code>offset</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>total</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>items</code></td>
<td><code class="typename"><span class="type struct-type">Collection</span>[]</code></td>
</tr>
</table>

</div>

### <em class="request-client-caller"></em>Fetch.ProfileGames



<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>profileId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Profile to use to fetch game</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> <em>none</em>
</p>


<div id="FetchProfileGamesParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-client-caller"></em>Fetch.ProfileGames <a href="#/?id=fetchprofilegames">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>profileId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### <em class="notification"></em>Fetch.ProfileGames.Yield



<p>
<span class="header">Payload</span> 
</p>


<table class="field-table">
<tr>
<td><code>offset</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>total</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>items</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#ProfileGame__TypeHint">ProfileGame</span>[]</code></td>
<td></td>
</tr>
</table>


<div id="FetchProfileGamesYieldNotification__TypeHint" style="display: none;" class="tip-content">
<p><em class="notification"></em>Fetch.ProfileGames.Yield <a href="#/?id=fetchprofilegamesyield">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>offset</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>total</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>items</code></td>
<td><code class="typename"><span class="type struct-type">ProfileGame</span>[]</code></td>
</tr>
</table>

</div>

### <em class="request-client-caller"></em>Fetch.ProfileOwnedKeys



<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>profileId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Profile to use to fetch game</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> <em>none</em>
</p>


<div id="FetchProfileOwnedKeysParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-client-caller"></em>Fetch.ProfileOwnedKeys <a href="#/?id=fetchprofileownedkeys">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>profileId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### <em class="notification"></em>Fetch.ProfileOwnedKeys.Yield



<p>
<span class="header">Payload</span> 
</p>


<table class="field-table">
<tr>
<td><code>offset</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>total</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>items</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#DownloadKey__TypeHint">DownloadKey</span>[]</code></td>
<td></td>
</tr>
</table>


<div id="FetchProfileOwnedKeysYieldNotification__TypeHint" style="display: none;" class="tip-content">
<p><em class="notification"></em>Fetch.ProfileOwnedKeys.Yield <a href="#/?id=fetchprofileownedkeysyield">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>offset</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>total</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>items</code></td>
<td><code class="typename"><span class="type struct-type">DownloadKey</span>[]</code></td>
</tr>
</table>

</div>

### <em class="request-client-caller"></em>Fetch.Commons



<p>
<span class="header">Parameters</span> <em>none</em>
</p>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>downloadKeys</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#DownloadKeySummary__TypeHint">DownloadKeySummary</span>[]</code></td>
<td></td>
</tr>
<tr>
<td><code>caves</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#CaveSummary__TypeHint">CaveSummary</span>[]</code></td>
<td></td>
</tr>
<tr>
<td><code>installLocations</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#InstallLocationSummary__TypeHint">InstallLocationSummary</span>[]</code></td>
<td></td>
</tr>
</table>


<div id="FetchCommonsParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-client-caller"></em>Fetch.Commons <a href="#/?id=fetchcommons">(Go to definition)</a></p>

</div>

### <em class="request-client-caller"></em>Fetch.Caves


<p>
<p>Retrieve info for all caves.</p>

</p>

<p>
<span class="header">Parameters</span> <em>none</em>
</p>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>caves</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Cave__TypeHint">Cave</span>[]</code></td>
<td></td>
</tr>
</table>


<div id="FetchCavesParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-client-caller"></em>Fetch.Caves <a href="#/?id=fetchcaves">(Go to definition)</a></p>

<p>
<p>Retrieve info for all caves.</p>

</p>
</div>

### <em class="request-client-caller"></em>Fetch.Cave


<p>
<p>Retrieve info on a cave by ID.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>caveId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>cave</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Cave__TypeHint">Cave</span></code></td>
<td></td>
</tr>
</table>


<div id="FetchCaveParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-client-caller"></em>Fetch.Cave <a href="#/?id=fetchcave">(Go to definition)</a></p>

<p>
<p>Retrieve info on a cave by ID.</p>

</p>

<table class="field-table">
<tr>
<td><code>caveId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### <em class="request-client-caller"></em>Fetch.CavesByGameID


<p>
<p>Retrieve all caves for a given game.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>gameId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>caves</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Cave__TypeHint">Cave</span>[]</code></td>
<td></td>
</tr>
</table>


<div id="FetchCavesByGameIDParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-client-caller"></em>Fetch.CavesByGameID <a href="#/?id=fetchcavesbygameid">(Go to definition)</a></p>

<p>
<p>Retrieve all caves for a given game.</p>

</p>

<table class="field-table">
<tr>
<td><code>gameId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### <em class="request-client-caller"></em>Fetch.CavesByInstallLocationID


<p>
<p>Retrieve all caves installed to a given location.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>installLocationId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>installLocationPath</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
<tr>
<td><code>installLocationSize</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>caves</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Cave__TypeHint">Cave</span>[]</code></td>
<td></td>
</tr>
</table>


<div id="FetchCavesByInstallLocationIDParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-client-caller"></em>Fetch.CavesByInstallLocationID <a href="#/?id=fetchcavesbyinstalllocationid">(Go to definition)</a></p>

<p>
<p>Retrieve all caves installed to a given location.</p>

</p>

<table class="field-table">
<tr>
<td><code>installLocationId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>


## Install

### <em class="request-client-caller"></em>Game.FindUploads


<p>
<p>Finds uploads compatible with the current runtime, for a given game.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td><p>Which game to find uploads for</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>uploads</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Upload__TypeHint">Upload</span>[]</code></td>
<td><p>A list of uploads that were found to be compatible.</p>
</td>
</tr>
</table>


<div id="GameFindUploadsParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-client-caller"></em>Game.FindUploads <a href="#/?id=gamefinduploads">(Go to definition)</a></p>

<p>
<p>Finds uploads compatible with the current runtime, for a given game.</p>

</p>

<table class="field-table">
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type struct-type">Game</span></code></td>
</tr>
</table>

</div>

### <em class="request-client-caller"></em>Install.Queue


<p>
<p>Queues an install operation to be later performed
via <code class="typename"><span class="type request-client-caller" data-tip-selector="#InstallPerformParams__TypeHint">Install.Perform</span></code>.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>caveId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p><span class="tag">Optional</span> ID of the cave to perform the install for.
If not specified, will create a new cave.</p>
</td>
</tr>
<tr>
<td><code>installLocationId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p><span class="tag">Optional</span> If CaveID is not specified, ID of an install location
to install to.</p>
</td>
</tr>
<tr>
<td><code>noCave</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span> If set, InstallFolder can be set and no cave
record will be read or modified</p>
</td>
</tr>
<tr>
<td><code>installFolder</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p><span class="tag">Optional</span> When NoCave is set, exactly where to install</p>
</td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td><p><span class="tag">Optional</span> Which game to install.</p>

<p>If unspecified and caveId is specified, the same game will be used.</p>
</td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Upload__TypeHint">Upload</span></code></td>
<td><p><span class="tag">Optional</span> Which upload to install.</p>

<p>If unspecified and caveId is specified, the same upload will be used.</p>
</td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Build__TypeHint">Build</span></code></td>
<td><p><span class="tag">Optional</span> Which build to install</p>

<p>If unspecified and caveId is specified, the same build will be used.</p>
</td>
</tr>
<tr>
<td><code>ignoreInstallers</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span> If true, do not run windows installers, just extract
whatever to the install folder.</p>
</td>
</tr>
<tr>
<td><code>stagingFolder</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p><span class="tag">Optional</span> A folder that butler can use to store temporary files, like
partial downloads, checkpoint files, etc.</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>caveId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td></td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Upload__TypeHint">Upload</span></code></td>
<td></td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Build__TypeHint">Build</span></code></td>
<td></td>
</tr>
<tr>
<td><code>installFolder</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
<tr>
<td><code>stagingFolder</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
</table>


<div id="InstallQueueParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-client-caller"></em>Install.Queue <a href="#/?id=installqueue">(Go to definition)</a></p>

<p>
<p>Queues an install operation to be later performed
via <code class="typename"><span class="type request-client-caller">Install.Perform</span></code>.</p>

</p>

<table class="field-table">
<tr>
<td><code>caveId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>installLocationId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>noCave</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>installFolder</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type struct-type">Game</span></code></td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type struct-type">Upload</span></code></td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type struct-type">Build</span></code></td>
</tr>
<tr>
<td><code>ignoreInstallers</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>stagingFolder</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### <em class="request-client-caller"></em>Install.Perform


<p>
<p>Perform an install that was previously queued via
<code class="typename"><span class="type request-client-caller" data-tip-selector="#InstallQueueParams__TypeHint">Install.Queue</span></code>.</p>

<p>Can be cancelled by passing the same <code>ID</code> to <code class="typename"><span class="type request-client-caller" data-tip-selector="#InstallCancelParams__TypeHint">Install.Cancel</span></code>.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>ID that can be later used in <code class="typename"><span class="type request-client-caller" data-tip-selector="#InstallCancelParams__TypeHint">Install.Cancel</span></code></p>
</td>
</tr>
<tr>
<td><code>stagingFolder</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>The folder turned by <code class="typename"><span class="type request-client-caller" data-tip-selector="#InstallQueueParams__TypeHint">Install.Queue</span></code></p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> <em>none</em>
</p>


<div id="InstallPerformParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-client-caller"></em>Install.Perform <a href="#/?id=installperform">(Go to definition)</a></p>

<p>
<p>Perform an install that was previously queued via
<code class="typename"><span class="type request-client-caller">Install.Queue</span></code>.</p>

<p>Can be cancelled by passing the same <code>ID</code> to <code class="typename"><span class="type request-client-caller">Install.Cancel</span></code>.</p>

</p>

<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>stagingFolder</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### <em class="request-client-caller"></em>Install.Cancel


<p>
<p>Attempt to gracefully cancel an ongoing operation.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>The UUID of the task to cancel, as passed to <code class="typename"><span class="type builtin-type">OperationStartParams</span></code></p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> <em>none</em>
</p>


<div id="InstallCancelParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-client-caller"></em>Install.Cancel <a href="#/?id=installcancel">(Go to definition)</a></p>

<p>
<p>Attempt to gracefully cancel an ongoing operation.</p>

</p>

<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### <em class="request-client-caller"></em>Uninstall.Perform


<p>
<p>UninstallParams contains all the parameters needed to perform
an uninstallation for a game via <code class="typename"><span class="type builtin-type">OperationStartParams</span></code>.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>caveId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>The cave to uninstall</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> <em>none</em>
</p>


<div id="UninstallPerformParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-client-caller"></em>Uninstall.Perform <a href="#/?id=uninstallperform">(Go to definition)</a></p>

<p>
<p>UninstallParams contains all the parameters needed to perform
an uninstallation for a game via <code class="typename"><span class="type builtin-type">OperationStartParams</span></code>.</p>

</p>

<table class="field-table">
<tr>
<td><code>caveId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### <em class="request-client-caller"></em>Install.VersionSwitch.Queue


<p>
<p>Prepare to queue a version switch. The client will
receive an <code class="typename"><span class="type request-server-caller" data-tip-selector="#InstallVersionSwitchPickParams__TypeHint">InstallVersionSwitchPick</span></code>.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>caveId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>The cave to switch to a different version</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> <em>none</em>
</p>


<div id="InstallVersionSwitchQueueParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-client-caller"></em>Install.VersionSwitch.Queue <a href="#/?id=installversionswitchqueue">(Go to definition)</a></p>

<p>
<p>Prepare to queue a version switch. The client will
receive an <code class="typename"><span class="type request-server-caller">InstallVersionSwitchPick</span></code>.</p>

</p>

<table class="field-table">
<tr>
<td><code>caveId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### <em class="request-server-caller"></em>InstallVersionSwitchPick


<p>
<p>Let the user pick which version to switch to.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>cave</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Cave__TypeHint">Cave</span></code></td>
<td></td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Upload__TypeHint">Upload</span></code></td>
<td></td>
</tr>
<tr>
<td><code>builds</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Build__TypeHint">Build</span>[]</code></td>
<td></td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>index</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>A negative index aborts the version switch</p>
</td>
</tr>
</table>


<div id="InstallVersionSwitchPickParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-server-caller"></em>InstallVersionSwitchPick <a href="#/?id=installversionswitchpick">(Go to definition)</a></p>

<p>
<p>Let the user pick which version to switch to.</p>

</p>

<table class="field-table">
<tr>
<td><code>cave</code></td>
<td><code class="typename"><span class="type struct-type">Cave</span></code></td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type struct-type">Upload</span></code></td>
</tr>
<tr>
<td><code>builds</code></td>
<td><code class="typename"><span class="type struct-type">Build</span>[]</code></td>
</tr>
</table>

</div>

### <em class="request-server-caller"></em>PickUpload


<p>
<p>Asks the user to pick between multiple available uploads</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>uploads</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Upload__TypeHint">Upload</span>[]</code></td>
<td><p>An array of upload objects to choose from</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>index</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>The index (in the original array) of the upload that was picked,
or a negative value to cancel.</p>
</td>
</tr>
</table>


<div id="PickUploadParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-server-caller"></em>PickUpload <a href="#/?id=pickupload">(Go to definition)</a></p>

<p>
<p>Asks the user to pick between multiple available uploads</p>

</p>

<table class="field-table">
<tr>
<td><code>uploads</code></td>
<td><code class="typename"><span class="type struct-type">Upload</span>[]</code></td>
</tr>
</table>

</div>

### <em class="notification"></em>Progress


<p>
<p>Sent periodically during <code class="typename"><span class="type request-client-caller" data-tip-selector="#InstallPerformParams__TypeHint">Install.Perform</span></code> to inform on the current state of an install</p>

</p>

<p>
<span class="header">Payload</span> 
</p>


<table class="field-table">
<tr>
<td><code>progress</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>An overall progress value between 0 and 1</p>
</td>
</tr>
<tr>
<td><code>eta</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Estimated completion time for the operation, in seconds (floating)</p>
</td>
</tr>
<tr>
<td><code>bps</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Network bandwidth used, in bytes per second (floating)</p>
</td>
</tr>
</table>


<div id="ProgressNotification__TypeHint" style="display: none;" class="tip-content">
<p><em class="notification"></em>Progress <a href="#/?id=progress">(Go to definition)</a></p>

<p>
<p>Sent periodically during <code class="typename"><span class="type request-client-caller">Install.Perform</span></code> to inform on the current state of an install</p>

</p>

<table class="field-table">
<tr>
<td><code>progress</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>eta</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>bps</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### <em class="enum-type"></em>TaskReason



<p>
<span class="header">Values</span> 
</p>


<table class="field-table">
<tr>
<td><code>"install"</code></td>
<td><p>Task was started for an install operation</p>
</td>
</tr>
<tr>
<td><code>"uninstall"</code></td>
<td><p>Task was started for an uninstall operation</p>
</td>
</tr>
</table>


<div id="TaskReason__TypeHint" style="display: none;" class="tip-content">
<p><em class="enum-type"></em>TaskReason <a href="#/?id=taskreason">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>"install"</code></td>
</tr>
<tr>
<td><code>"uninstall"</code></td>
</tr>
</table>

</div>

### <em class="enum-type"></em>TaskType



<p>
<span class="header">Values</span> 
</p>


<table class="field-table">
<tr>
<td><code>"download"</code></td>
<td><p>We&rsquo;re fetching files from a remote server</p>
</td>
</tr>
<tr>
<td><code>"install"</code></td>
<td><p>We&rsquo;re running an installer</p>
</td>
</tr>
<tr>
<td><code>"uninstall"</code></td>
<td><p>We&rsquo;re running an uninstaller</p>
</td>
</tr>
<tr>
<td><code>"update"</code></td>
<td><p>We&rsquo;re applying some patches</p>
</td>
</tr>
<tr>
<td><code>"heal"</code></td>
<td><p>We&rsquo;re healing from a signature and heal source</p>
</td>
</tr>
</table>


<div id="TaskType__TypeHint" style="display: none;" class="tip-content">
<p><em class="enum-type"></em>TaskType <a href="#/?id=tasktype">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>"download"</code></td>
</tr>
<tr>
<td><code>"install"</code></td>
</tr>
<tr>
<td><code>"uninstall"</code></td>
</tr>
<tr>
<td><code>"update"</code></td>
</tr>
<tr>
<td><code>"heal"</code></td>
</tr>
</table>

</div>

### <em class="notification"></em>TaskStarted


<p>
<p>Each operation is made up of one or more tasks. This notification
is sent during <code class="typename"><span class="type builtin-type">OperationStartParams</span></code> whenever a specific task starts.</p>

</p>

<p>
<span class="header">Payload</span> 
</p>


<table class="field-table">
<tr>
<td><code>reason</code></td>
<td><code class="typename"><span class="type enum-type" data-tip-selector="#TaskReason__TypeHint">TaskReason</span></code></td>
<td><p>Why this task was started</p>
</td>
</tr>
<tr>
<td><code>type</code></td>
<td><code class="typename"><span class="type enum-type" data-tip-selector="#TaskType__TypeHint">TaskType</span></code></td>
<td><p>Is this task a download? An install?</p>
</td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td><p>The game this task is dealing with</p>
</td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Upload__TypeHint">Upload</span></code></td>
<td><p>The upload this task is dealing with</p>
</td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Build__TypeHint">Build</span></code></td>
<td><p>The build this task is dealing with (if any)</p>
</td>
</tr>
<tr>
<td><code>totalSize</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Total size in bytes</p>
</td>
</tr>
</table>


<div id="TaskStartedNotification__TypeHint" style="display: none;" class="tip-content">
<p><em class="notification"></em>TaskStarted <a href="#/?id=taskstarted">(Go to definition)</a></p>

<p>
<p>Each operation is made up of one or more tasks. This notification
is sent during <code class="typename"><span class="type builtin-type">OperationStartParams</span></code> whenever a specific task starts.</p>

</p>

<table class="field-table">
<tr>
<td><code>reason</code></td>
<td><code class="typename"><span class="type enum-type">TaskReason</span></code></td>
</tr>
<tr>
<td><code>type</code></td>
<td><code class="typename"><span class="type enum-type">TaskType</span></code></td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type struct-type">Game</span></code></td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type struct-type">Upload</span></code></td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type struct-type">Build</span></code></td>
</tr>
<tr>
<td><code>totalSize</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### <em class="notification"></em>TaskSucceeded


<p>
<p>Sent during <code class="typename"><span class="type builtin-type">OperationStartParams</span></code> whenever a task succeeds for an operation.</p>

</p>

<p>
<span class="header">Payload</span> 
</p>


<table class="field-table">
<tr>
<td><code>type</code></td>
<td><code class="typename"><span class="type enum-type" data-tip-selector="#TaskType__TypeHint">TaskType</span></code></td>
<td></td>
</tr>
<tr>
<td><code>installResult</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#InstallResult__TypeHint">InstallResult</span></code></td>
<td><p>If the task installed something, then this contains
info about the game, upload, build that were installed</p>
</td>
</tr>
</table>


<div id="TaskSucceededNotification__TypeHint" style="display: none;" class="tip-content">
<p><em class="notification"></em>TaskSucceeded <a href="#/?id=tasksucceeded">(Go to definition)</a></p>

<p>
<p>Sent during <code class="typename"><span class="type builtin-type">OperationStartParams</span></code> whenever a task succeeds for an operation.</p>

</p>

<table class="field-table">
<tr>
<td><code>type</code></td>
<td><code class="typename"><span class="type enum-type">TaskType</span></code></td>
</tr>
<tr>
<td><code>installResult</code></td>
<td><code class="typename"><span class="type struct-type">InstallResult</span></code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>InstallResult


<p>
<p>What was installed by a subtask of <code class="typename"><span class="type builtin-type">OperationStartParams</span></code>.</p>

<p>See <code class="typename"><span class="type notification" data-tip-selector="#TaskSucceededNotification__TypeHint">TaskSucceeded</span></code>.</p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td><p>The game we installed</p>
</td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Upload__TypeHint">Upload</span></code></td>
<td><p>The upload we installed</p>
</td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Build__TypeHint">Build</span></code></td>
<td><p><span class="tag">Optional</span> The build we installed</p>
</td>
</tr>
</table>


<div id="InstallResult__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>InstallResult <a href="#/?id=installresult">(Go to definition)</a></p>

<p>
<p>What was installed by a subtask of <code class="typename"><span class="type builtin-type">OperationStartParams</span></code>.</p>

<p>See <code class="typename"><span class="type notification">TaskSucceeded</span></code>.</p>

</p>

<table class="field-table">
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type struct-type">Game</span></code></td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type struct-type">Upload</span></code></td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type struct-type">Build</span></code></td>
</tr>
</table>

</div>


## Downloads

### <em class="request-client-caller"></em>Downloads.Queue



<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>item</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#InstallQueueResult__TypeHint">InstallQueue</span></code></td>
<td></td>
</tr>
</table>



<p>
<span class="header">Result</span> <em>none</em>
</p>


<div id="DownloadsQueueParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-client-caller"></em>Downloads.Queue <a href="#/?id=downloadsqueue">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>item</code></td>
<td><code class="typename"><span class="type struct-type">InstallQueue</span></code></td>
</tr>
</table>

</div>

### <em class="request-client-caller"></em>Downloads.Prioritize



<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>downloadId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
</table>



<p>
<span class="header">Result</span> <em>none</em>
</p>


<div id="DownloadsPrioritizeParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-client-caller"></em>Downloads.Prioritize <a href="#/?id=downloadsprioritize">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>downloadId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### <em class="request-client-caller"></em>Downloads.List



<p>
<span class="header">Parameters</span> <em>none</em>
</p>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>downloads</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Download__TypeHint">Download</span>[]</code></td>
<td></td>
</tr>
</table>


<div id="DownloadsListParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-client-caller"></em>Downloads.List <a href="#/?id=downloadslist">(Go to definition)</a></p>

</div>

### <em class="request-client-caller"></em>Downloads.Drive


<p>
<p>Drive downloads, which is: perform them one at a time,
until they&rsquo;re all finished.</p>

</p>

<p>
<span class="header">Parameters</span> <em>none</em>
</p>



<p>
<span class="header">Result</span> <em>none</em>
</p>


<div id="DownloadsDriveParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-client-caller"></em>Downloads.Drive <a href="#/?id=downloadsdrive">(Go to definition)</a></p>

<p>
<p>Drive downloads, which is: perform them one at a time,
until they&rsquo;re all finished.</p>

</p>
</div>


## Update

### <em class="request-client-caller"></em>CheckUpdate


<p>
<p>Looks for one or more game updates.</p>

<p>Updates found are regularly sent via <code class="typename"><span class="type notification" data-tip-selector="#GameUpdateAvailableNotification__TypeHint">GameUpdateAvailable</span></code>, and
then all at once in the result.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>items</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#CheckUpdateItem__TypeHint">CheckUpdateItem</span>[]</code></td>
<td><p>A list of items, each of it will be checked for updates</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>updates</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#GameUpdate__TypeHint">GameUpdate</span>[]</code></td>
<td><p>Any updates found (might be empty)</p>
</td>
</tr>
<tr>
<td><code>warnings</code></td>
<td><code class="typename"><span class="type builtin-type">string</span>[]</code></td>
<td><p>Warnings messages logged while looking for updates</p>
</td>
</tr>
</table>


<div id="CheckUpdateParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-client-caller"></em>CheckUpdate <a href="#/?id=checkupdate">(Go to definition)</a></p>

<p>
<p>Looks for one or more game updates.</p>

<p>Updates found are regularly sent via <code class="typename"><span class="type notification">GameUpdateAvailable</span></code>, and
then all at once in the result.</p>

</p>

<table class="field-table">
<tr>
<td><code>items</code></td>
<td><code class="typename"><span class="type struct-type">CheckUpdateItem</span>[]</code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>CheckUpdateItem



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>itemId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>An UUID generated by the client, which allows it to map back the
results to its own items.</p>
</td>
</tr>
<tr>
<td><code>installedAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
<td><p>Timestamp of the last successful install operation</p>
</td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td><p>Game for which to look for an update</p>
</td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Upload__TypeHint">Upload</span></code></td>
<td><p>Currently installed upload</p>
</td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Build__TypeHint">Build</span></code></td>
<td><p>Currently installed build</p>
</td>
</tr>
</table>


<div id="CheckUpdateItem__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>CheckUpdateItem <a href="#/?id=checkupdateitem">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>itemId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>installedAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type struct-type">Game</span></code></td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type struct-type">Upload</span></code></td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type struct-type">Build</span></code></td>
</tr>
</table>

</div>

### <em class="notification"></em>GameUpdateAvailable


<p>
<p>Sent during <code class="typename"><span class="type request-client-caller" data-tip-selector="#CheckUpdateParams__TypeHint">CheckUpdate</span></code>, every time butler
finds an update for a game. Can be safely ignored if displaying
updates as they are found is not a requirement for the client.</p>

</p>

<p>
<span class="header">Payload</span> 
</p>


<table class="field-table">
<tr>
<td><code>update</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#GameUpdate__TypeHint">GameUpdate</span></code></td>
<td></td>
</tr>
</table>


<div id="GameUpdateAvailableNotification__TypeHint" style="display: none;" class="tip-content">
<p><em class="notification"></em>GameUpdateAvailable <a href="#/?id=gameupdateavailable">(Go to definition)</a></p>

<p>
<p>Sent during <code class="typename"><span class="type request-client-caller">CheckUpdate</span></code>, every time butler
finds an update for a game. Can be safely ignored if displaying
updates as they are found is not a requirement for the client.</p>

</p>

<table class="field-table">
<tr>
<td><code>update</code></td>
<td><code class="typename"><span class="type struct-type">GameUpdate</span></code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>GameUpdate


<p>
<p>Describes an available update for a particular game install.</p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>itemId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Identifier originally passed in CheckUpdateItem</p>
</td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td><p>Game we found an update for</p>
</td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Upload__TypeHint">Upload</span></code></td>
<td><p>Upload to be installed</p>
</td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Build__TypeHint">Build</span></code></td>
<td><p>Build to be installed (may be nil)</p>
</td>
</tr>
</table>


<div id="GameUpdate__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>GameUpdate <a href="#/?id=gameupdate">(Go to definition)</a></p>

<p>
<p>Describes an available update for a particular game install.</p>

</p>

<table class="field-table">
<tr>
<td><code>itemId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type struct-type">Game</span></code></td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type struct-type">Upload</span></code></td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type struct-type">Build</span></code></td>
</tr>
</table>

</div>


## Launch

### <em class="request-client-caller"></em>Launch


<p>
<p>Attempt to launch an installed game.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>caveId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>The ID of the cave to launch</p>
</td>
</tr>
<tr>
<td><code>prereqsDir</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>The directory to use to store installer files for prerequisites</p>
</td>
</tr>
<tr>
<td><code>forcePrereqs</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span> Force installing all prerequisites, even if they&rsquo;re already marked as installed</p>
</td>
</tr>
<tr>
<td><code>sandbox</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>Enable sandbox (regardless of manifest opt-in)</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> <em>none</em>
</p>


<div id="LaunchParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-client-caller"></em>Launch <a href="#/?id=launch">(Go to definition)</a></p>

<p>
<p>Attempt to launch an installed game.</p>

</p>

<table class="field-table">
<tr>
<td><code>caveId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>prereqsDir</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>forcePrereqs</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>sandbox</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>

### <em class="notification"></em>LaunchRunning


<p>
<p>Sent during <code class="typename"><span class="type request-client-caller" data-tip-selector="#LaunchParams__TypeHint">Launch</span></code>, when the game is configured, prerequisites are installed
sandbox is set up (if enabled), and the game is actually running.</p>

</p>

<p>
<span class="header">Payload</span> <em>none</em>
</p>


<div id="LaunchRunningNotification__TypeHint" style="display: none;" class="tip-content">
<p><em class="notification"></em>LaunchRunning <a href="#/?id=launchrunning">(Go to definition)</a></p>

<p>
<p>Sent during <code class="typename"><span class="type request-client-caller">Launch</span></code>, when the game is configured, prerequisites are installed
sandbox is set up (if enabled), and the game is actually running.</p>

</p>
</div>

### <em class="notification"></em>LaunchExited


<p>
<p>Sent during <code class="typename"><span class="type request-client-caller" data-tip-selector="#LaunchParams__TypeHint">Launch</span></code>, when the game has actually exited.</p>

</p>

<p>
<span class="header">Payload</span> <em>none</em>
</p>


<div id="LaunchExitedNotification__TypeHint" style="display: none;" class="tip-content">
<p><em class="notification"></em>LaunchExited <a href="#/?id=launchexited">(Go to definition)</a></p>

<p>
<p>Sent during <code class="typename"><span class="type request-client-caller">Launch</span></code>, when the game has actually exited.</p>

</p>
</div>

### <em class="request-server-caller"></em>PickManifestAction


<p>
<p>Sent during <code class="typename"><span class="type request-client-caller" data-tip-selector="#LaunchParams__TypeHint">Launch</span></code>, ask the user to pick a manifest action to launch.</p>

<p>See <a href="https://itch.io/docs/itch/integrating/manifest.html">itch app manifests</a>.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>actions</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Action__TypeHint">Action</span>[]</code></td>
<td><p>A list of actions to pick from. Must be shown to the user in the order they&rsquo;re passed.</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>index</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Index of action picked by user, or negative if aborting</p>
</td>
</tr>
</table>


<div id="PickManifestActionParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-server-caller"></em>PickManifestAction <a href="#/?id=pickmanifestaction">(Go to definition)</a></p>

<p>
<p>Sent during <code class="typename"><span class="type request-client-caller">Launch</span></code>, ask the user to pick a manifest action to launch.</p>

<p>See <a href="https://itch.io/docs/itch/integrating/manifest.html">itch app manifests</a>.</p>

</p>

<table class="field-table">
<tr>
<td><code>actions</code></td>
<td><code class="typename"><span class="type struct-type">Action</span>[]</code></td>
</tr>
</table>

</div>

### <em class="request-server-caller"></em>ShellLaunch


<p>
<p>Ask the client to perform a shell launch, ie. open an item
with the operating system&rsquo;s default handler (File explorer).</p>

<p>Sent during <code class="typename"><span class="type request-client-caller" data-tip-selector="#LaunchParams__TypeHint">Launch</span></code>.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>itemPath</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Absolute path of item to open, e.g. <code>D:\\Games\\Itch\\garden\\README.txt</code></p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> <em>none</em>
</p>


<div id="ShellLaunchParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-server-caller"></em>ShellLaunch <a href="#/?id=shelllaunch">(Go to definition)</a></p>

<p>
<p>Ask the client to perform a shell launch, ie. open an item
with the operating system&rsquo;s default handler (File explorer).</p>

<p>Sent during <code class="typename"><span class="type request-client-caller">Launch</span></code>.</p>

</p>

<table class="field-table">
<tr>
<td><code>itemPath</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### <em class="request-server-caller"></em>HTMLLaunch


<p>
<p>Ask the client to perform an HTML launch, ie. open an HTML5
game, ideally in an embedded browser.</p>

<p>Sent during <code class="typename"><span class="type request-client-caller" data-tip-selector="#LaunchParams__TypeHint">Launch</span></code>.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>rootFolder</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Absolute path on disk to serve</p>
</td>
</tr>
<tr>
<td><code>indexPath</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Path of index file, relative to root folder</p>
</td>
</tr>
<tr>
<td><code>args</code></td>
<td><code class="typename"><span class="type builtin-type">string</span>[]</code></td>
<td><p>Command-line arguments, to pass as <code>global.Itch.args</code></p>
</td>
</tr>
<tr>
<td><code>env</code></td>
<td><code class="typename">Map&lt;<span class="type builtin-type">string</span>, <span class="type builtin-type">string</span>&gt;</code></td>
<td><p>Environment variables, to pass as <code>global.Itch.env</code></p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> <em>none</em>
</p>


<div id="HTMLLaunchParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-server-caller"></em>HTMLLaunch <a href="#/?id=htmllaunch">(Go to definition)</a></p>

<p>
<p>Ask the client to perform an HTML launch, ie. open an HTML5
game, ideally in an embedded browser.</p>

<p>Sent during <code class="typename"><span class="type request-client-caller">Launch</span></code>.</p>

</p>

<table class="field-table">
<tr>
<td><code>rootFolder</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>indexPath</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>args</code></td>
<td><code class="typename"><span class="type builtin-type">string</span>[]</code></td>
</tr>
<tr>
<td><code>env</code></td>
<td><code class="typename">Map&lt;<span class="type builtin-type">string</span>, <span class="type builtin-type">string</span>&gt;</code></td>
</tr>
</table>

</div>

### <em class="request-server-caller"></em>URLLaunch


<p>
<p>Ask the client to perform an URL launch, ie. open an address
with the system browser or appropriate.</p>

<p>Sent during <code class="typename"><span class="type request-client-caller" data-tip-selector="#LaunchParams__TypeHint">Launch</span></code>.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>url</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>URL to open, e.g. <code>https://itch.io/community</code></p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> <em>none</em>
</p>


<div id="URLLaunchParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-server-caller"></em>URLLaunch <a href="#/?id=urllaunch">(Go to definition)</a></p>

<p>
<p>Ask the client to perform an URL launch, ie. open an address
with the system browser or appropriate.</p>

<p>Sent during <code class="typename"><span class="type request-client-caller">Launch</span></code>.</p>

</p>

<table class="field-table">
<tr>
<td><code>url</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### <em class="request-server-caller"></em>AllowSandboxSetup


<p>
<p>Ask the user to allow sandbox setup. Will be followed by
a UAC prompt (on Windows) or a pkexec dialog (on Linux) if
the user allows.</p>

<p>Sent during <code class="typename"><span class="type request-client-caller" data-tip-selector="#LaunchParams__TypeHint">Launch</span></code>.</p>

</p>

<p>
<span class="header">Parameters</span> <em>none</em>
</p>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>allow</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>Set to true if user allowed the sandbox setup, false otherwise</p>
</td>
</tr>
</table>


<div id="AllowSandboxSetupParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-server-caller"></em>AllowSandboxSetup <a href="#/?id=allowsandboxsetup">(Go to definition)</a></p>

<p>
<p>Ask the user to allow sandbox setup. Will be followed by
a UAC prompt (on Windows) or a pkexec dialog (on Linux) if
the user allows.</p>

<p>Sent during <code class="typename"><span class="type request-client-caller">Launch</span></code>.</p>

</p>
</div>

### <em class="notification"></em>PrereqsStarted


<p>
<p>Sent during <code class="typename"><span class="type request-client-caller" data-tip-selector="#LaunchParams__TypeHint">Launch</span></code>, when some prerequisites are about to be installed.</p>

<p>This is a good time to start showing a UI element with the state of prereq
tasks.</p>

<p>Updates are regularly provided via <code class="typename"><span class="type notification" data-tip-selector="#PrereqsTaskStateNotification__TypeHint">PrereqsTaskState</span></code>.</p>

</p>

<p>
<span class="header">Payload</span> 
</p>


<table class="field-table">
<tr>
<td><code>tasks</code></td>
<td><code class="typename">Map&lt;<span class="type builtin-type">string</span>, <span class="type struct-type" data-tip-selector="#PrereqTask__TypeHint">PrereqTask</span>&gt;</code></td>
<td><p>A list of prereqs that need to be tended to</p>
</td>
</tr>
</table>


<div id="PrereqsStartedNotification__TypeHint" style="display: none;" class="tip-content">
<p><em class="notification"></em>PrereqsStarted <a href="#/?id=prereqsstarted">(Go to definition)</a></p>

<p>
<p>Sent during <code class="typename"><span class="type request-client-caller">Launch</span></code>, when some prerequisites are about to be installed.</p>

<p>This is a good time to start showing a UI element with the state of prereq
tasks.</p>

<p>Updates are regularly provided via <code class="typename"><span class="type notification">PrereqsTaskState</span></code>.</p>

</p>

<table class="field-table">
<tr>
<td><code>tasks</code></td>
<td><code class="typename">Map&lt;<span class="type builtin-type">string</span>, <span class="type struct-type">PrereqTask</span>&gt;</code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>PrereqTask


<p>
<p>Information about a prerequisite task.</p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>fullName</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Full name of the prerequisite, for example: <code>Microsoft .NET Framework 4.6.2</code></p>
</td>
</tr>
<tr>
<td><code>order</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Order of task in the list. Respect this order in the UI if you want consistent progress indicators.</p>
</td>
</tr>
</table>


<div id="PrereqTask__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>PrereqTask <a href="#/?id=prereqtask">(Go to definition)</a></p>

<p>
<p>Information about a prerequisite task.</p>

</p>

<table class="field-table">
<tr>
<td><code>fullName</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>order</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### <em class="notification"></em>PrereqsTaskState


<p>
<p>Current status of a prerequisite task</p>

<p>Sent during <code class="typename"><span class="type request-client-caller" data-tip-selector="#LaunchParams__TypeHint">Launch</span></code>, after <code class="typename"><span class="type notification" data-tip-selector="#PrereqsStartedNotification__TypeHint">PrereqsStarted</span></code>, repeatedly
until all prereq tasks are done.</p>

</p>

<p>
<span class="header">Payload</span> 
</p>


<table class="field-table">
<tr>
<td><code>name</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Short name of the prerequisite task (e.g. <code>xna-4.0</code>)</p>
</td>
</tr>
<tr>
<td><code>status</code></td>
<td><code class="typename"><span class="type enum-type" data-tip-selector="#PrereqStatus__TypeHint">PrereqStatus</span></code></td>
<td><p>Current status of the prereq</p>
</td>
</tr>
<tr>
<td><code>progress</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Value between 0 and 1 (floating)</p>
</td>
</tr>
<tr>
<td><code>eta</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>ETA in seconds (floating)</p>
</td>
</tr>
<tr>
<td><code>bps</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Network bandwidth used in bytes per second (floating)</p>
</td>
</tr>
</table>


<div id="PrereqsTaskStateNotification__TypeHint" style="display: none;" class="tip-content">
<p><em class="notification"></em>PrereqsTaskState <a href="#/?id=prereqstaskstate">(Go to definition)</a></p>

<p>
<p>Current status of a prerequisite task</p>

<p>Sent during <code class="typename"><span class="type request-client-caller">Launch</span></code>, after <code class="typename"><span class="type notification">PrereqsStarted</span></code>, repeatedly
until all prereq tasks are done.</p>

</p>

<table class="field-table">
<tr>
<td><code>name</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>status</code></td>
<td><code class="typename"><span class="type enum-type">PrereqStatus</span></code></td>
</tr>
<tr>
<td><code>progress</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>eta</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>bps</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### <em class="enum-type"></em>PrereqStatus



<p>
<span class="header">Values</span> 
</p>


<table class="field-table">
<tr>
<td><code>"pending"</code></td>
<td><p>Prerequisite has not started downloading yet</p>
</td>
</tr>
<tr>
<td><code>"downloading"</code></td>
<td><p>Prerequisite is currently being downloaded</p>
</td>
</tr>
<tr>
<td><code>"ready"</code></td>
<td><p>Prerequisite has been downloaded and is pending installation</p>
</td>
</tr>
<tr>
<td><code>"installing"</code></td>
<td><p>Prerequisite is currently installing</p>
</td>
</tr>
<tr>
<td><code>"done"</code></td>
<td><p>Prerequisite was installed (successfully or not)</p>
</td>
</tr>
</table>


<div id="PrereqStatus__TypeHint" style="display: none;" class="tip-content">
<p><em class="enum-type"></em>PrereqStatus <a href="#/?id=prereqstatus">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>"pending"</code></td>
</tr>
<tr>
<td><code>"downloading"</code></td>
</tr>
<tr>
<td><code>"ready"</code></td>
</tr>
<tr>
<td><code>"installing"</code></td>
</tr>
<tr>
<td><code>"done"</code></td>
</tr>
</table>

</div>

### <em class="notification"></em>PrereqsEnded


<p>
<p>Sent during <code class="typename"><span class="type request-client-caller" data-tip-selector="#LaunchParams__TypeHint">Launch</span></code>, when all prereqs have finished installing (successfully or not)</p>

<p>After this is received, it&rsquo;s safe to close any UI element showing prereq task state.</p>

</p>

<p>
<span class="header">Payload</span> <em>none</em>
</p>


<div id="PrereqsEndedNotification__TypeHint" style="display: none;" class="tip-content">
<p><em class="notification"></em>PrereqsEnded <a href="#/?id=prereqsended">(Go to definition)</a></p>

<p>
<p>Sent during <code class="typename"><span class="type request-client-caller">Launch</span></code>, when all prereqs have finished installing (successfully or not)</p>

<p>After this is received, it&rsquo;s safe to close any UI element showing prereq task state.</p>

</p>
</div>

### <em class="request-server-caller"></em>PrereqsFailed


<p>
<p>Sent during <code class="typename"><span class="type request-client-caller" data-tip-selector="#LaunchParams__TypeHint">Launch</span></code>, when one or more prerequisites have failed to install.
The user may choose to proceed with the launch anyway.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>error</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Short error</p>
</td>
</tr>
<tr>
<td><code>errorStack</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Longer error (to include in logs)</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>continue</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>Set to true if the user wants to proceed with the launch in spite of the prerequisites failure</p>
</td>
</tr>
</table>


<div id="PrereqsFailedParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-server-caller"></em>PrereqsFailed <a href="#/?id=prereqsfailed">(Go to definition)</a></p>

<p>
<p>Sent during <code class="typename"><span class="type request-client-caller">Launch</span></code>, when one or more prerequisites have failed to install.
The user may choose to proceed with the launch anyway.</p>

</p>

<table class="field-table">
<tr>
<td><code>error</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>errorStack</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>


## Clean Downloads

### <em class="request-client-caller"></em>CleanDownloads.Search


<p>
<p>Look for folders we can clean up in various download folders.
This finds anything that doesn&rsquo;t correspond to any current downloads
we know about.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>roots</code></td>
<td><code class="typename"><span class="type builtin-type">string</span>[]</code></td>
<td><p>A list of folders to scan for potential subfolders to clean up</p>
</td>
</tr>
<tr>
<td><code>whitelist</code></td>
<td><code class="typename"><span class="type builtin-type">string</span>[]</code></td>
<td><p>A list of subfolders to not consider when cleaning
(staging folders for in-progress downloads)</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>entries</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#CleanDownloadsEntry__TypeHint">CleanDownloadsEntry</span>[]</code></td>
<td><p>Entries we found that could use some cleaning (with path and size information)</p>
</td>
</tr>
</table>


<div id="CleanDownloadsSearchParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-client-caller"></em>CleanDownloads.Search <a href="#/?id=cleandownloadssearch">(Go to definition)</a></p>

<p>
<p>Look for folders we can clean up in various download folders.
This finds anything that doesn&rsquo;t correspond to any current downloads
we know about.</p>

</p>

<table class="field-table">
<tr>
<td><code>roots</code></td>
<td><code class="typename"><span class="type builtin-type">string</span>[]</code></td>
</tr>
<tr>
<td><code>whitelist</code></td>
<td><code class="typename"><span class="type builtin-type">string</span>[]</code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>CleanDownloadsEntry



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>path</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>The complete path of the file or folder we intend to remove</p>
</td>
</tr>
<tr>
<td><code>size</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>The size of the folder or file, in bytes</p>
</td>
</tr>
</table>


<div id="CleanDownloadsEntry__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>CleanDownloadsEntry <a href="#/?id=cleandownloadsentry">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>path</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>size</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### <em class="request-client-caller"></em>CleanDownloads.Apply


<p>
<p>Remove the specified entries from disk, freeing up disk space.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>entries</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#CleanDownloadsEntry__TypeHint">CleanDownloadsEntry</span>[]</code></td>
<td></td>
</tr>
</table>



<p>
<span class="header">Result</span> <em>none</em>
</p>


<div id="CleanDownloadsApplyParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-client-caller"></em>CleanDownloads.Apply <a href="#/?id=cleandownloadsapply">(Go to definition)</a></p>

<p>
<p>Remove the specified entries from disk, freeing up disk space.</p>

</p>

<table class="field-table">
<tr>
<td><code>entries</code></td>
<td><code class="typename"><span class="type struct-type">CleanDownloadsEntry</span>[]</code></td>
</tr>
</table>

</div>


## Test

### <em class="request-client-caller"></em>Test.DoubleTwice


<p>
<p>Test request: asks butler to double a number twice.
First by calling <code class="typename"><span class="type request-server-caller" data-tip-selector="#TestDoubleParams__TypeHint">Test.Double</span></code>, then by
returning the result of that call doubled.</p>

<p>Use that to try out your JSON-RPC 2.0 over TCP implementation.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>number</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>The number to quadruple</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>number</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>The input, quadrupled</p>
</td>
</tr>
</table>


<div id="TestDoubleTwiceParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-client-caller"></em>Test.DoubleTwice <a href="#/?id=testdoubletwice">(Go to definition)</a></p>

<p>
<p>Test request: asks butler to double a number twice.
First by calling <code class="typename"><span class="type request-server-caller">Test.Double</span></code>, then by
returning the result of that call doubled.</p>

<p>Use that to try out your JSON-RPC 2.0 over TCP implementation.</p>

</p>

<table class="field-table">
<tr>
<td><code>number</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### <em class="request-server-caller"></em>Test.Double


<p>
<p>Test request: return a number, doubled. Implement that to
use <code class="typename"><span class="type request-client-caller" data-tip-selector="#TestDoubleTwiceParams__TypeHint">Test.DoubleTwice</span></code> in your testing.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>number</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>The number to double</p>
</td>
</tr>
</table>


<p>
<p>Result for Test.Double</p>

</p>

<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>number</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>The number, doubled</p>
</td>
</tr>
</table>


<div id="TestDoubleParams__TypeHint" style="display: none;" class="tip-content">
<p><em class="request-server-caller"></em>Test.Double <a href="#/?id=testdouble">(Go to definition)</a></p>

<p>
<p>Test request: return a number, doubled. Implement that to
use <code class="typename"><span class="type request-client-caller">Test.DoubleTwice</span></code> in your testing.</p>

</p>

<table class="field-table">
<tr>
<td><code>number</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>


## Miscellaneous

### <em class="struct-type"></em>Profile


<p>
<p>Represents a user for which we have profile information,
ie. that we can connect as, etc.</p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>itch.io user ID, doubling as profile ID</p>
</td>
</tr>
<tr>
<td><code>lastConnected</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
<td><p>Timestamp the user last connected at (to the client)</p>
</td>
</tr>
<tr>
<td><code>user</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#User__TypeHint">User</span></code></td>
<td><p>User information</p>
</td>
</tr>
</table>


<div id="Profile__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>Profile <a href="#/?id=profile">(Go to definition)</a></p>

<p>
<p>Represents a user for which we have profile information,
ie. that we can connect as, etc.</p>

</p>

<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>lastConnected</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
</tr>
<tr>
<td><code>user</code></td>
<td><code class="typename"><span class="type struct-type">User</span></code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>ProfileGame



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td></td>
</tr>
<tr>
<td><code>user</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#User__TypeHint">User</span></code></td>
<td></td>
</tr>
<tr>
<td><code>position</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Position on profile, from 0 to N</p>
</td>
</tr>
<tr>
<td><code>viewsCount</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>downloadsCount</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>purchasesCount</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>published</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td></td>
</tr>
</table>


<div id="ProfileGame__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>ProfileGame <a href="#/?id=profilegame">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type struct-type">Game</span></code></td>
</tr>
<tr>
<td><code>user</code></td>
<td><code class="typename"><span class="type struct-type">User</span></code></td>
</tr>
<tr>
<td><code>position</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>viewsCount</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>downloadsCount</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>purchasesCount</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>published</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>DownloadKeySummary



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Site-wide unique identifier generated by itch.io</p>
</td>
</tr>
<tr>
<td><code>gameId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Identifier of the game to which this download key grants access</p>
</td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
<td><p>Date this key was created at (often coincides with purchase time)</p>
</td>
</tr>
</table>


<div id="DownloadKeySummary__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>DownloadKeySummary <a href="#/?id=downloadkeysummary">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>gameId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>CaveSummary



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
<tr>
<td><code>gameId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>lastTouchedAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
<td></td>
</tr>
<tr>
<td><code>secondsRun</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>installedSize</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
</table>


<div id="CaveSummary__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>CaveSummary <a href="#/?id=cavesummary">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>gameId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>lastTouchedAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
</tr>
<tr>
<td><code>secondsRun</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>installedSize</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>Cave



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td></td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Upload__TypeHint">Upload</span></code></td>
<td></td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Build__TypeHint">Build</span></code></td>
<td></td>
</tr>
<tr>
<td><code>stats</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#CaveStats__TypeHint">CaveStats</span></code></td>
<td></td>
</tr>
<tr>
<td><code>installInfo</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#CaveInstallInfo__TypeHint">CaveInstallInfo</span></code></td>
<td></td>
</tr>
</table>


<div id="Cave__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>Cave <a href="#/?id=cave">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type struct-type">Game</span></code></td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type struct-type">Upload</span></code></td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type struct-type">Build</span></code></td>
</tr>
<tr>
<td><code>stats</code></td>
<td><code class="typename"><span class="type struct-type">CaveStats</span></code></td>
</tr>
<tr>
<td><code>installInfo</code></td>
<td><code class="typename"><span class="type struct-type">CaveInstallInfo</span></code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>CaveStats



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>installedAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
<td></td>
</tr>
<tr>
<td><code>lastTouchedAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
<td></td>
</tr>
<tr>
<td><code>secondsRun</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
</table>


<div id="CaveStats__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>CaveStats <a href="#/?id=cavestats">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>installedAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
</tr>
<tr>
<td><code>lastTouchedAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
</tr>
<tr>
<td><code>secondsRun</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>CaveInstallInfo



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>installedSize</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>installLocation</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
<tr>
<td><code>installFolder</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
</table>


<div id="CaveInstallInfo__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>CaveInstallInfo <a href="#/?id=caveinstallinfo">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>installedSize</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>installLocation</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>installFolder</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>InstallLocationSummary



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>installLocation</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
<tr>
<td><code>size</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
</table>


<div id="InstallLocationSummary__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>InstallLocationSummary <a href="#/?id=installlocationsummary">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>installLocation</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>size</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>GameCredentials


<p>
<p>GameCredentials contains all the credentials required to make API requests
including the download key if any.</p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>server</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p><span class="tag">Optional</span> Defaults to <code>https://itch.io</code></p>
</td>
</tr>
<tr>
<td><code>apiKey</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>A valid itch.io API key</p>
</td>
</tr>
<tr>
<td><code>downloadKey</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p><span class="tag">Optional</span> A download key identifier, or 0 if no download key is available</p>
</td>
</tr>
</table>


<div id="GameCredentials__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>GameCredentials <a href="#/?id=gamecredentials">(Go to definition)</a></p>

<p>
<p>GameCredentials contains all the credentials required to make API requests
including the download key if any.</p>

</p>

<table class="field-table">
<tr>
<td><code>server</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>apiKey</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>downloadKey</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### <em class="notification"></em>Downloads.Drive.Progress



<p>
<span class="header">Payload</span> 
</p>


<table class="field-table">
<tr>
<td><code>download</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Download__TypeHint">Download</span></code></td>
<td></td>
</tr>
<tr>
<td><code>progress</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#DownloadProgress__TypeHint">DownloadProgress</span></code></td>
<td></td>
</tr>
</table>


<div id="DownloadsDriveProgressNotification__TypeHint" style="display: none;" class="tip-content">
<p><em class="notification"></em>Downloads.Drive.Progress <a href="#/?id=downloadsdriveprogress">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>download</code></td>
<td><code class="typename"><span class="type struct-type">Download</span></code></td>
</tr>
<tr>
<td><code>progress</code></td>
<td><code class="typename"><span class="type struct-type">DownloadProgress</span></code></td>
</tr>
</table>

</div>

### <em class="notification"></em>Downloads.Drive.Finished



<p>
<span class="header">Payload</span> 
</p>


<table class="field-table">
<tr>
<td><code>download</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Download__TypeHint">Download</span></code></td>
<td></td>
</tr>
</table>


<div id="DownloadsDriveFinishedNotification__TypeHint" style="display: none;" class="tip-content">
<p><em class="notification"></em>Downloads.Drive.Finished <a href="#/?id=downloadsdrivefinished">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>download</code></td>
<td><code class="typename"><span class="type struct-type">Download</span></code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>Download


<p>
<p>Represents a download queued, which will be
performed whenever <code class="typename"><span class="type request-client-caller" data-tip-selector="#DownloadsDriveParams__TypeHint">Downloads.Drive</span></code> is called.</p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>caveId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td></td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Upload__TypeHint">Upload</span></code></td>
<td></td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Build__TypeHint">Build</span></code></td>
<td></td>
</tr>
</table>


<div id="Download__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>Download <a href="#/?id=download">(Go to definition)</a></p>

<p>
<p>Represents a download queued, which will be
performed whenever <code class="typename"><span class="type request-client-caller">Downloads.Drive</span></code> is called.</p>

</p>

<table class="field-table">
<tr>
<td><code>caveId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type struct-type">Game</span></code></td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type struct-type">Upload</span></code></td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type struct-type">Build</span></code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>DownloadProgress



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>stage</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
<tr>
<td><code>progress</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>eta</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>bps</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
</table>


<div id="DownloadProgress__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>DownloadProgress <a href="#/?id=downloadprogress">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>stage</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>progress</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>eta</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>bps</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### <em class="notification"></em>Log


<p>
<p>Sent any time butler needs to send a log message. The client should
relay them in their own stdout / stderr, and collect them so they
can be part of an issue report if something goes wrong.</p>

</p>

<p>
<span class="header">Payload</span> 
</p>


<table class="field-table">
<tr>
<td><code>level</code></td>
<td><code class="typename"><span class="type enum-type" data-tip-selector="#LogLevel__TypeHint">LogLevel</span></code></td>
<td><p>Level of the message (<code>info</code>, <code>warn</code>, etc.)</p>
</td>
</tr>
<tr>
<td><code>message</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Contents of the message.</p>

<p>Note: logs may contain non-ASCII characters, or even emojis.</p>
</td>
</tr>
</table>


<div id="LogNotification__TypeHint" style="display: none;" class="tip-content">
<p><em class="notification"></em>Log <a href="#/?id=log">(Go to definition)</a></p>

<p>
<p>Sent any time butler needs to send a log message. The client should
relay them in their own stdout / stderr, and collect them so they
can be part of an issue report if something goes wrong.</p>

</p>

<table class="field-table">
<tr>
<td><code>level</code></td>
<td><code class="typename"><span class="type enum-type">LogLevel</span></code></td>
</tr>
<tr>
<td><code>message</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### <em class="enum-type"></em>LogLevel



<p>
<span class="header">Values</span> 
</p>


<table class="field-table">
<tr>
<td><code>"debug"</code></td>
<td><p>Hidden from logs by default, noisy</p>
</td>
</tr>
<tr>
<td><code>"info"</code></td>
<td><p>Just thinking out loud</p>
</td>
</tr>
<tr>
<td><code>"warning"</code></td>
<td><p>We&rsquo;re continuing, but we&rsquo;re not thrilled about it</p>
</td>
</tr>
<tr>
<td><code>"error"</code></td>
<td><p>We&rsquo;re eventually going to fail loudly</p>
</td>
</tr>
</table>


<div id="LogLevel__TypeHint" style="display: none;" class="tip-content">
<p><em class="enum-type"></em>LogLevel <a href="#/?id=loglevel">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>"debug"</code></td>
</tr>
<tr>
<td><code>"info"</code></td>
</tr>
<tr>
<td><code>"warning"</code></td>
</tr>
<tr>
<td><code>"error"</code></td>
</tr>
</table>

</div>

### <em class="enum-type"></em>ItchPlatform



<p>
<span class="header">Values</span> 
</p>


<table class="field-table">
<tr>
<td><code>"osx"</code></td>
<td></td>
</tr>
<tr>
<td><code>"windows"</code></td>
<td></td>
</tr>
<tr>
<td><code>"linux"</code></td>
<td></td>
</tr>
<tr>
<td><code>"unknown"</code></td>
<td></td>
</tr>
</table>


<div id="ItchPlatform__TypeHint" style="display: none;" class="tip-content">
<p><em class="enum-type"></em>ItchPlatform <a href="#/?id=itchplatform">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>"osx"</code></td>
</tr>
<tr>
<td><code>"windows"</code></td>
</tr>
<tr>
<td><code>"linux"</code></td>
</tr>
<tr>
<td><code>"unknown"</code></td>
</tr>
</table>

</div>

### <em class="enum-type"></em>Code


<p>
<p>Buse JSON-RPC 2.0 error codes</p>

</p>

<p>
<span class="header">Values</span> 
</p>


<table class="field-table">
<tr>
<td><code>499</code></td>
<td><p>An operation was cancelled gracefully</p>
</td>
</tr>
<tr>
<td><code>410</code></td>
<td><p>An operation was aborted by the user</p>
</td>
</tr>
</table>


<div id="Code__TypeHint" style="display: none;" class="tip-content">
<p><em class="enum-type"></em>Code <a href="#/?id=code">(Go to definition)</a></p>

<p>
<p>Buse JSON-RPC 2.0 error codes</p>

</p>

<table class="field-table">
<tr>
<td><code>499</code></td>
</tr>
<tr>
<td><code>410</code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>Manifest


<p>
<p>A Manifest describes prerequisites (dependencies) and actions that
can be taken while launching a game.</p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>actions</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Action__TypeHint">Action</span>[]</code></td>
<td><p>Actions are a list of options to give the user when launching a game.</p>
</td>
</tr>
<tr>
<td><code>prereqs</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Prereq__TypeHint">Prereq</span>[]</code></td>
<td><p>Prereqs describe libraries or frameworks that must be installed
prior to launching a game</p>
</td>
</tr>
</table>


<div id="Manifest__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>Manifest <a href="#/?id=manifest">(Go to definition)</a></p>

<p>
<p>A Manifest describes prerequisites (dependencies) and actions that
can be taken while launching a game.</p>

</p>

<table class="field-table">
<tr>
<td><code>actions</code></td>
<td><code class="typename"><span class="type struct-type">Action</span>[]</code></td>
</tr>
<tr>
<td><code>prereqs</code></td>
<td><code class="typename"><span class="type struct-type">Prereq</span>[]</code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>Action


<p>
<p>An Action is a choice for the user to pick when launching a game.</p>

<p>see <a href="https://itch.io/docs/itch/integrating/manifest.html">https://itch.io/docs/itch/integrating/manifest.html</a></p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>name</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>human-readable or standard name</p>
</td>
</tr>
<tr>
<td><code>path</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>file path (relative to manifest or absolute), URL, etc.</p>
</td>
</tr>
<tr>
<td><code>icon</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>icon name (see static/fonts/icomoon/demo.html, don&rsquo;t include <code>icon-</code> prefix)</p>
</td>
</tr>
<tr>
<td><code>args</code></td>
<td><code class="typename"><span class="type builtin-type">string</span>[]</code></td>
<td><p>command-line arguments</p>
</td>
</tr>
<tr>
<td><code>sandbox</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>sandbox opt-in</p>
</td>
</tr>
<tr>
<td><code>scope</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>requested API scope</p>
</td>
</tr>
<tr>
<td><code>console</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>don&rsquo;t redirect stdout/stderr, open in new console window</p>
</td>
</tr>
<tr>
<td><code>platform</code></td>
<td><code class="typename"><span class="type enum-type" data-tip-selector="#ItchPlatform__TypeHint">ItchPlatform</span></code></td>
<td><p>platform to restrict this action too</p>
</td>
</tr>
<tr>
<td><code>locales</code></td>
<td><code class="typename">Map&lt;<span class="type builtin-type">string</span>, <span class="type struct-type" data-tip-selector="#ActionLocale__TypeHint">ActionLocale</span>&gt;</code></td>
<td><p>localized action name</p>
</td>
</tr>
</table>


<div id="Action__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>Action <a href="#/?id=action">(Go to definition)</a></p>

<p>
<p>An Action is a choice for the user to pick when launching a game.</p>

<p>see <a href="https://itch.io/docs/itch/integrating/manifest.html">https://itch.io/docs/itch/integrating/manifest.html</a></p>

</p>

<table class="field-table">
<tr>
<td><code>name</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>path</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>icon</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>args</code></td>
<td><code class="typename"><span class="type builtin-type">string</span>[]</code></td>
</tr>
<tr>
<td><code>sandbox</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>scope</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>console</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>platform</code></td>
<td><code class="typename"><span class="type enum-type">ItchPlatform</span></code></td>
</tr>
<tr>
<td><code>locales</code></td>
<td><code class="typename">Map&lt;<span class="type builtin-type">string</span>, <span class="type struct-type">ActionLocale</span>&gt;</code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>Prereq



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>name</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>A prerequisite to be installed, see <a href="https://itch.io/docs/itch/integrating/prereqs/">https://itch.io/docs/itch/integrating/prereqs/</a> for the full list.</p>
</td>
</tr>
</table>


<div id="Prereq__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>Prereq <a href="#/?id=prereq">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>name</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>ActionLocale



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>name</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>A localized action name</p>
</td>
</tr>
</table>


<div id="ActionLocale__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>ActionLocale <a href="#/?id=actionlocale">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>name</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>User


<p>
<p>User represents an itch.io account, with basic profile info</p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Site-wide unique identifier generated by itch.io</p>
</td>
</tr>
<tr>
<td><code>username</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>The user&rsquo;s username (used for login)</p>
</td>
</tr>
<tr>
<td><code>displayName</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>The user&rsquo;s display name: human-friendly, may contain spaces, unicode etc.</p>
</td>
</tr>
<tr>
<td><code>developer</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>Has the user opted into creating games?</p>
</td>
</tr>
<tr>
<td><code>pressUser</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>Is the user part of itch.io&rsquo;s press program?</p>
</td>
</tr>
<tr>
<td><code>url</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>The address of the user&rsquo;s page on itch.io</p>
</td>
</tr>
<tr>
<td><code>coverUrl</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>User&rsquo;s avatar, may be a GIF</p>
</td>
</tr>
<tr>
<td><code>stillCoverUrl</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Static version of user&rsquo;s avatar, only set if the main cover URL is a GIF</p>
</td>
</tr>
</table>


<div id="User__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>User <a href="#/?id=user">(Go to definition)</a></p>

<p>
<p>User represents an itch.io account, with basic profile info</p>

</p>

<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>username</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>displayName</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>developer</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>pressUser</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>url</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>coverUrl</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>stillCoverUrl</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>Game


<p>
<p>Game represents a page on itch.io, it could be a game,
a tool, a comic, etc.</p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Site-wide unique identifier generated by itch.io</p>
</td>
</tr>
<tr>
<td><code>url</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Canonical address of the game&rsquo;s page on itch.io</p>
</td>
</tr>
<tr>
<td><code>title</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Human-friendly title (may contain any character)</p>
</td>
</tr>
<tr>
<td><code>shortText</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Human-friendly short description</p>
</td>
</tr>
<tr>
<td><code>type</code></td>
<td><code class="typename"><span class="type enum-type" data-tip-selector="#GameType__TypeHint">GameType</span></code></td>
<td><p>Downloadable game, html game, etc.</p>
</td>
</tr>
<tr>
<td><code>classification</code></td>
<td><code class="typename"><span class="type enum-type" data-tip-selector="#GameClassification__TypeHint">GameClassification</span></code></td>
<td><p>Classification: game, tool, comic, etc.</p>
</td>
</tr>
<tr>
<td><code>embed</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#GameEmbedData__TypeHint">GameEmbedData</span></code></td>
<td><p><span class="tag">Optional</span> Configuration for embedded (HTML5) games</p>
</td>
</tr>
<tr>
<td><code>coverUrl</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Cover url (might be a GIF)</p>
</td>
</tr>
<tr>
<td><code>stillCoverUrl</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Non-gif cover url, only set if main cover url is a GIF</p>
</td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
<td><p>Date the game was created</p>
</td>
</tr>
<tr>
<td><code>publishedAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
<td><p>Date the game was published, empty if not currently published</p>
</td>
</tr>
<tr>
<td><code>minPrice</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Price in cents of a dollar</p>
</td>
</tr>
<tr>
<td><code>canBeBought</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>Can this game be bought?</p>
</td>
</tr>
<tr>
<td><code>inPressSystem</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>Is this game downloadable by press users for free?</p>
</td>
</tr>
<tr>
<td><code>hasDemo</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>Does this game have a demo that can be downloaded for free?</p>
</td>
</tr>
<tr>
<td><code>pOsx</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>Does this game have an upload tagged with &lsquo;macOS compatible&rsquo;? (creator-controlled)</p>
</td>
</tr>
<tr>
<td><code>pLinux</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>Does this game have an upload tagged with &lsquo;Linux compatible&rsquo;? (creator-controlled)</p>
</td>
</tr>
<tr>
<td><code>pWindows</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>Does this game have an upload tagged with &lsquo;Windows compatible&rsquo;? (creator-controlled)</p>
</td>
</tr>
<tr>
<td><code>pAndroid</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>Does this game have an upload tagged with &lsquo;Android compatible&rsquo;? (creator-controlled)</p>
</td>
</tr>
<tr>
<td><code>user</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#User__TypeHint">User</span></code></td>
<td><p><span class="tag">Optional</span> The user account this game is associated to</p>
</td>
</tr>
<tr>
<td><code>userId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>ID of the user account this game is associated to</p>
</td>
</tr>
<tr>
<td><code>sale</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Sale__TypeHint">Sale</span></code></td>
<td><p><span class="tag">Optional</span> The best current sale for this game</p>
</td>
</tr>
<tr>
<td><code>viewsCount</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>downloadsCount</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>purchasesCount</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>published</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td></td>
</tr>
</table>


<div id="Game__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>Game <a href="#/?id=game">(Go to definition)</a></p>

<p>
<p>Game represents a page on itch.io, it could be a game,
a tool, a comic, etc.</p>

</p>

<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>url</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>title</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>shortText</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>type</code></td>
<td><code class="typename"><span class="type enum-type">GameType</span></code></td>
</tr>
<tr>
<td><code>classification</code></td>
<td><code class="typename"><span class="type enum-type">GameClassification</span></code></td>
</tr>
<tr>
<td><code>embed</code></td>
<td><code class="typename"><span class="type struct-type">GameEmbedData</span></code></td>
</tr>
<tr>
<td><code>coverUrl</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>stillCoverUrl</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
</tr>
<tr>
<td><code>publishedAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
</tr>
<tr>
<td><code>minPrice</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>canBeBought</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>inPressSystem</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>hasDemo</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>pOsx</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>pLinux</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>pWindows</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>pAndroid</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>user</code></td>
<td><code class="typename"><span class="type struct-type">User</span></code></td>
</tr>
<tr>
<td><code>userId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>sale</code></td>
<td><code class="typename"><span class="type struct-type">Sale</span></code></td>
</tr>
<tr>
<td><code>viewsCount</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>downloadsCount</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>purchasesCount</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>published</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>

### <em class="enum-type"></em>GameType


<p>
<p>Type of an itch.io game page, mostly related to
how it should be presented on web (downloadable or embed)</p>

</p>

<p>
<span class="header">Values</span> 
</p>


<table class="field-table">
<tr>
<td><code>"default"</code></td>
<td><p>downloadable</p>
</td>
</tr>
<tr>
<td><code>"flash"</code></td>
<td><p>.swf (legacy)</p>
</td>
</tr>
<tr>
<td><code>"unity"</code></td>
<td><p>.unity3d (legacy)</p>
</td>
</tr>
<tr>
<td><code>"java"</code></td>
<td><p>.jar (legacy)</p>
</td>
</tr>
<tr>
<td><code>"html"</code></td>
<td><p>.html (thriving)</p>
</td>
</tr>
</table>


<div id="GameType__TypeHint" style="display: none;" class="tip-content">
<p><em class="enum-type"></em>GameType <a href="#/?id=gametype">(Go to definition)</a></p>

<p>
<p>Type of an itch.io game page, mostly related to
how it should be presented on web (downloadable or embed)</p>

</p>

<table class="field-table">
<tr>
<td><code>"default"</code></td>
</tr>
<tr>
<td><code>"flash"</code></td>
</tr>
<tr>
<td><code>"unity"</code></td>
</tr>
<tr>
<td><code>"java"</code></td>
</tr>
<tr>
<td><code>"html"</code></td>
</tr>
</table>

</div>

### <em class="enum-type"></em>GameClassification


<p>
<p>Creator-picked classification for a page</p>

</p>

<p>
<span class="header">Values</span> 
</p>


<table class="field-table">
<tr>
<td><code>"game"</code></td>
<td><p>something you can play</p>
</td>
</tr>
<tr>
<td><code>"tool"</code></td>
<td><p>all software pretty much</p>
</td>
</tr>
<tr>
<td><code>"assets"</code></td>
<td><p>assets: graphics, sounds, etc.</p>
</td>
</tr>
<tr>
<td><code>"game_mod"</code></td>
<td><p>game mod (no link to game, purely creator tagging)</p>
</td>
</tr>
<tr>
<td><code>"physical_game"</code></td>
<td><p>printable / board / card game</p>
</td>
</tr>
<tr>
<td><code>"soundtrack"</code></td>
<td><p>bunch of music files</p>
</td>
</tr>
<tr>
<td><code>"other"</code></td>
<td><p>anything that creators think don&rsquo;t fit in any other category</p>
</td>
</tr>
<tr>
<td><code>"comic"</code></td>
<td><p>comic book (pdf, jpg, specific comic formats, etc.)</p>
</td>
</tr>
<tr>
<td><code>"book"</code></td>
<td><p>book (pdf, jpg, specific e-book formats, etc.)</p>
</td>
</tr>
</table>


<div id="GameClassification__TypeHint" style="display: none;" class="tip-content">
<p><em class="enum-type"></em>GameClassification <a href="#/?id=gameclassification">(Go to definition)</a></p>

<p>
<p>Creator-picked classification for a page</p>

</p>

<table class="field-table">
<tr>
<td><code>"game"</code></td>
</tr>
<tr>
<td><code>"tool"</code></td>
</tr>
<tr>
<td><code>"assets"</code></td>
</tr>
<tr>
<td><code>"game_mod"</code></td>
</tr>
<tr>
<td><code>"physical_game"</code></td>
</tr>
<tr>
<td><code>"soundtrack"</code></td>
</tr>
<tr>
<td><code>"other"</code></td>
</tr>
<tr>
<td><code>"comic"</code></td>
</tr>
<tr>
<td><code>"book"</code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>GameEmbedData


<p>
<p>Presentation information for embed games</p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>gameId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Game this embed info is for</p>
</td>
</tr>
<tr>
<td><code>width</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>width of the initial viewport, in pixels</p>
</td>
</tr>
<tr>
<td><code>height</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>height of the initial viewport, in pixels</p>
</td>
</tr>
<tr>
<td><code>fullscreen</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>for itch.io website, whether or not a fullscreen button should be shown</p>
</td>
</tr>
</table>


<div id="GameEmbedData__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>GameEmbedData <a href="#/?id=gameembeddata">(Go to definition)</a></p>

<p>
<p>Presentation information for embed games</p>

</p>

<table class="field-table">
<tr>
<td><code>gameId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>width</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>height</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>fullscreen</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>Sale


<p>
<p>Describes a discount for a game.</p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Site-wide unique identifier generated by itch.io</p>
</td>
</tr>
<tr>
<td><code>gameId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Game this sale is for</p>
</td>
</tr>
<tr>
<td><code>rate</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Discount rate in percent.
Can be negative, see <a href="https://itch.io/updates/introducing-reverse-sales">https://itch.io/updates/introducing-reverse-sales</a></p>
</td>
</tr>
<tr>
<td><code>startDate</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Timestamp the sale started at</p>
</td>
</tr>
<tr>
<td><code>endDate</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Timestamp the sale ends at</p>
</td>
</tr>
</table>


<div id="Sale__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>Sale <a href="#/?id=sale">(Go to definition)</a></p>

<p>
<p>Describes a discount for a game.</p>

</p>

<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>gameId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>rate</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>startDate</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>endDate</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>Upload


<p>
<p>An Upload is a downloadable file. Some are wharf-enabled, which means
they&rsquo;re actually a &ldquo;channel&rdquo; that may contain multiple builds, pushed
with <a href="https://github.com/itchio/butler">https://github.com/itchio/butler</a></p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Site-wide unique identifier generated by itch.io</p>
</td>
</tr>
<tr>
<td><code>filename</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Original file name (example: <code>Overland_x64.zip</code>)</p>
</td>
</tr>
<tr>
<td><code>displayName</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Human-friendly name set by developer (example: <code>Overland for Windows 64-bit</code>)</p>
</td>
</tr>
<tr>
<td><code>size</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Size of upload in bytes. For wharf-enabled uploads, it&rsquo;s the archive size.</p>
</td>
</tr>
<tr>
<td><code>channelName</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Name of the wharf channel for this upload, if it&rsquo;s a wharf-enabled upload</p>
</td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Build__TypeHint">Build</span></code></td>
<td><p>Latest build for this upload, if it&rsquo;s a wharf-enabled upload</p>
</td>
</tr>
<tr>
<td><code>demo</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>Is this upload a demo that can be downloaded for free?</p>
</td>
</tr>
<tr>
<td><code>preorder</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>Is this upload a pre-order placeholder?</p>
</td>
</tr>
<tr>
<td><code>type</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Upload type: default, soundtrack, etc.</p>
</td>
</tr>
<tr>
<td><code>pOsx</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>Is this upload tagged with &lsquo;macOS compatible&rsquo;? (creator-controlled)</p>
</td>
</tr>
<tr>
<td><code>pLinux</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>Is this upload tagged with &lsquo;Linux compatible&rsquo;? (creator-controlled)</p>
</td>
</tr>
<tr>
<td><code>pWindows</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>Is this upload tagged with &lsquo;Windows compatible&rsquo;? (creator-controlled)</p>
</td>
</tr>
<tr>
<td><code>pAndroid</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>Is this upload tagged with &lsquo;Android compatible&rsquo;? (creator-controlled)</p>
</td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
<td><p>Date this upload was created at</p>
</td>
</tr>
<tr>
<td><code>updatedAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
<td><p>Date this upload was last updated at (order changed, display name set, etc.)</p>
</td>
</tr>
</table>


<div id="Upload__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>Upload <a href="#/?id=upload">(Go to definition)</a></p>

<p>
<p>An Upload is a downloadable file. Some are wharf-enabled, which means
they&rsquo;re actually a &ldquo;channel&rdquo; that may contain multiple builds, pushed
with <a href="https://github.com/itchio/butler">https://github.com/itchio/butler</a></p>

</p>

<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>filename</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>displayName</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>size</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>channelName</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type struct-type">Build</span></code></td>
</tr>
<tr>
<td><code>demo</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>preorder</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>type</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>pOsx</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>pLinux</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>pWindows</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>pAndroid</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
</tr>
<tr>
<td><code>updatedAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>Collection


<p>
<p>A Collection is a set of games, curated by humans.</p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Site-wide unique identifier generated by itch.io</p>
</td>
</tr>
<tr>
<td><code>title</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Human-friendly title for collection, for example <code>Couch coop games</code></p>
</td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
<td><p>Date this collection was created at</p>
</td>
</tr>
<tr>
<td><code>updatedAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
<td><p>Date this collection was last updated at (item added, title set, etc.)</p>
</td>
</tr>
<tr>
<td><code>gamesCount</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Number of games in the collection. This might not be accurate
as some games might not be accessible to whoever is asking (project
page deleted, visibility level changed, etc.)</p>
</td>
</tr>
<tr>
<td><code>games</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Game__TypeHint">Game</span>[]</code></td>
<td><p>Games in this collection: filled in API response</p>
</td>
</tr>
<tr>
<td><code>collectionGames</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#CollectionGame__TypeHint">CollectionGame</span>[]</code></td>
<td><p>Games in this collection, with additional info</p>
</td>
</tr>
<tr>
<td><code>userId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>user</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#User__TypeHint">User</span></code></td>
<td></td>
</tr>
</table>


<div id="Collection__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>Collection <a href="#/?id=collection">(Go to definition)</a></p>

<p>
<p>A Collection is a set of games, curated by humans.</p>

</p>

<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>title</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
</tr>
<tr>
<td><code>updatedAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
</tr>
<tr>
<td><code>gamesCount</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>games</code></td>
<td><code class="typename"><span class="type struct-type">Game</span>[]</code></td>
</tr>
<tr>
<td><code>collectionGames</code></td>
<td><code class="typename"><span class="type struct-type">CollectionGame</span>[]</code></td>
</tr>
<tr>
<td><code>userId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>user</code></td>
<td><code class="typename"><span class="type struct-type">User</span></code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>CollectionGame



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>collectionId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>collection</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Collection__TypeHint">Collection</span></code></td>
<td></td>
</tr>
<tr>
<td><code>gameId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td></td>
</tr>
<tr>
<td><code>position</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
<td></td>
</tr>
<tr>
<td><code>updatedAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
<td></td>
</tr>
<tr>
<td><code>blurb</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
<tr>
<td><code>userId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
</table>


<div id="CollectionGame__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>CollectionGame <a href="#/?id=collectiongame">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>collectionId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>collection</code></td>
<td><code class="typename"><span class="type struct-type">Collection</span></code></td>
</tr>
<tr>
<td><code>gameId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type struct-type">Game</span></code></td>
</tr>
<tr>
<td><code>position</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
</tr>
<tr>
<td><code>updatedAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
</tr>
<tr>
<td><code>blurb</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>userId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>DownloadKey


<p>
<p>A download key is often generated when a purchase is made, it
allows downloading uploads for a game that are not available
for free.</p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Site-wide unique identifier generated by itch.io</p>
</td>
</tr>
<tr>
<td><code>gameId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Identifier of the game to which this download key grants access</p>
</td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td><p>Game to which this download key grants access</p>
</td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
<td><p>Date this key was created at (often coincides with purchase time)</p>
</td>
</tr>
<tr>
<td><code>updatedAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
<td><p>Date this key was last updated at</p>
</td>
</tr>
<tr>
<td><code>ownerId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Identifier of the itch.io user to which this key belongs</p>
</td>
</tr>
</table>


<div id="DownloadKey__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>DownloadKey <a href="#/?id=downloadkey">(Go to definition)</a></p>

<p>
<p>A download key is often generated when a purchase is made, it
allows downloading uploads for a game that are not available
for free.</p>

</p>

<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>gameId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type struct-type">Game</span></code></td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
</tr>
<tr>
<td><code>updatedAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
</tr>
<tr>
<td><code>ownerId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>Build


<p>
<p>Build contains information about a specific build</p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Site-wide unique identifier generated by itch.io</p>
</td>
</tr>
<tr>
<td><code>parentBuildId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Identifier of the build before this one on the same channel,
or 0 if this is the initial build.</p>
</td>
</tr>
<tr>
<td><code>state</code></td>
<td><code class="typename"><span class="type enum-type" data-tip-selector="#BuildState__TypeHint">BuildState</span></code></td>
<td><p>State of the build: started, processing, etc.</p>
</td>
</tr>
<tr>
<td><code>version</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Automatically-incremented version number, starting with 1</p>
</td>
</tr>
<tr>
<td><code>userVersion</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Value specified by developer with <code>--userversion</code> when pushing a build
Might not be unique across builds of a given channel.</p>
</td>
</tr>
<tr>
<td><code>files</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#BuildFile__TypeHint">BuildFile</span>[]</code></td>
<td><p>Files associated with this build - often at least an archive,
a signature, and a patch. Some might be missing while the build
is still processing or if processing has failed.</p>
</td>
</tr>
<tr>
<td><code>user</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#User__TypeHint">User</span></code></td>
<td><p>User who pushed the build</p>
</td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
<td><p>Timestamp the build was created at</p>
</td>
</tr>
<tr>
<td><code>updatedAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
<td><p>Timestamp the build was last updated at</p>
</td>
</tr>
</table>


<div id="Build__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>Build <a href="#/?id=build">(Go to definition)</a></p>

<p>
<p>Build contains information about a specific build</p>

</p>

<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>parentBuildId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>state</code></td>
<td><code class="typename"><span class="type enum-type">BuildState</span></code></td>
</tr>
<tr>
<td><code>version</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>userVersion</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>files</code></td>
<td><code class="typename"><span class="type struct-type">BuildFile</span>[]</code></td>
</tr>
<tr>
<td><code>user</code></td>
<td><code class="typename"><span class="type struct-type">User</span></code></td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
</tr>
<tr>
<td><code>updatedAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
</tr>
</table>

</div>

### <em class="enum-type"></em>BuildState


<p>
<p>BuildState describes the state of a build, relative to its initial upload, and
its processing.</p>

</p>

<p>
<span class="header">Values</span> 
</p>


<table class="field-table">
<tr>
<td><code>"started"</code></td>
<td><p>BuildStateStarted is the state of a build from its creation until the initial upload is complete</p>
</td>
</tr>
<tr>
<td><code>"processing"</code></td>
<td><p>BuildStateProcessing is the state of a build from the initial upload&rsquo;s completion to its fully-processed state.
This state does not mean the build is actually being processed right now, it&rsquo;s just queued for processing.</p>
</td>
</tr>
<tr>
<td><code>"completed"</code></td>
<td><p>BuildStateCompleted means the build was successfully processed. Its patch hasn&rsquo;t necessarily been
rediff&rsquo;d yet, but we have the holy (patch,signature,archive) trinity.</p>
</td>
</tr>
<tr>
<td><code>"failed"</code></td>
<td><p>BuildStateFailed means something went wrong with the build. A failing build will not update the channel
head and can be requeued by the itch.io team, although if a new build is pushed before they do,
that new build will &ldquo;win&rdquo;.</p>
</td>
</tr>
</table>


<div id="BuildState__TypeHint" style="display: none;" class="tip-content">
<p><em class="enum-type"></em>BuildState <a href="#/?id=buildstate">(Go to definition)</a></p>

<p>
<p>BuildState describes the state of a build, relative to its initial upload, and
its processing.</p>

</p>

<table class="field-table">
<tr>
<td><code>"started"</code></td>
</tr>
<tr>
<td><code>"processing"</code></td>
</tr>
<tr>
<td><code>"completed"</code></td>
</tr>
<tr>
<td><code>"failed"</code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>BuildFile


<p>
<p>BuildFile contains information about a build&rsquo;s &ldquo;file&rdquo;, which could be its
archive, its signature, its patch, etc.</p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Site-wide unique identifier generated by itch.io</p>
</td>
</tr>
<tr>
<td><code>size</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Size of this build file</p>
</td>
</tr>
<tr>
<td><code>state</code></td>
<td><code class="typename"><span class="type enum-type" data-tip-selector="#BuildFileState__TypeHint">BuildFileState</span></code></td>
<td><p>State of this file: created, uploading, uploaded, etc.</p>
</td>
</tr>
<tr>
<td><code>type</code></td>
<td><code class="typename"><span class="type enum-type" data-tip-selector="#BuildFileType__TypeHint">BuildFileType</span></code></td>
<td><p>Type of this build file: archive, signature, patch, etc.</p>
</td>
</tr>
<tr>
<td><code>subType</code></td>
<td><code class="typename"><span class="type enum-type" data-tip-selector="#BuildFileSubType__TypeHint">BuildFileSubType</span></code></td>
<td><p>Subtype of this build file, usually indicates compression</p>
</td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
<td><p>Date this build file was created at</p>
</td>
</tr>
<tr>
<td><code>updatedAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
<td><p>Date this build file was last updated at</p>
</td>
</tr>
</table>


<div id="BuildFile__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>BuildFile <a href="#/?id=buildfile">(Go to definition)</a></p>

<p>
<p>BuildFile contains information about a build&rsquo;s &ldquo;file&rdquo;, which could be its
archive, its signature, its patch, etc.</p>

</p>

<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>size</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>state</code></td>
<td><code class="typename"><span class="type enum-type">BuildFileState</span></code></td>
</tr>
<tr>
<td><code>type</code></td>
<td><code class="typename"><span class="type enum-type">BuildFileType</span></code></td>
</tr>
<tr>
<td><code>subType</code></td>
<td><code class="typename"><span class="type enum-type">BuildFileSubType</span></code></td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
</tr>
<tr>
<td><code>updatedAt</code></td>
<td><code class="typename"><span class="type builtin-type">Date</span></code></td>
</tr>
</table>

</div>

### <em class="enum-type"></em>BuildFileState


<p>
<p>BuildFileState describes the state of a specific file for a build</p>

</p>

<p>
<span class="header">Values</span> 
</p>


<table class="field-table">
<tr>
<td><code>"created"</code></td>
<td><p>BuildFileStateCreated means the file entry exists on itch.io</p>
</td>
</tr>
<tr>
<td><code>"uploading"</code></td>
<td><p>BuildFileStateUploading means the file is currently being uploaded to storage</p>
</td>
</tr>
<tr>
<td><code>"uploaded"</code></td>
<td><p>BuildFileStateUploaded means the file is ready</p>
</td>
</tr>
<tr>
<td><code>"failed"</code></td>
<td><p>BuildFileStateFailed means the file failed uploading</p>
</td>
</tr>
</table>


<div id="BuildFileState__TypeHint" style="display: none;" class="tip-content">
<p><em class="enum-type"></em>BuildFileState <a href="#/?id=buildfilestate">(Go to definition)</a></p>

<p>
<p>BuildFileState describes the state of a specific file for a build</p>

</p>

<table class="field-table">
<tr>
<td><code>"created"</code></td>
</tr>
<tr>
<td><code>"uploading"</code></td>
</tr>
<tr>
<td><code>"uploaded"</code></td>
</tr>
<tr>
<td><code>"failed"</code></td>
</tr>
</table>

</div>

### <em class="enum-type"></em>BuildFileType


<p>
<p>BuildFileType describes the type of a build file: patch, archive, signature, etc.</p>

</p>

<p>
<span class="header">Values</span> 
</p>


<table class="field-table">
<tr>
<td><code>"patch"</code></td>
<td><p>BuildFileTypePatch describes wharf patch files (.pwr)</p>
</td>
</tr>
<tr>
<td><code>"archive"</code></td>
<td><p>BuildFileTypeArchive describes canonical archive form (.zip)</p>
</td>
</tr>
<tr>
<td><code>"signature"</code></td>
<td><p>BuildFileTypeSignature describes wharf signature files (.pws)</p>
</td>
</tr>
<tr>
<td><code>"manifest"</code></td>
<td><p>BuildFileTypeManifest is reserved</p>
</td>
</tr>
<tr>
<td><code>"unpacked"</code></td>
<td><p>BuildFileTypeUnpacked describes the single file that is in the build (if it was just a single file)</p>
</td>
</tr>
</table>


<div id="BuildFileType__TypeHint" style="display: none;" class="tip-content">
<p><em class="enum-type"></em>BuildFileType <a href="#/?id=buildfiletype">(Go to definition)</a></p>

<p>
<p>BuildFileType describes the type of a build file: patch, archive, signature, etc.</p>

</p>

<table class="field-table">
<tr>
<td><code>"patch"</code></td>
</tr>
<tr>
<td><code>"archive"</code></td>
</tr>
<tr>
<td><code>"signature"</code></td>
</tr>
<tr>
<td><code>"manifest"</code></td>
</tr>
<tr>
<td><code>"unpacked"</code></td>
</tr>
</table>

</div>

### <em class="enum-type"></em>BuildFileSubType


<p>
<p>BuildFileSubType describes the subtype of a build file: mostly its compression
level. For example, rediff&rsquo;d patches are &ldquo;optimized&rdquo;, whereas initial patches are &ldquo;default&rdquo;</p>

</p>

<p>
<span class="header">Values</span> 
</p>


<table class="field-table">
<tr>
<td><code>"default"</code></td>
<td><p>BuildFileSubTypeDefault describes default compression (rsync patches)</p>
</td>
</tr>
<tr>
<td><code>"gzip"</code></td>
<td><p>BuildFileSubTypeGzip is reserved</p>
</td>
</tr>
<tr>
<td><code>"optimized"</code></td>
<td><p>BuildFileSubTypeOptimized describes optimized compression (rediff&rsquo;d / bsdiff patches)</p>
</td>
</tr>
</table>


<div id="BuildFileSubType__TypeHint" style="display: none;" class="tip-content">
<p><em class="enum-type"></em>BuildFileSubType <a href="#/?id=buildfilesubtype">(Go to definition)</a></p>

<p>
<p>BuildFileSubType describes the subtype of a build file: mostly its compression
level. For example, rediff&rsquo;d patches are &ldquo;optimized&rdquo;, whereas initial patches are &ldquo;default&rdquo;</p>

</p>

<table class="field-table">
<tr>
<td><code>"default"</code></td>
</tr>
<tr>
<td><code>"gzip"</code></td>
</tr>
<tr>
<td><code>"optimized"</code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>Verdict


<p>
<p>A Verdict contains a wealth of information on how to &ldquo;launch&rdquo; or &ldquo;open&rdquo; a specific
folder.</p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>basePath</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>BasePath is the absolute path of the folder that was configured</p>
</td>
</tr>
<tr>
<td><code>totalSize</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>TotalSize is the size in bytes of the folder and all its children, recursively</p>
</td>
</tr>
<tr>
<td><code>candidates</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Candidate__TypeHint">Candidate</span>[]</code></td>
<td><p>Candidates is a list of potentially interesting files, with a lot of additional info</p>
</td>
</tr>
</table>


<div id="Verdict__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>Verdict <a href="#/?id=verdict">(Go to definition)</a></p>

<p>
<p>A Verdict contains a wealth of information on how to &ldquo;launch&rdquo; or &ldquo;open&rdquo; a specific
folder.</p>

</p>

<table class="field-table">
<tr>
<td><code>basePath</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>totalSize</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>candidates</code></td>
<td><code class="typename"><span class="type struct-type">Candidate</span>[]</code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>Candidate


<p>
<p>A Candidate is a potentially interesting launch target, be it
a native executable, a Java or Love2D bundle, an HTML index, etc.</p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>path</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Path is relative to the configured folder</p>
</td>
</tr>
<tr>
<td><code>mode</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Mode describes file permissions</p>
</td>
</tr>
<tr>
<td><code>depth</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Depth is the number of path elements leading up to this candidate</p>
</td>
</tr>
<tr>
<td><code>flavor</code></td>
<td><code class="typename"><span class="type enum-type" data-tip-selector="#Flavor__TypeHint">Flavor</span></code></td>
<td><p>Flavor is the type of a candidate - native, html, jar etc.</p>
</td>
</tr>
<tr>
<td><code>arch</code></td>
<td><code class="typename"><span class="type enum-type" data-tip-selector="#Arch__TypeHint">Arch</span></code></td>
<td><p>Arch describes the architecture of a candidate (where relevant)</p>
</td>
</tr>
<tr>
<td><code>size</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Size is the size of the candidate&rsquo;s file, in bytes</p>
</td>
</tr>
<tr>
<td><code>spell</code></td>
<td><code class="typename"><span class="type builtin-type">string</span>[]</code></td>
<td><p><span class="tag">Optional</span> Spell contains raw output from <a href="https://github.com/fasterthanlime/wizardry">https://github.com/fasterthanlime/wizardry</a></p>
</td>
</tr>
<tr>
<td><code>windowsInfo</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#WindowsInfo__TypeHint">WindowsInfo</span></code></td>
<td><p><span class="tag">Optional</span> WindowsInfo contains information specific to native Windows candidates</p>
</td>
</tr>
<tr>
<td><code>linuxInfo</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#LinuxInfo__TypeHint">LinuxInfo</span></code></td>
<td><p><span class="tag">Optional</span> LinuxInfo contains information specific to native Linux candidates</p>
</td>
</tr>
<tr>
<td><code>macosInfo</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#MacosInfo__TypeHint">MacosInfo</span></code></td>
<td><p><span class="tag">Optional</span> MacosInfo contains information specific to native macOS candidates</p>
</td>
</tr>
<tr>
<td><code>loveInfo</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#LoveInfo__TypeHint">LoveInfo</span></code></td>
<td><p><span class="tag">Optional</span> LoveInfo contains information specific to Love2D bundles (<code>.love</code> files)</p>
</td>
</tr>
<tr>
<td><code>scriptInfo</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#ScriptInfo__TypeHint">ScriptInfo</span></code></td>
<td><p><span class="tag">Optional</span> ScriptInfo contains information specific to shell scripts (<code>.sh</code>, <code>.bat</code> etc.)</p>
</td>
</tr>
<tr>
<td><code>jarInfo</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#JarInfo__TypeHint">JarInfo</span></code></td>
<td><p><span class="tag">Optional</span> JarInfo contains information specific to Java archives (<code>.jar</code> files)</p>
</td>
</tr>
</table>


<div id="Candidate__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>Candidate <a href="#/?id=candidate">(Go to definition)</a></p>

<p>
<p>A Candidate is a potentially interesting launch target, be it
a native executable, a Java or Love2D bundle, an HTML index, etc.</p>

</p>

<table class="field-table">
<tr>
<td><code>path</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>mode</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>depth</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>flavor</code></td>
<td><code class="typename"><span class="type enum-type">Flavor</span></code></td>
</tr>
<tr>
<td><code>arch</code></td>
<td><code class="typename"><span class="type enum-type">Arch</span></code></td>
</tr>
<tr>
<td><code>size</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>spell</code></td>
<td><code class="typename"><span class="type builtin-type">string</span>[]</code></td>
</tr>
<tr>
<td><code>windowsInfo</code></td>
<td><code class="typename"><span class="type struct-type">WindowsInfo</span></code></td>
</tr>
<tr>
<td><code>linuxInfo</code></td>
<td><code class="typename"><span class="type struct-type">LinuxInfo</span></code></td>
</tr>
<tr>
<td><code>macosInfo</code></td>
<td><code class="typename"><span class="type struct-type">MacosInfo</span></code></td>
</tr>
<tr>
<td><code>loveInfo</code></td>
<td><code class="typename"><span class="type struct-type">LoveInfo</span></code></td>
</tr>
<tr>
<td><code>scriptInfo</code></td>
<td><code class="typename"><span class="type struct-type">ScriptInfo</span></code></td>
</tr>
<tr>
<td><code>jarInfo</code></td>
<td><code class="typename"><span class="type struct-type">JarInfo</span></code></td>
</tr>
</table>

</div>

### <em class="enum-type"></em>Flavor


<p>
<p>Flavor describes whether we&rsquo;re dealing with a native executables, a Java archive, a love2d bundle, etc.</p>

</p>

<p>
<span class="header">Values</span> 
</p>


<table class="field-table">
<tr>
<td><code>"linux"</code></td>
<td><p>FlavorNativeLinux denotes native linux executables</p>
</td>
</tr>
<tr>
<td><code>"macos"</code></td>
<td><p>ExecNativeMacos denotes native macOS executables</p>
</td>
</tr>
<tr>
<td><code>"windows"</code></td>
<td><p>FlavorPe denotes native windows executables</p>
</td>
</tr>
<tr>
<td><code>"app-macos"</code></td>
<td><p>FlavorAppMacos denotes a macOS app bundle</p>
</td>
</tr>
<tr>
<td><code>"script"</code></td>
<td><p>FlavorScript denotes scripts starting with a shebang (#!)</p>
</td>
</tr>
<tr>
<td><code>"windows-script"</code></td>
<td><p>FlavorScriptWindows denotes windows scripts (.bat or .cmd)</p>
</td>
</tr>
<tr>
<td><code>"jar"</code></td>
<td><p>FlavorJar denotes a .jar archive with a Main-Class</p>
</td>
</tr>
<tr>
<td><code>"html"</code></td>
<td><p>FlavorHTML denotes an index html file</p>
</td>
</tr>
<tr>
<td><code>"love"</code></td>
<td><p>FlavorLove denotes a love package</p>
</td>
</tr>
</table>


<div id="Flavor__TypeHint" style="display: none;" class="tip-content">
<p><em class="enum-type"></em>Flavor <a href="#/?id=flavor">(Go to definition)</a></p>

<p>
<p>Flavor describes whether we&rsquo;re dealing with a native executables, a Java archive, a love2d bundle, etc.</p>

</p>

<table class="field-table">
<tr>
<td><code>"linux"</code></td>
</tr>
<tr>
<td><code>"macos"</code></td>
</tr>
<tr>
<td><code>"windows"</code></td>
</tr>
<tr>
<td><code>"app-macos"</code></td>
</tr>
<tr>
<td><code>"script"</code></td>
</tr>
<tr>
<td><code>"windows-script"</code></td>
</tr>
<tr>
<td><code>"jar"</code></td>
</tr>
<tr>
<td><code>"html"</code></td>
</tr>
<tr>
<td><code>"love"</code></td>
</tr>
</table>

</div>

### <em class="enum-type"></em>Arch


<p>
<p>The architecture of an executable</p>

</p>

<p>
<span class="header">Values</span> 
</p>


<table class="field-table">
<tr>
<td><code>"386"</code></td>
<td><p>32-bit</p>
</td>
</tr>
<tr>
<td><code>"amd64"</code></td>
<td><p>64-bit</p>
</td>
</tr>
</table>


<div id="Arch__TypeHint" style="display: none;" class="tip-content">
<p><em class="enum-type"></em>Arch <a href="#/?id=arch">(Go to definition)</a></p>

<p>
<p>The architecture of an executable</p>

</p>

<table class="field-table">
<tr>
<td><code>"386"</code></td>
</tr>
<tr>
<td><code>"amd64"</code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>WindowsInfo


<p>
<p>Contains information specific to native windows executables
or installer packages.</p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>installerType</code></td>
<td><code class="typename"><span class="type enum-type" data-tip-selector="#WindowsInstallerType__TypeHint">WindowsInstallerType</span></code></td>
<td><p><span class="tag">Optional</span> Particular type of installer (msi, inno, etc.)</p>
</td>
</tr>
<tr>
<td><code>uninstaller</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span> True if we suspect this might be an uninstaller rather than an installer</p>
</td>
</tr>
<tr>
<td><code>gui</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span> Is this executable marked as GUI? This can be false and still pop a GUI, it&rsquo;s just a hint.</p>
</td>
</tr>
<tr>
<td><code>dotNet</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span> Is this a .NET assembly?</p>
</td>
</tr>
</table>


<div id="WindowsInfo__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>WindowsInfo <a href="#/?id=windowsinfo">(Go to definition)</a></p>

<p>
<p>Contains information specific to native windows executables
or installer packages.</p>

</p>

<table class="field-table">
<tr>
<td><code>installerType</code></td>
<td><code class="typename"><span class="type enum-type">WindowsInstallerType</span></code></td>
</tr>
<tr>
<td><code>uninstaller</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>gui</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>dotNet</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>

### <em class="enum-type"></em>WindowsInstallerType


<p>
<p>Which particular type of windows-specific installer</p>

</p>

<p>
<span class="header">Values</span> 
</p>


<table class="field-table">
<tr>
<td><code>"msi"</code></td>
<td><p>Microsoft install packages (<code>.msi</code> files)</p>
</td>
</tr>
<tr>
<td><code>"inno"</code></td>
<td><p>InnoSetup installers</p>
</td>
</tr>
<tr>
<td><code>"nsis"</code></td>
<td><p>NSIS installers</p>
</td>
</tr>
<tr>
<td><code>"archive"</code></td>
<td><p>Self-extracting installers that 7-zip knows how to extract</p>
</td>
</tr>
</table>


<div id="WindowsInstallerType__TypeHint" style="display: none;" class="tip-content">
<p><em class="enum-type"></em>WindowsInstallerType <a href="#/?id=windowsinstallertype">(Go to definition)</a></p>

<p>
<p>Which particular type of windows-specific installer</p>

</p>

<table class="field-table">
<tr>
<td><code>"msi"</code></td>
</tr>
<tr>
<td><code>"inno"</code></td>
</tr>
<tr>
<td><code>"nsis"</code></td>
</tr>
<tr>
<td><code>"archive"</code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>MacosInfo


<p>
<p>Contains information specific to native macOS executables
or app bundles.</p>

</p>

<p>
<span class="header">Fields</span> <em>none</em>
</p>


<div id="MacosInfo__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>MacosInfo <a href="#/?id=macosinfo">(Go to definition)</a></p>

<p>
<p>Contains information specific to native macOS executables
or app bundles.</p>

</p>
</div>

### <em class="struct-type"></em>LinuxInfo


<p>
<p>Contains information specific to native Linux executables</p>

</p>

<p>
<span class="header">Fields</span> <em>none</em>
</p>


<div id="LinuxInfo__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>LinuxInfo <a href="#/?id=linuxinfo">(Go to definition)</a></p>

<p>
<p>Contains information specific to native Linux executables</p>

</p>
</div>

### <em class="struct-type"></em>LoveInfo


<p>
<p>Contains information specific to Love2D bundles</p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>version</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p><span class="tag">Optional</span> The version of love2D required to open this bundle. May be empty</p>
</td>
</tr>
</table>


<div id="LoveInfo__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>LoveInfo <a href="#/?id=loveinfo">(Go to definition)</a></p>

<p>
<p>Contains information specific to Love2D bundles</p>

</p>

<table class="field-table">
<tr>
<td><code>version</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>ScriptInfo


<p>
<p>Contains information specific to shell scripts</p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>interpreter</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p><span class="tag">Optional</span> Something like <code>/bin/bash</code></p>
</td>
</tr>
</table>


<div id="ScriptInfo__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>ScriptInfo <a href="#/?id=scriptinfo">(Go to definition)</a></p>

<p>
<p>Contains information specific to shell scripts</p>

</p>

<table class="field-table">
<tr>
<td><code>interpreter</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>JarInfo


<p>
<p>Contains information specific to Java archives</p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>mainClass</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p><span class="tag">Optional</span> The main Java class as specified by the manifest included in the .jar (if any)</p>
</td>
</tr>
</table>


<div id="JarInfo__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>JarInfo <a href="#/?id=jarinfo">(Go to definition)</a></p>

<p>
<p>Contains information specific to Java archives</p>

</p>

<table class="field-table">
<tr>
<td><code>mainClass</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### <em class="struct-type"></em>Receipt


<p>
<p>A Receipt describes what was installed to a specific folder.</p>

<p>It&rsquo;s compressed and written to <code>./.itch/receipt.json.gz</code> every
time an install operation completes successfully, and is used
in further install operations to make sure ghosts are busted and/or
angels are saved.</p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td><p>The itch.io game installed at this location</p>
</td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Upload__TypeHint">Upload</span></code></td>
<td><p>The itch.io upload installed at this location</p>
</td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type struct-type" data-tip-selector="#Build__TypeHint">Build</span></code></td>
<td><p>The itch.io build installed at this location. Null for non-wharf upload.</p>
</td>
</tr>
<tr>
<td><code>files</code></td>
<td><code class="typename"><span class="type builtin-type">string</span>[]</code></td>
<td><p>A list of installed files (slash-separated paths, relative to install folder)</p>
</td>
</tr>
<tr>
<td><code>installerName</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p><span class="tag">Optional</span> The installer used to install at this location</p>
</td>
</tr>
<tr>
<td><code>msiProductCode</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p><span class="tag">Optional</span> If this was installed from an MSI package, the product code,
used for a clean uninstall.</p>
</td>
</tr>
</table>


<div id="Receipt__TypeHint" style="display: none;" class="tip-content">
<p><em class="struct-type"></em>Receipt <a href="#/?id=receipt">(Go to definition)</a></p>

<p>
<p>A Receipt describes what was installed to a specific folder.</p>

<p>It&rsquo;s compressed and written to <code>./.itch/receipt.json.gz</code> every
time an install operation completes successfully, and is used
in further install operations to make sure ghosts are busted and/or
angels are saved.</p>

</p>

<table class="field-table">
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type struct-type">Game</span></code></td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type struct-type">Upload</span></code></td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type struct-type">Build</span></code></td>
</tr>
<tr>
<td><code>files</code></td>
<td><code class="typename"><span class="type builtin-type">string</span>[]</code></td>
</tr>
<tr>
<td><code>installerName</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>msiProductCode</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>


