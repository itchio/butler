# Caution: this is a draft

!> This document is a draft! It should not be used yet for implementing
   clients for butlerd. The API and recommendations are still subject to change.

# Overview

butlerd (butler daemon) is a JSON-RPC 2.0 service.

It is used heavily by the [itch.io app](https://itch.io/app) as of version 25.x.

## Starting an instance of the daemon

To start an instance, run:

```bash
butler daemon --json --dbpath path/to/butler.db
```

Use the `--log` command-line option to log all TCP message exchanges.

## Making requests

By default, butlerd listens over TCP. It'll let the OS pick a random port on startup.

When started, it will output a line of JSON to stdout with the following structure:

```json
{
  "secret": "<some secret>",
  "tcp": {
    "address":"127.0.0.1:53702"
  },
  "time": 1563196004,
  "type": "butlerd/listen-notification"
}
```

It's important that you **do not hardcode** port numbers in your client, but rather
parse butler's standard output line by line, trying to interpret each of these
as JSON, and only connecting when you get an object with `type` set to
`butlerd/listen-notification`.

butler may output lines to stdout that are not JSON - your client should not
crash if that is the case, but just ignore (or log) them.

## JSON-RPC 2.0 over TCP

Each peer (butlerd, and your client) can send requests, like these:

```json
{
  "jsonrpc": "2.0",
  "id": 0,
  "method": "Version.Get",
  "params": {}
}
```

(For this section, JSON is formatted on multiple lines to make it easier to read.
However, over TCP, each JSON object is on one single line).

And gets back either errors:

```json
{
  "jsonrpc": "2.0",
  "id": 0,
  "error": {
    "code": 100,
    "message": "Something bad happened"
  },
}
```

or results:

```json
{
  "jsonrpc": "2.0",
  "id": 0,
  "result": {
    "version": "v17.0.0",
    "versionString": "v17.0.0, built in the future"
  },
}
```

Both sides can also send notifications:

```json
{
  "jsonrpc": "2.0",
  "method": "Log",
  "params": {
    "msg": "Just a log message, it doesn't need a reply so it doesn't have an ID"
  }
}
```

For JSON-RPC 2.0 over TCP, we're sending UTF-8 "\n"-separated lines. Each line
can be a request, a reply (error or result), or a notification.

(For more on json-rpc 2.0, review [the specification](https://www.jsonrpc.org/specification))

For example, <http://github.com/itchio/cutter> uses the TCP transport. To
use it in your application, use the `--transport tcp` option of the daemon command.

Note: before any other endpoints can be called, `Meta.Authenticate` needs to be called
with the secret included in the `butlerd/listen-notification` JSON line printed to stdout.

```json
{
  "jsonrpc": "2.0",
  "method": "Meta.Authenticate",
  "params": {
    "secret": "<whatever butler generated and printed as 'secret' in its JSON line to stdout>",
  }
}
```

## Instances and connections

The recommended way to use butlerd is to have a **single instance**, but
**multiple connections**.

Multiple connections are useful because JSON-RPC notifications are not tied
to specific requests.

Long-running operations, like performing an install, or a launch, benefit
from having their own connection, so that their notifications can
be isolated from the rest, and show UI relevant to the item being installed
or launched.

## Making sure butlerd exits at the same time as your process

Depending on how you start butlerd, there's a chance that it'll keep running
after your application exits - if you don't want that to happen ever,
you can pass your application's PID (process identifier) with the `--destiny-pid PID`
command-line option.

> Why destiny? Because if you pass that flag, two processes' destiny are linked.
> If your process exits, butlerd does too.

You can use `--destiny-pid` several times, to specify multiple PIDs to watch for.
This is only useful in very rare cases (such as... our integration testing setup),
but there, now it's documented.

## Updating

Clients are responsible for regularly checking for butler updates, and
installing them.

### HTTP endpoints

Use the following HTTP endpoint to check for a newer version:

  * <https://broth.itch.ovh/butler/windows-amd64/LATEST>

Where `windows-amd64` is one of:

  * `windows-386` - 32-bit Windows
  * `windows-amd64` - 64-bit Windows
  * `linux-amd64` - 64-bit Linux
  * `darwin-amd64` - 64-bit macOS

`LATEST` is a text file that contains a version number.

For example, if the contents of `LATEST` is `11.1.0`, then
the latest version of butler can be downloaded via:

  * <https://broth.itch.ovh/butler/windows-amd64/11.1.0/butler/.zip>

Make sure when you unzip it, that the executable bit is set on Linux & macOS.

(The permissions are set properly in the zip file, but not all zip extractors
faithfully restore them)

### Friendly update deployment

See <https://github.com/itchio/itch/issues/1721>


# Messages


## Utilities Category

### Meta.Authenticate (client request)


<p>
<p>When using TCP transport, must be the first message sent</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>secret</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>ok</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td></td>
</tr>
</table>


<div id="MetaAuthenticateParams__TypeHint" class="tip-content">
<p>Meta.Authenticate (client request) <a href="#/?id=metaauthenticate-client-request">(Go to definition)</a></p>

<p>
<p>When using TCP transport, must be the first message sent</p>

</p>

<table class="field-table">
<tr>
<td><code>secret</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>


<div id="MetaAuthenticateResult__TypeHint" class="tip-content">
<p>MetaAuthenticate  <a href="#/?id=metaauthenticate-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>ok</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>

### Meta.Flow (client request)


<p>
<p>When called, defines the entire duration of the daemon&rsquo;s life.</p>

<p>Cancelling that conversation (or closing the TCP connection) will
shut down the daemon after all other requests have finished. This
allows gracefully switching to another daemon.</p>

<p>This conversation is also used to send all global notifications,
regarding data that&rsquo;s fetched, network state, etc.</p>

<p>Note that this call never returns - you have to cancel it when you&rsquo;re
done with the daemon.</p>

</p>

<p>
<span class="header">Parameters</span> <em>none</em>
</p>



<p>
<span class="header">Result</span> <em>none</em>
</p>


<div id="MetaFlowParams__TypeHint" class="tip-content">
<p>Meta.Flow (client request) <a href="#/?id=metaflow-client-request">(Go to definition)</a></p>

<p>
<p>When called, defines the entire duration of the daemon&rsquo;s life.</p>

<p>Cancelling that conversation (or closing the TCP connection) will
shut down the daemon after all other requests have finished. This
allows gracefully switching to another daemon.</p>

<p>This conversation is also used to send all global notifications,
regarding data that&rsquo;s fetched, network state, etc.</p>

<p>Note that this call never returns - you have to cancel it when you&rsquo;re
done with the daemon.</p>

</p>
</div>


<div id="MetaFlowResult__TypeHint" class="tip-content">
<p>MetaFlow  <a href="#/?id=metaflow-">(Go to definition)</a></p>

</div>

### Meta.Shutdown (client request)


<p>
<p>When called, gracefully shutdown the butler daemon.</p>

</p>

<p>
<span class="header">Parameters</span> <em>none</em>
</p>



<p>
<span class="header">Result</span> <em>none</em>
</p>


<div id="MetaShutdownParams__TypeHint" class="tip-content">
<p>Meta.Shutdown (client request) <a href="#/?id=metashutdown-client-request">(Go to definition)</a></p>

<p>
<p>When called, gracefully shutdown the butler daemon.</p>

</p>
</div>


<div id="MetaShutdownResult__TypeHint" class="tip-content">
<p>MetaShutdown  <a href="#/?id=metashutdown-">(Go to definition)</a></p>

</div>

### MetaFlowEstablished (notification)


<p>
<p>The first notification sent when <code class="typename"><span class="type" data-tip-selector="#MetaFlowParams__TypeHint">Meta.Flow</span></code> is called.</p>

</p>

<p>
<span class="header">Payload</span> 
</p>


<table class="field-table">
<tr>
<td><code>pid</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>The identifier of the daemon process for which the flow was established</p>
</td>
</tr>
</table>


<div id="MetaFlowEstablishedNotification__TypeHint" class="tip-content">
<p>MetaFlowEstablished (notification) <a href="#/?id=metaflowestablished-notification">(Go to definition)</a></p>

<p>
<p>The first notification sent when <code class="typename"><span class="type">Meta.Flow</span></code> is called.</p>

</p>

<table class="field-table">
<tr>
<td><code>pid</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### Version.Get (client request)


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


<div id="VersionGetParams__TypeHint" class="tip-content">
<p>Version.Get (client request) <a href="#/?id=versionget-client-request">(Go to definition)</a></p>

<p>
<p>Retrieves the version of the butler instance the client
is connected to.</p>

<p>This endpoint is meant to gather information when reporting
issues, rather than feature sniffing. Conforming clients should
automatically download new versions of butler, see the <strong>Updating</strong> section.</p>

</p>
</div>


<div id="VersionGetResult__TypeHint" class="tip-content">
<p>VersionGet  <a href="#/?id=versionget-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>version</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>versionString</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### Network.SetSimulateOffline (client request)



<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>enabled</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>If true, all operations after this point will behave
as if there were no network connections</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> <em>none</em>
</p>


<div id="NetworkSetSimulateOfflineParams__TypeHint" class="tip-content">
<p>Network.SetSimulateOffline (client request) <a href="#/?id=networksetsimulateoffline-client-request">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>enabled</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>


<div id="NetworkSetSimulateOfflineResult__TypeHint" class="tip-content">
<p>NetworkSetSimulateOffline  <a href="#/?id=networksetsimulateoffline-">(Go to definition)</a></p>

</div>

### Network.SetBandwidthThrottle (client request)



<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>enabled</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>If true, will limit. If false, will clear any bandwidth throttles in place</p>
</td>
</tr>
<tr>
<td><code>rate</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>The target bandwidth, in kbps</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> <em>none</em>
</p>


<div id="NetworkSetBandwidthThrottleParams__TypeHint" class="tip-content">
<p>Network.SetBandwidthThrottle (client request) <a href="#/?id=networksetbandwidththrottle-client-request">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>enabled</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>rate</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>


<div id="NetworkSetBandwidthThrottleResult__TypeHint" class="tip-content">
<p>NetworkSetBandwidthThrottle  <a href="#/?id=networksetbandwidththrottle-">(Go to definition)</a></p>

</div>


## Profile Category

### Profile.List (client request)


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
<td><code class="typename"><span class="type" data-tip-selector="#Profile__TypeHint">Profile</span>[]</code></td>
<td><p>A list of remembered profiles</p>
</td>
</tr>
</table>


<div id="ProfileListParams__TypeHint" class="tip-content">
<p>Profile.List (client request) <a href="#/?id=profilelist-client-request">(Go to definition)</a></p>

<p>
<p>Lists remembered profiles</p>

</p>
</div>


<div id="ProfileListResult__TypeHint" class="tip-content">
<p>ProfileList  <a href="#/?id=profilelist-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>profiles</code></td>
<td><code class="typename"><span class="type">Profile</span>[]</code></td>
</tr>
</table>

</div>

### Profile.LoginWithPassword (client request)


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
<td><code class="typename"><span class="type" data-tip-selector="#Profile__TypeHint">Profile</span></code></td>
<td><p>Information for the new profile, now remembered</p>
</td>
</tr>
<tr>
<td><code>cookie</code></td>
<td><code class="typename"><span class="type builtin-type">{ [key: string]: string }</span></code></td>
<td><p>Profile cookie for website</p>
</td>
</tr>
</table>


<div id="ProfileLoginWithPasswordParams__TypeHint" class="tip-content">
<p>Profile.LoginWithPassword (client request) <a href="#/?id=profileloginwithpassword-client-request">(Go to definition)</a></p>

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


<div id="ProfileLoginWithPasswordResult__TypeHint" class="tip-content">
<p>ProfileLoginWithPassword  <a href="#/?id=profileloginwithpassword-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>profile</code></td>
<td><code class="typename"><span class="type">Profile</span></code></td>
</tr>
<tr>
<td><code>cookie</code></td>
<td><code class="typename"><span class="type builtin-type">{ [key: string]: string }</span></code></td>
</tr>
</table>

</div>

### Profile.LoginWithAPIKey (client request)


<p>
<p>Add a new profile by API key login. This can be used
for integration tests, for example. Note that no cookies
are returned for this kind of login.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>apiKey</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>The API token to use</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>profile</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Profile__TypeHint">Profile</span></code></td>
<td><p>Information for the new profile, now remembered</p>
</td>
</tr>
</table>


<div id="ProfileLoginWithAPIKeyParams__TypeHint" class="tip-content">
<p>Profile.LoginWithAPIKey (client request) <a href="#/?id=profileloginwithapikey-client-request">(Go to definition)</a></p>

<p>
<p>Add a new profile by API key login. This can be used
for integration tests, for example. Note that no cookies
are returned for this kind of login.</p>

</p>

<table class="field-table">
<tr>
<td><code>apiKey</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>


<div id="ProfileLoginWithAPIKeyResult__TypeHint" class="tip-content">
<p>ProfileLoginWithAPIKey  <a href="#/?id=profileloginwithapikey-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>profile</code></td>
<td><code class="typename"><span class="type">Profile</span></code></td>
</tr>
</table>

</div>

### Profile.RequestCaptcha (client caller)


<p>
<p>Ask the user to solve a captcha challenge
Sent during <code class="typename"><span class="type" data-tip-selector="#ProfileLoginWithPasswordParams__TypeHint">Profile.LoginWithPassword</span></code> if certain
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


<div id="ProfileRequestCaptchaParams__TypeHint" class="tip-content">
<p>Profile.RequestCaptcha (client caller) <a href="#/?id=profilerequestcaptcha-client-caller">(Go to definition)</a></p>

<p>
<p>Ask the user to solve a captcha challenge
Sent during <code class="typename"><span class="type">Profile.LoginWithPassword</span></code> if certain
conditions are met.</p>

</p>

<table class="field-table">
<tr>
<td><code>recaptchaUrl</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>


<div id="ProfileRequestCaptchaResult__TypeHint" class="tip-content">
<p>ProfileRequestCaptcha  <a href="#/?id=profilerequestcaptcha-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>recaptchaResponse</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### Profile.RequestTOTP (client caller)


<p>
<p>Ask the user to provide a TOTP token.
Sent during <code class="typename"><span class="type" data-tip-selector="#ProfileLoginWithPasswordParams__TypeHint">Profile.LoginWithPassword</span></code> if the user has
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


<div id="ProfileRequestTOTPParams__TypeHint" class="tip-content">
<p>Profile.RequestTOTP (client caller) <a href="#/?id=profilerequesttotp-client-caller">(Go to definition)</a></p>

<p>
<p>Ask the user to provide a TOTP token.
Sent during <code class="typename"><span class="type">Profile.LoginWithPassword</span></code> if the user has
two-factor authentication enabled.</p>

</p>
</div>


<div id="ProfileRequestTOTPResult__TypeHint" class="tip-content">
<p>ProfileRequestTOTP  <a href="#/?id=profilerequesttotp-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>code</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### Profile.UseSavedLogin (client request)


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
<td><code class="typename"><span class="type" data-tip-selector="#Profile__TypeHint">Profile</span></code></td>
<td><p>Information for the now validated profile</p>
</td>
</tr>
</table>


<div id="ProfileUseSavedLoginParams__TypeHint" class="tip-content">
<p>Profile.UseSavedLogin (client request) <a href="#/?id=profileusesavedlogin-client-request">(Go to definition)</a></p>

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


<div id="ProfileUseSavedLoginResult__TypeHint" class="tip-content">
<p>ProfileUseSavedLogin  <a href="#/?id=profileusesavedlogin-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>profile</code></td>
<td><code class="typename"><span class="type">Profile</span></code></td>
</tr>
</table>

</div>

### Profile.Forget (client request)


<p>
<p>Forgets a remembered profile - it won&rsquo;t appear in the
<code class="typename"><span class="type" data-tip-selector="#ProfileListParams__TypeHint">Profile.List</span></code> results anymore.</p>

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


<div id="ProfileForgetParams__TypeHint" class="tip-content">
<p>Profile.Forget (client request) <a href="#/?id=profileforget-client-request">(Go to definition)</a></p>

<p>
<p>Forgets a remembered profile - it won&rsquo;t appear in the
<code class="typename"><span class="type">Profile.List</span></code> results anymore.</p>

</p>

<table class="field-table">
<tr>
<td><code>profileId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>


<div id="ProfileForgetResult__TypeHint" class="tip-content">
<p>ProfileForget  <a href="#/?id=profileforget-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>success</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>

### Profile.Data.Put (client request)


<p>
<p>Stores some data associated to a profile, by key.</p>

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
<tr>
<td><code>key</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
<tr>
<td><code>value</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
</table>



<p>
<span class="header">Result</span> <em>none</em>
</p>


<div id="ProfileDataPutParams__TypeHint" class="tip-content">
<p>Profile.Data.Put (client request) <a href="#/?id=profiledataput-client-request">(Go to definition)</a></p>

<p>
<p>Stores some data associated to a profile, by key.</p>

</p>

<table class="field-table">
<tr>
<td><code>profileId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>key</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>value</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>


<div id="ProfileDataPutResult__TypeHint" class="tip-content">
<p>ProfileDataPut  <a href="#/?id=profiledataput-">(Go to definition)</a></p>

</div>

### Profile.Data.Get (client request)


<p>
<p>Retrieves some data associated to a profile, by key.</p>

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
<tr>
<td><code>key</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>ok</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>True if the value existed</p>
</td>
</tr>
<tr>
<td><code>value</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
</table>


<div id="ProfileDataGetParams__TypeHint" class="tip-content">
<p>Profile.Data.Get (client request) <a href="#/?id=profiledataget-client-request">(Go to definition)</a></p>

<p>
<p>Retrieves some data associated to a profile, by key.</p>

</p>

<table class="field-table">
<tr>
<td><code>profileId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>key</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>


<div id="ProfileDataGetResult__TypeHint" class="tip-content">
<p>ProfileDataGet  <a href="#/?id=profiledataget-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>ok</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>value</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>


## Search Category

### Search.Games (client request)


<p>
<p>Searches for games.</p>

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
<tr>
<td><code>query</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>games</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Game__TypeHint">Game</span>[]</code></td>
<td></td>
</tr>
</table>


<div id="SearchGamesParams__TypeHint" class="tip-content">
<p>Search.Games (client request) <a href="#/?id=searchgames-client-request">(Go to definition)</a></p>

<p>
<p>Searches for games.</p>

</p>

<table class="field-table">
<tr>
<td><code>profileId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>query</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>


<div id="SearchGamesResult__TypeHint" class="tip-content">
<p>SearchGames  <a href="#/?id=searchgames-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>games</code></td>
<td><code class="typename"><span class="type">Game</span>[]</code></td>
</tr>
</table>

</div>

### Search.Users (client request)


<p>
<p>Searches for users.</p>

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
<tr>
<td><code>query</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>users</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#User__TypeHint">User</span>[]</code></td>
<td></td>
</tr>
</table>


<div id="SearchUsersParams__TypeHint" class="tip-content">
<p>Search.Users (client request) <a href="#/?id=searchusers-client-request">(Go to definition)</a></p>

<p>
<p>Searches for users.</p>

</p>

<table class="field-table">
<tr>
<td><code>profileId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>query</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>


<div id="SearchUsersResult__TypeHint" class="tip-content">
<p>SearchUsers  <a href="#/?id=searchusers-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>users</code></td>
<td><code class="typename"><span class="type">User</span>[]</code></td>
</tr>
</table>

</div>


## Fetch Category

### Fetch.Game (client request)


<p>
<p>Fetches information for an itch.io game.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>gameId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Identifier of game to look for</p>
</td>
</tr>
<tr>
<td><code>fresh</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span> Force an API request</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td><p>Game info</p>
</td>
</tr>
<tr>
<td><code>stale</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span> Marks that a request should be issued afterwards with &lsquo;Fresh&rsquo; set</p>
</td>
</tr>
</table>


<div id="FetchGameParams__TypeHint" class="tip-content">
<p>Fetch.Game (client request) <a href="#/?id=fetchgame-client-request">(Go to definition)</a></p>

<p>
<p>Fetches information for an itch.io game.</p>

</p>

<table class="field-table">
<tr>
<td><code>gameId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>fresh</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>


<div id="FetchGameResult__TypeHint" class="tip-content">
<p>FetchGame  <a href="#/?id=fetchgame-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type">Game</span></code></td>
</tr>
<tr>
<td><code>stale</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>

### Fetch.GameRecords (client request)


<p>
<p>Fetches game records - owned, installed, in collection,
with search, etc. Includes download key info, cave info, etc.</p>

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
<td><code>source</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#GameRecordsSource__TypeHint">GameRecordsSource</span></code></td>
<td><p>Source from which to fetch games</p>
</td>
</tr>
<tr>
<td><code>collectionId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p><span class="tag">Optional</span> Collection ID, required if <code>Source</code> is &ldquo;collection&rdquo;</p>
</td>
</tr>
<tr>
<td><code>limit</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p><span class="tag">Optional</span> Maximum number of games to return at a time</p>
</td>
</tr>
<tr>
<td><code>offset</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p><span class="tag">Optional</span> Games to skip</p>
</td>
</tr>
<tr>
<td><code>search</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p><span class="tag">Optional</span> When specified only shows game titles that contain this string</p>
</td>
</tr>
<tr>
<td><code>sortBy</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p><span class="tag">Optional</span> Criterion to sort by</p>
</td>
</tr>
<tr>
<td><code>filters</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#GameRecordsFilters__TypeHint">GameRecordsFilters</span></code></td>
<td><p><span class="tag">Optional</span> Filters</p>
</td>
</tr>
<tr>
<td><code>reverse</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span></p>
</td>
</tr>
<tr>
<td><code>fresh</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span> If set, will force fresh data</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>records</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#GameRecord__TypeHint">GameRecord</span>[]</code></td>
<td><p>All the records that were fetched</p>
</td>
</tr>
<tr>
<td><code>stale</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span> Marks that a request should be issued afterwards with &lsquo;Fresh&rsquo; set</p>
</td>
</tr>
</table>


<div id="FetchGameRecordsParams__TypeHint" class="tip-content">
<p>Fetch.GameRecords (client request) <a href="#/?id=fetchgamerecords-client-request">(Go to definition)</a></p>

<p>
<p>Fetches game records - owned, installed, in collection,
with search, etc. Includes download key info, cave info, etc.</p>

</p>

<table class="field-table">
<tr>
<td><code>profileId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>source</code></td>
<td><code class="typename"><span class="type">GameRecordsSource</span></code></td>
</tr>
<tr>
<td><code>collectionId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>limit</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>offset</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>search</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>sortBy</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>filters</code></td>
<td><code class="typename"><span class="type">GameRecordsFilters</span></code></td>
</tr>
<tr>
<td><code>reverse</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>fresh</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>


<div id="FetchGameRecordsResult__TypeHint" class="tip-content">
<p>FetchGameRecords  <a href="#/?id=fetchgamerecords-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>records</code></td>
<td><code class="typename"><span class="type">GameRecord</span>[]</code></td>
</tr>
<tr>
<td><code>stale</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>

### Fetch.DownloadKey (client request)


<p>
<p>Fetches a download key</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>downloadKeyId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>profileId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>fresh</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span> Force an API request</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>downloadKey</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#DownloadKey__TypeHint">DownloadKey</span></code></td>
<td></td>
</tr>
<tr>
<td><code>stale</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span> Marks that a request should be issued afterwards with &lsquo;Fresh&rsquo; set</p>
</td>
</tr>
</table>


<div id="FetchDownloadKeyParams__TypeHint" class="tip-content">
<p>Fetch.DownloadKey (client request) <a href="#/?id=fetchdownloadkey-client-request">(Go to definition)</a></p>

<p>
<p>Fetches a download key</p>

</p>

<table class="field-table">
<tr>
<td><code>downloadKeyId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>profileId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>fresh</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>


<div id="FetchDownloadKeyResult__TypeHint" class="tip-content">
<p>FetchDownloadKey  <a href="#/?id=fetchdownloadkey-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>downloadKey</code></td>
<td><code class="typename"><span class="type">DownloadKey</span></code></td>
</tr>
<tr>
<td><code>stale</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>

### Fetch.DownloadKeys (client request)


<p>
<p>Fetches multiple download keys</p>

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
<tr>
<td><code>offset</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p><span class="tag">Optional</span> Number of items to skip</p>
</td>
</tr>
<tr>
<td><code>limit</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p><span class="tag">Optional</span> Max number of results per page (default = 5)</p>
</td>
</tr>
<tr>
<td><code>filters</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#FetchDownloadKeysFilter__TypeHint">FetchDownloadKeysFilter</span></code></td>
<td><p><span class="tag">Optional</span> Filter results</p>
</td>
</tr>
<tr>
<td><code>fresh</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span> Force an API request</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>items</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#DownloadKey__TypeHint">DownloadKey</span>[]</code></td>
<td><p>All the download keys found in the local DB.</p>
</td>
</tr>
<tr>
<td><code>stale</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>Whether the information was fetched from a stale cache,
and could warrant a refresh if online.</p>
</td>
</tr>
</table>


<div id="FetchDownloadKeysParams__TypeHint" class="tip-content">
<p>Fetch.DownloadKeys (client request) <a href="#/?id=fetchdownloadkeys-client-request">(Go to definition)</a></p>

<p>
<p>Fetches multiple download keys</p>

</p>

<table class="field-table">
<tr>
<td><code>profileId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>offset</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>limit</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>filters</code></td>
<td><code class="typename"><span class="type">FetchDownloadKeysFilter</span></code></td>
</tr>
<tr>
<td><code>fresh</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>


<div id="FetchDownloadKeysResult__TypeHint" class="tip-content">
<p>FetchDownloadKeys  <a href="#/?id=fetchdownloadkeys-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>items</code></td>
<td><code class="typename"><span class="type">DownloadKey</span>[]</code></td>
</tr>
<tr>
<td><code>stale</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>

### Fetch.GameUploads (client request)


<p>
<p>Fetches uploads for an itch.io game</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>gameId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Identifier of the game whose uploads we should look for</p>
</td>
</tr>
<tr>
<td><code>compatible</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>Only returns compatible uploads</p>
</td>
</tr>
<tr>
<td><code>fresh</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span> Force an API request</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>uploads</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Upload__TypeHint">Upload</span>[]</code></td>
<td><p>List of uploads</p>
</td>
</tr>
<tr>
<td><code>stale</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span> Marks that a request should be issued
afterwards with &lsquo;Fresh&rsquo; set</p>
</td>
</tr>
</table>


<div id="FetchGameUploadsParams__TypeHint" class="tip-content">
<p>Fetch.GameUploads (client request) <a href="#/?id=fetchgameuploads-client-request">(Go to definition)</a></p>

<p>
<p>Fetches uploads for an itch.io game</p>

</p>

<table class="field-table">
<tr>
<td><code>gameId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>compatible</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>fresh</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>


<div id="FetchGameUploadsResult__TypeHint" class="tip-content">
<p>FetchGameUploads  <a href="#/?id=fetchgameuploads-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>uploads</code></td>
<td><code class="typename"><span class="type">Upload</span>[]</code></td>
</tr>
<tr>
<td><code>stale</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>

### Fetch.User (client request)


<p>
<p>Fetches information for an itch.io user.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>userId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Identifier of the user to look for</p>
</td>
</tr>
<tr>
<td><code>profileId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Profile to use to look upser</p>
</td>
</tr>
<tr>
<td><code>fresh</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span> Force an API request</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>user</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#User__TypeHint">User</span></code></td>
<td><p>User info</p>
</td>
</tr>
<tr>
<td><code>stale</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span> Marks that a request should be issued
afterwards with &lsquo;Fresh&rsquo; set</p>
</td>
</tr>
</table>


<div id="FetchUserParams__TypeHint" class="tip-content">
<p>Fetch.User (client request) <a href="#/?id=fetchuser-client-request">(Go to definition)</a></p>

<p>
<p>Fetches information for an itch.io user.</p>

</p>

<table class="field-table">
<tr>
<td><code>userId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>profileId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>fresh</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>


<div id="FetchUserResult__TypeHint" class="tip-content">
<p>FetchUser  <a href="#/?id=fetchuser-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>user</code></td>
<td><code class="typename"><span class="type">User</span></code></td>
</tr>
<tr>
<td><code>stale</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>

### Fetch.Sale (client request)


<p>
<p>Fetches the best current <em>locally cached</em> sale for a given
game.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>gameId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Identifier of the game for which to look for a sale</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>sale</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Sale__TypeHint">Sale</span></code></td>
<td><p><span class="tag">Optional</span></p>
</td>
</tr>
</table>


<div id="FetchSaleParams__TypeHint" class="tip-content">
<p>Fetch.Sale (client request) <a href="#/?id=fetchsale-client-request">(Go to definition)</a></p>

<p>
<p>Fetches the best current <em>locally cached</em> sale for a given
game.</p>

</p>

<table class="field-table">
<tr>
<td><code>gameId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>


<div id="FetchSaleResult__TypeHint" class="tip-content">
<p>FetchSale  <a href="#/?id=fetchsale-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>sale</code></td>
<td><code class="typename"><span class="type">Sale</span></code></td>
</tr>
</table>

</div>

### Fetch.Collection (client request)


<p>
<p>Fetch a collection&rsquo;s title, gamesCount, etc.
but not its games.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>profileId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Profile to use to fetch collection</p>
</td>
</tr>
<tr>
<td><code>collectionId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Collection to fetch</p>
</td>
</tr>
<tr>
<td><code>fresh</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span> Force an API request before replying.
Usually set after getting &lsquo;stale&rsquo; in the response.</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>collection</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Collection__TypeHint">Collection</span></code></td>
<td><p>Collection info</p>
</td>
</tr>
<tr>
<td><code>stale</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span> True if the info was from local DB and
it should be re-queried using &ldquo;Fresh&rdquo;</p>
</td>
</tr>
</table>


<div id="FetchCollectionParams__TypeHint" class="tip-content">
<p>Fetch.Collection (client request) <a href="#/?id=fetchcollection-client-request">(Go to definition)</a></p>

<p>
<p>Fetch a collection&rsquo;s title, gamesCount, etc.
but not its games.</p>

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
<tr>
<td><code>fresh</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>


<div id="FetchCollectionResult__TypeHint" class="tip-content">
<p>FetchCollection  <a href="#/?id=fetchcollection-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>collection</code></td>
<td><code class="typename"><span class="type">Collection</span></code></td>
</tr>
<tr>
<td><code>stale</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>

### Fetch.Collection.Games (client request)


<p>
<p>Fetches information about a collection and the games it
contains.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>profileId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Profile to use to fetch collection</p>
</td>
</tr>
<tr>
<td><code>collectionId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Identifier of the collection to look for</p>
</td>
</tr>
<tr>
<td><code>limit</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p><span class="tag">Optional</span> Maximum number of games to return at a time.</p>
</td>
</tr>
<tr>
<td><code>search</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p><span class="tag">Optional</span> When specified only shows game titles that contain this string</p>
</td>
</tr>
<tr>
<td><code>sortBy</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p><span class="tag">Optional</span> Criterion to sort by</p>
</td>
</tr>
<tr>
<td><code>filters</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#CollectionGamesFilters__TypeHint">CollectionGamesFilters</span></code></td>
<td><p><span class="tag">Optional</span> Filters</p>
</td>
</tr>
<tr>
<td><code>reverse</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span></p>
</td>
</tr>
<tr>
<td><code>cursor</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Cursor__TypeHint">Cursor</span></code></td>
<td><p><span class="tag">Optional</span> Used for pagination, if specified</p>
</td>
</tr>
<tr>
<td><code>fresh</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span> If set, will force fresh data</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>items</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#CollectionGame__TypeHint">CollectionGame</span>[]</code></td>
<td><p>Requested games for this collection</p>
</td>
</tr>
<tr>
<td><code>nextCursor</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Cursor__TypeHint">Cursor</span></code></td>
<td><p><span class="tag">Optional</span> Use to fetch the next &lsquo;page&rsquo; of results</p>
</td>
</tr>
<tr>
<td><code>stale</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span> If true, re-issue request with &lsquo;Fresh&rsquo;</p>
</td>
</tr>
</table>


<div id="FetchCollectionGamesParams__TypeHint" class="tip-content">
<p>Fetch.Collection.Games (client request) <a href="#/?id=fetchcollectiongames-client-request">(Go to definition)</a></p>

<p>
<p>Fetches information about a collection and the games it
contains.</p>

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
<tr>
<td><code>limit</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>search</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>sortBy</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>filters</code></td>
<td><code class="typename"><span class="type">CollectionGamesFilters</span></code></td>
</tr>
<tr>
<td><code>reverse</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>cursor</code></td>
<td><code class="typename"><span class="type">Cursor</span></code></td>
</tr>
<tr>
<td><code>fresh</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>


<div id="FetchCollectionGamesResult__TypeHint" class="tip-content">
<p>FetchCollectionGames  <a href="#/?id=fetchcollectiongames-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>items</code></td>
<td><code class="typename"><span class="type">CollectionGame</span>[]</code></td>
</tr>
<tr>
<td><code>nextCursor</code></td>
<td><code class="typename"><span class="type">Cursor</span></code></td>
</tr>
<tr>
<td><code>stale</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>

### Fetch.ProfileCollections (client request)


<p>
<p>Lists collections for a profile. Does not contain
games.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>profileId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Profile for which to fetch collections</p>
</td>
</tr>
<tr>
<td><code>limit</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p><span class="tag">Optional</span> Maximum number of collections to return at a time.</p>
</td>
</tr>
<tr>
<td><code>search</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p><span class="tag">Optional</span> When specified only shows collection titles that contain this string</p>
</td>
</tr>
<tr>
<td><code>sortBy</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p><span class="tag">Optional</span> Criterion to sort by</p>
</td>
</tr>
<tr>
<td><code>reverse</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span></p>
</td>
</tr>
<tr>
<td><code>cursor</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Cursor__TypeHint">Cursor</span></code></td>
<td><p><span class="tag">Optional</span> Used for pagination, if specified</p>
</td>
</tr>
<tr>
<td><code>fresh</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span> If set, will force fresh data</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>items</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Collection__TypeHint">Collection</span>[]</code></td>
<td><p>Collections belonging to the profile</p>
</td>
</tr>
<tr>
<td><code>nextCursor</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Cursor__TypeHint">Cursor</span></code></td>
<td><p><span class="tag">Optional</span> Used to fetch the next page</p>
</td>
</tr>
<tr>
<td><code>stale</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span> If true, re-issue request with &ldquo;Fresh&rdquo;</p>
</td>
</tr>
</table>


<div id="FetchProfileCollectionsParams__TypeHint" class="tip-content">
<p>Fetch.ProfileCollections (client request) <a href="#/?id=fetchprofilecollections-client-request">(Go to definition)</a></p>

<p>
<p>Lists collections for a profile. Does not contain
games.</p>

</p>

<table class="field-table">
<tr>
<td><code>profileId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>limit</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>search</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>sortBy</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>reverse</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>cursor</code></td>
<td><code class="typename"><span class="type">Cursor</span></code></td>
</tr>
<tr>
<td><code>fresh</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>


<div id="FetchProfileCollectionsResult__TypeHint" class="tip-content">
<p>FetchProfileCollections  <a href="#/?id=fetchprofilecollections-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>items</code></td>
<td><code class="typename"><span class="type">Collection</span>[]</code></td>
</tr>
<tr>
<td><code>nextCursor</code></td>
<td><code class="typename"><span class="type">Cursor</span></code></td>
</tr>
<tr>
<td><code>stale</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>

### Fetch.ProfileGames (client request)



<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>profileId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Profile for which to fetch games</p>
</td>
</tr>
<tr>
<td><code>limit</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p><span class="tag">Optional</span> Maximum number of items to return at a time.</p>
</td>
</tr>
<tr>
<td><code>search</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p><span class="tag">Optional</span> When specified only shows game titles that contain this string</p>
</td>
</tr>
<tr>
<td><code>sortBy</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p><span class="tag">Optional</span> Criterion to sort by</p>
</td>
</tr>
<tr>
<td><code>filters</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#ProfileGameFilters__TypeHint">ProfileGameFilters</span></code></td>
<td><p><span class="tag">Optional</span> Filters</p>
</td>
</tr>
<tr>
<td><code>reverse</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span></p>
</td>
</tr>
<tr>
<td><code>cursor</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Cursor__TypeHint">Cursor</span></code></td>
<td><p><span class="tag">Optional</span> Used for pagination, if specified</p>
</td>
</tr>
<tr>
<td><code>fresh</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span> If set, will force fresh data</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>items</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#ProfileGame__TypeHint">ProfileGame</span>[]</code></td>
<td><p>Profile games</p>
</td>
</tr>
<tr>
<td><code>nextCursor</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Cursor__TypeHint">Cursor</span></code></td>
<td><p><span class="tag">Optional</span> Used to fetch the next page</p>
</td>
</tr>
<tr>
<td><code>stale</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span> If true, re-issue request with &ldquo;Fresh&rdquo;</p>
</td>
</tr>
</table>


<div id="FetchProfileGamesParams__TypeHint" class="tip-content">
<p>Fetch.ProfileGames (client request) <a href="#/?id=fetchprofilegames-client-request">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>profileId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>limit</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>search</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>sortBy</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>filters</code></td>
<td><code class="typename"><span class="type">ProfileGameFilters</span></code></td>
</tr>
<tr>
<td><code>reverse</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>cursor</code></td>
<td><code class="typename"><span class="type">Cursor</span></code></td>
</tr>
<tr>
<td><code>fresh</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>


<div id="FetchProfileGamesResult__TypeHint" class="tip-content">
<p>FetchProfileGames  <a href="#/?id=fetchprofilegames-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>items</code></td>
<td><code class="typename"><span class="type">ProfileGame</span>[]</code></td>
</tr>
<tr>
<td><code>nextCursor</code></td>
<td><code class="typename"><span class="type">Cursor</span></code></td>
</tr>
<tr>
<td><code>stale</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>

### Fetch.ProfileOwnedKeys (client request)



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
<td><code>limit</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p><span class="tag">Optional</span> Maximum number of owned keys to return at a time.</p>
</td>
</tr>
<tr>
<td><code>search</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p><span class="tag">Optional</span> When specified only shows game titles that contain this string</p>
</td>
</tr>
<tr>
<td><code>sortBy</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p><span class="tag">Optional</span> Criterion to sort by</p>
</td>
</tr>
<tr>
<td><code>filters</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#ProfileOwnedKeysFilters__TypeHint">ProfileOwnedKeysFilters</span></code></td>
<td><p><span class="tag">Optional</span> Filters</p>
</td>
</tr>
<tr>
<td><code>reverse</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span></p>
</td>
</tr>
<tr>
<td><code>cursor</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Cursor__TypeHint">Cursor</span></code></td>
<td><p><span class="tag">Optional</span> Used for pagination, if specified</p>
</td>
</tr>
<tr>
<td><code>fresh</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span> If set, will force fresh data</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>items</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#DownloadKey__TypeHint">DownloadKey</span>[]</code></td>
<td><p>Download keys fetched for profile</p>
</td>
</tr>
<tr>
<td><code>nextCursor</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Cursor__TypeHint">Cursor</span></code></td>
<td><p><span class="tag">Optional</span> Used to fetch the next page</p>
</td>
</tr>
<tr>
<td><code>stale</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span> If true, re-issue request with &ldquo;Fresh&rdquo;</p>
</td>
</tr>
</table>


<div id="FetchProfileOwnedKeysParams__TypeHint" class="tip-content">
<p>Fetch.ProfileOwnedKeys (client request) <a href="#/?id=fetchprofileownedkeys-client-request">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>profileId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>limit</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>search</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>sortBy</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>filters</code></td>
<td><code class="typename"><span class="type">ProfileOwnedKeysFilters</span></code></td>
</tr>
<tr>
<td><code>reverse</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>cursor</code></td>
<td><code class="typename"><span class="type">Cursor</span></code></td>
</tr>
<tr>
<td><code>fresh</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>


<div id="FetchProfileOwnedKeysResult__TypeHint" class="tip-content">
<p>FetchProfileOwnedKeys  <a href="#/?id=fetchprofileownedkeys-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>items</code></td>
<td><code class="typename"><span class="type">DownloadKey</span>[]</code></td>
</tr>
<tr>
<td><code>nextCursor</code></td>
<td><code class="typename"><span class="type">Cursor</span></code></td>
</tr>
<tr>
<td><code>stale</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>

### Fetch.Commons (client request)



<p>
<span class="header">Parameters</span> <em>none</em>
</p>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>downloadKeys</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#DownloadKeySummary__TypeHint">DownloadKeySummary</span>[]</code></td>
<td></td>
</tr>
<tr>
<td><code>caves</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#CaveSummary__TypeHint">CaveSummary</span>[]</code></td>
<td></td>
</tr>
<tr>
<td><code>installLocations</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#InstallLocationSummary__TypeHint">InstallLocationSummary</span>[]</code></td>
<td></td>
</tr>
</table>


<div id="FetchCommonsParams__TypeHint" class="tip-content">
<p>Fetch.Commons (client request) <a href="#/?id=fetchcommons-client-request">(Go to definition)</a></p>

</div>


<div id="FetchCommonsResult__TypeHint" class="tip-content">
<p>FetchCommons  <a href="#/?id=fetchcommons-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>downloadKeys</code></td>
<td><code class="typename"><span class="type">DownloadKeySummary</span>[]</code></td>
</tr>
<tr>
<td><code>caves</code></td>
<td><code class="typename"><span class="type">CaveSummary</span>[]</code></td>
</tr>
<tr>
<td><code>installLocations</code></td>
<td><code class="typename"><span class="type">InstallLocationSummary</span>[]</code></td>
</tr>
</table>

</div>

### Fetch.Caves (client request)


<p>
<p>Retrieve info for all caves.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>limit</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p><span class="tag">Optional</span> Maximum number of caves to return at a time.</p>
</td>
</tr>
<tr>
<td><code>search</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p><span class="tag">Optional</span> When specified only shows game titles that contain this string</p>
</td>
</tr>
<tr>
<td><code>sortBy</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p><span class="tag">Optional</span></p>
</td>
</tr>
<tr>
<td><code>filters</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#CavesFilters__TypeHint">CavesFilters</span></code></td>
<td><p><span class="tag">Optional</span> Filters</p>
</td>
</tr>
<tr>
<td><code>reverse</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span></p>
</td>
</tr>
<tr>
<td><code>cursor</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Cursor__TypeHint">Cursor</span></code></td>
<td><p><span class="tag">Optional</span> Used for pagination, if specified</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>items</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Cave__TypeHint">Cave</span>[]</code></td>
<td></td>
</tr>
<tr>
<td><code>nextCursor</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Cursor__TypeHint">Cursor</span></code></td>
<td><p><span class="tag">Optional</span> Use to fetch the next &lsquo;page&rsquo; of results</p>
</td>
</tr>
</table>


<div id="FetchCavesParams__TypeHint" class="tip-content">
<p>Fetch.Caves (client request) <a href="#/?id=fetchcaves-client-request">(Go to definition)</a></p>

<p>
<p>Retrieve info for all caves.</p>

</p>

<table class="field-table">
<tr>
<td><code>limit</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>search</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>sortBy</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>filters</code></td>
<td><code class="typename"><span class="type">CavesFilters</span></code></td>
</tr>
<tr>
<td><code>reverse</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>cursor</code></td>
<td><code class="typename"><span class="type">Cursor</span></code></td>
</tr>
</table>

</div>


<div id="FetchCavesResult__TypeHint" class="tip-content">
<p>FetchCaves  <a href="#/?id=fetchcaves-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>items</code></td>
<td><code class="typename"><span class="type">Cave</span>[]</code></td>
</tr>
<tr>
<td><code>nextCursor</code></td>
<td><code class="typename"><span class="type">Cursor</span></code></td>
</tr>
</table>

</div>

### Fetch.Cave (client request)


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
<td><code class="typename"><span class="type" data-tip-selector="#Cave__TypeHint">Cave</span></code></td>
<td></td>
</tr>
</table>


<div id="FetchCaveParams__TypeHint" class="tip-content">
<p>Fetch.Cave (client request) <a href="#/?id=fetchcave-client-request">(Go to definition)</a></p>

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


<div id="FetchCaveResult__TypeHint" class="tip-content">
<p>FetchCave  <a href="#/?id=fetchcave-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>cave</code></td>
<td><code class="typename"><span class="type">Cave</span></code></td>
</tr>
</table>

</div>

### Fetch.ExpireAll (client request)


<p>
<p>Mark all local data as stale.</p>

</p>

<p>
<span class="header">Parameters</span> <em>none</em>
</p>



<p>
<span class="header">Result</span> <em>none</em>
</p>


<div id="FetchExpireAllParams__TypeHint" class="tip-content">
<p>Fetch.ExpireAll (client request) <a href="#/?id=fetchexpireall-client-request">(Go to definition)</a></p>

<p>
<p>Mark all local data as stale.</p>

</p>
</div>


<div id="FetchExpireAllResult__TypeHint" class="tip-content">
<p>FetchExpireAll  <a href="#/?id=fetchexpireall-">(Go to definition)</a></p>

</div>


## Install Category

### Game.FindUploads (client request)


<p>
<p>Finds uploads compatible with the current runtime, for a given game.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
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
<td><code class="typename"><span class="type" data-tip-selector="#Upload__TypeHint">Upload</span>[]</code></td>
<td><p>A list of uploads that were found to be compatible.</p>
</td>
</tr>
</table>


<div id="GameFindUploadsParams__TypeHint" class="tip-content">
<p>Game.FindUploads (client request) <a href="#/?id=gamefinduploads-client-request">(Go to definition)</a></p>

<p>
<p>Finds uploads compatible with the current runtime, for a given game.</p>

</p>

<table class="field-table">
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type">Game</span></code></td>
</tr>
</table>

</div>


<div id="GameFindUploadsResult__TypeHint" class="tip-content">
<p>GameFindUploads  <a href="#/?id=gamefinduploads-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>uploads</code></td>
<td><code class="typename"><span class="type">Upload</span>[]</code></td>
</tr>
</table>

</div>

### Install.Queue (client request)


<p>
<p>Queues an install operation to be later performed
via <code class="typename"><span class="type" data-tip-selector="#InstallPerformParams__TypeHint">Install.Perform</span></code>.</p>

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
<td><code>reason</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#DownloadReason__TypeHint">DownloadReason</span></code></td>
<td><p><span class="tag">Optional</span> If unspecified, will default to &lsquo;install&rsquo;</p>
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
<td><code class="typename"><span class="type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td><p><span class="tag">Optional</span> Which game to install.</p>

<p>If unspecified and caveId is specified, the same game will be used.</p>
</td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Upload__TypeHint">Upload</span></code></td>
<td><p><span class="tag">Optional</span> Which upload to install.</p>

<p>If unspecified and caveId is specified, the same upload will be used.</p>
</td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Build__TypeHint">Build</span></code></td>
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
<tr>
<td><code>queueDownload</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span> If set, and the install operation is successfully disambiguated,
will queue it as a download for butler to drive.
See <code class="typename"><span class="type" data-tip-selector="#DownloadsDriveParams__TypeHint">Downloads.Drive</span></code>.</p>
</td>
</tr>
<tr>
<td><code>fastQueue</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span> Don&rsquo;t run install prepare (assume we can just run it at perform time)</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
<tr>
<td><code>reason</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#DownloadReason__TypeHint">DownloadReason</span></code></td>
<td></td>
</tr>
<tr>
<td><code>caveId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td></td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Upload__TypeHint">Upload</span></code></td>
<td></td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Build__TypeHint">Build</span></code></td>
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
<tr>
<td><code>installLocationId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
</table>


<div id="InstallQueueParams__TypeHint" class="tip-content">
<p>Install.Queue (client request) <a href="#/?id=installqueue-client-request">(Go to definition)</a></p>

<p>
<p>Queues an install operation to be later performed
via <code class="typename"><span class="type">Install.Perform</span></code>.</p>

</p>

<table class="field-table">
<tr>
<td><code>caveId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>reason</code></td>
<td><code class="typename"><span class="type">DownloadReason</span></code></td>
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
<td><code class="typename"><span class="type">Game</span></code></td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type">Upload</span></code></td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type">Build</span></code></td>
</tr>
<tr>
<td><code>ignoreInstallers</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>stagingFolder</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>queueDownload</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>fastQueue</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>


<div id="InstallQueueResult__TypeHint" class="tip-content">
<p>InstallQueue  <a href="#/?id=installqueue-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>reason</code></td>
<td><code class="typename"><span class="type">DownloadReason</span></code></td>
</tr>
<tr>
<td><code>caveId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type">Game</span></code></td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type">Upload</span></code></td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type">Build</span></code></td>
</tr>
<tr>
<td><code>installFolder</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>stagingFolder</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>installLocationId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### Install.Plan (client request)


<p>
<p>For modal-first install</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>gameId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>The ID of the game we&rsquo;re planning to install</p>
</td>
</tr>
<tr>
<td><code>downloadSessionId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p><span class="tag">Optional</span> The download session ID to use for this install plan</p>
</td>
</tr>
<tr>
<td><code>uploadId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p><span class="tag">Optional</span></p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td></td>
</tr>
<tr>
<td><code>uploads</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Upload__TypeHint">Upload</span>[]</code></td>
<td></td>
</tr>
<tr>
<td><code>info</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#InstallPlanInfo__TypeHint">InstallPlanInfo</span></code></td>
<td></td>
</tr>
</table>


<div id="InstallPlanParams__TypeHint" class="tip-content">
<p>Install.Plan (client request) <a href="#/?id=installplan-client-request">(Go to definition)</a></p>

<p>
<p>For modal-first install</p>

</p>

<table class="field-table">
<tr>
<td><code>gameId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>downloadSessionId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>uploadId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>


<div id="InstallPlanResult__TypeHint" class="tip-content">
<p>InstallPlan  <a href="#/?id=installplan-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type">Game</span></code></td>
</tr>
<tr>
<td><code>uploads</code></td>
<td><code class="typename"><span class="type">Upload</span>[]</code></td>
</tr>
<tr>
<td><code>info</code></td>
<td><code class="typename"><span class="type">InstallPlanInfo</span></code></td>
</tr>
</table>

</div>

### Caves.SetPinned (client request)



<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>caveId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>ID of the cave to pin/unpin</p>
</td>
</tr>
<tr>
<td><code>pinned</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>Pinned state the cave should have after this call</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> <em>none</em>
</p>


<div id="CavesSetPinnedParams__TypeHint" class="tip-content">
<p>Caves.SetPinned (client request) <a href="#/?id=cavessetpinned-client-request">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>caveId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>pinned</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>


<div id="CavesSetPinnedResult__TypeHint" class="tip-content">
<p>CavesSetPinned  <a href="#/?id=cavessetpinned-">(Go to definition)</a></p>

</div>

### Install.CreateShortcut (client request)


<p>
<p>Create a shortcut for an existing cave .</p>

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
<span class="header">Result</span> <em>none</em>
</p>


<div id="InstallCreateShortcutParams__TypeHint" class="tip-content">
<p>Install.CreateShortcut (client request) <a href="#/?id=installcreateshortcut-client-request">(Go to definition)</a></p>

<p>
<p>Create a shortcut for an existing cave .</p>

</p>

<table class="field-table">
<tr>
<td><code>caveId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>


<div id="InstallCreateShortcutResult__TypeHint" class="tip-content">
<p>InstallCreateShortcut  <a href="#/?id=installcreateshortcut-">(Go to definition)</a></p>

</div>

### Install.Perform (client request)


<p>
<p>Perform an install that was previously queued via
<code class="typename"><span class="type" data-tip-selector="#InstallQueueParams__TypeHint">Install.Queue</span></code>.</p>

<p>Can be cancelled by passing the same <code>ID</code> to <code class="typename"><span class="type" data-tip-selector="#InstallCancelParams__TypeHint">Install.Cancel</span></code>.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>ID that can be later used in <code class="typename"><span class="type" data-tip-selector="#InstallCancelParams__TypeHint">Install.Cancel</span></code></p>
</td>
</tr>
<tr>
<td><code>stagingFolder</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>The folder turned by <code class="typename"><span class="type" data-tip-selector="#InstallQueueParams__TypeHint">Install.Queue</span></code></p>
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
<td><code>events</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#InstallEvent__TypeHint">InstallEvent</span>[]</code></td>
<td></td>
</tr>
</table>


<div id="InstallPerformParams__TypeHint" class="tip-content">
<p>Install.Perform (client request) <a href="#/?id=installperform-client-request">(Go to definition)</a></p>

<p>
<p>Perform an install that was previously queued via
<code class="typename"><span class="type">Install.Queue</span></code>.</p>

<p>Can be cancelled by passing the same <code>ID</code> to <code class="typename"><span class="type">Install.Cancel</span></code>.</p>

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


<div id="InstallPerformResult__TypeHint" class="tip-content">
<p>InstallPerform  <a href="#/?id=installperform-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>caveId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>events</code></td>
<td><code class="typename"><span class="type">InstallEvent</span>[]</code></td>
</tr>
</table>

</div>

### Install.Cancel (client request)


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
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>didCancel</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td></td>
</tr>
</table>


<div id="InstallCancelParams__TypeHint" class="tip-content">
<p>Install.Cancel (client request) <a href="#/?id=installcancel-client-request">(Go to definition)</a></p>

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


<div id="InstallCancelResult__TypeHint" class="tip-content">
<p>InstallCancel  <a href="#/?id=installcancel-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>didCancel</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>

### Uninstall.Perform (client request)


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
<tr>
<td><code>hard</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span> If true, don&rsquo;t attempt to run any uninstallers, just
remove the DB record and burn the install folder to the ground.</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> <em>none</em>
</p>


<div id="UninstallPerformParams__TypeHint" class="tip-content">
<p>Uninstall.Perform (client request) <a href="#/?id=uninstallperform-client-request">(Go to definition)</a></p>

<p>
<p>UninstallParams contains all the parameters needed to perform
an uninstallation for a game via <code class="typename"><span class="type builtin-type">OperationStartParams</span></code>.</p>

</p>

<table class="field-table">
<tr>
<td><code>caveId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>hard</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>


<div id="UninstallPerformResult__TypeHint" class="tip-content">
<p>UninstallPerform  <a href="#/?id=uninstallperform-">(Go to definition)</a></p>

</div>

### Install.VersionSwitch.Queue (client request)


<p>
<p>Prepare to queue a version switch. The client will
receive an <code class="typename"><span class="type" data-tip-selector="#InstallVersionSwitchPickParams__TypeHint">InstallVersionSwitchPick</span></code>.</p>

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


<div id="InstallVersionSwitchQueueParams__TypeHint" class="tip-content">
<p>Install.VersionSwitch.Queue (client request) <a href="#/?id=installversionswitchqueue-client-request">(Go to definition)</a></p>

<p>
<p>Prepare to queue a version switch. The client will
receive an <code class="typename"><span class="type">InstallVersionSwitchPick</span></code>.</p>

</p>

<table class="field-table">
<tr>
<td><code>caveId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>


<div id="InstallVersionSwitchQueueResult__TypeHint" class="tip-content">
<p>InstallVersionSwitchQueue  <a href="#/?id=installversionswitchqueue-">(Go to definition)</a></p>

</div>

### InstallVersionSwitchPick (client caller)


<p>
<p>Let the user pick which version to switch to.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>cave</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Cave__TypeHint">Cave</span></code></td>
<td></td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Upload__TypeHint">Upload</span></code></td>
<td></td>
</tr>
<tr>
<td><code>builds</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Build__TypeHint">Build</span>[]</code></td>
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


<div id="InstallVersionSwitchPickParams__TypeHint" class="tip-content">
<p>InstallVersionSwitchPick (client caller) <a href="#/?id=installversionswitchpick-client-caller">(Go to definition)</a></p>

<p>
<p>Let the user pick which version to switch to.</p>

</p>

<table class="field-table">
<tr>
<td><code>cave</code></td>
<td><code class="typename"><span class="type">Cave</span></code></td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type">Upload</span></code></td>
</tr>
<tr>
<td><code>builds</code></td>
<td><code class="typename"><span class="type">Build</span>[]</code></td>
</tr>
</table>

</div>


<div id="InstallVersionSwitchPickResult__TypeHint" class="tip-content">
<p>InstallVersionSwitchPick  <a href="#/?id=installversionswitchpick-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>index</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### PickUpload (client caller)


<p>
<p>Asks the user to pick between multiple available uploads</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>uploads</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Upload__TypeHint">Upload</span>[]</code></td>
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


<div id="PickUploadParams__TypeHint" class="tip-content">
<p>PickUpload (client caller) <a href="#/?id=pickupload-client-caller">(Go to definition)</a></p>

<p>
<p>Asks the user to pick between multiple available uploads</p>

</p>

<table class="field-table">
<tr>
<td><code>uploads</code></td>
<td><code class="typename"><span class="type">Upload</span>[]</code></td>
</tr>
</table>

</div>


<div id="PickUploadResult__TypeHint" class="tip-content">
<p>PickUpload  <a href="#/?id=pickupload-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>index</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### Progress (notification)


<p>
<p>Sent periodically during <code class="typename"><span class="type" data-tip-selector="#InstallPerformParams__TypeHint">Install.Perform</span></code> to inform on the current state of an install</p>

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


<div id="ProgressNotification__TypeHint" class="tip-content">
<p>Progress (notification) <a href="#/?id=progress-notification">(Go to definition)</a></p>

<p>
<p>Sent periodically during <code class="typename"><span class="type">Install.Perform</span></code> to inform on the current state of an install</p>

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

### TaskReason (enum)



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


<div id="TaskReason__TypeHint" class="tip-content">
<p>TaskReason (enum) <a href="#/?id=taskreason-enum">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>"install"</code></td>
</tr>
<tr>
<td><code>"uninstall"</code></td>
</tr>
</table>

</div>

### TaskType (enum)



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


<div id="TaskType__TypeHint" class="tip-content">
<p>TaskType (enum) <a href="#/?id=tasktype-enum">(Go to definition)</a></p>


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

### TaskStarted (notification)


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
<td><code class="typename"><span class="type" data-tip-selector="#TaskReason__TypeHint">TaskReason</span></code></td>
<td><p>Why this task was started</p>
</td>
</tr>
<tr>
<td><code>type</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#TaskType__TypeHint">TaskType</span></code></td>
<td><p>Is this task a download? An install?</p>
</td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td><p>The game this task is dealing with</p>
</td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Upload__TypeHint">Upload</span></code></td>
<td><p>The upload this task is dealing with</p>
</td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Build__TypeHint">Build</span></code></td>
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


<div id="TaskStartedNotification__TypeHint" class="tip-content">
<p>TaskStarted (notification) <a href="#/?id=taskstarted-notification">(Go to definition)</a></p>

<p>
<p>Each operation is made up of one or more tasks. This notification
is sent during <code class="typename"><span class="type builtin-type">OperationStartParams</span></code> whenever a specific task starts.</p>

</p>

<table class="field-table">
<tr>
<td><code>reason</code></td>
<td><code class="typename"><span class="type">TaskReason</span></code></td>
</tr>
<tr>
<td><code>type</code></td>
<td><code class="typename"><span class="type">TaskType</span></code></td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type">Game</span></code></td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type">Upload</span></code></td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type">Build</span></code></td>
</tr>
<tr>
<td><code>totalSize</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### TaskSucceeded (notification)


<p>
<p>Sent during <code class="typename"><span class="type builtin-type">OperationStartParams</span></code> whenever a task succeeds for an operation.</p>

</p>

<p>
<span class="header">Payload</span> 
</p>


<table class="field-table">
<tr>
<td><code>type</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#TaskType__TypeHint">TaskType</span></code></td>
<td></td>
</tr>
<tr>
<td><code>installResult</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#InstallResult__TypeHint">InstallResult</span></code></td>
<td><p>If the task installed something, then this contains
info about the game, upload, build that were installed</p>
</td>
</tr>
</table>


<div id="TaskSucceededNotification__TypeHint" class="tip-content">
<p>TaskSucceeded (notification) <a href="#/?id=tasksucceeded-notification">(Go to definition)</a></p>

<p>
<p>Sent during <code class="typename"><span class="type builtin-type">OperationStartParams</span></code> whenever a task succeeds for an operation.</p>

</p>

<table class="field-table">
<tr>
<td><code>type</code></td>
<td><code class="typename"><span class="type">TaskType</span></code></td>
</tr>
<tr>
<td><code>installResult</code></td>
<td><code class="typename"><span class="type">InstallResult</span></code></td>
</tr>
</table>

</div>

### InstallResult (struct)


<p>
<p>What was installed by a subtask of <code class="typename"><span class="type builtin-type">OperationStartParams</span></code>.</p>

<p>See <code class="typename"><span class="type" data-tip-selector="#TaskSucceededNotification__TypeHint">TaskSucceeded</span></code>.</p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td><p>The game we installed</p>
</td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Upload__TypeHint">Upload</span></code></td>
<td><p>The upload we installed</p>
</td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Build__TypeHint">Build</span></code></td>
<td><p><span class="tag">Optional</span> The build we installed</p>
</td>
</tr>
</table>


<div id="InstallResult__TypeHint" class="tip-content">
<p>InstallResult (struct) <a href="#/?id=installresult-struct">(Go to definition)</a></p>

<p>
<p>What was installed by a subtask of <code class="typename"><span class="type builtin-type">OperationStartParams</span></code>.</p>

<p>See <code class="typename"><span class="type">TaskSucceeded</span></code>.</p>

</p>

<table class="field-table">
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type">Game</span></code></td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type">Upload</span></code></td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type">Build</span></code></td>
</tr>
</table>

</div>

### Install.Locations.List (client request)



<p>
<span class="header">Parameters</span> <em>none</em>
</p>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>installLocations</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#InstallLocationSummary__TypeHint">InstallLocationSummary</span>[]</code></td>
<td></td>
</tr>
</table>


<div id="InstallLocationsListParams__TypeHint" class="tip-content">
<p>Install.Locations.List (client request) <a href="#/?id=installlocationslist-client-request">(Go to definition)</a></p>

</div>


<div id="InstallLocationsListResult__TypeHint" class="tip-content">
<p>InstallLocationsList  <a href="#/?id=installlocationslist-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>installLocations</code></td>
<td><code class="typename"><span class="type">InstallLocationSummary</span>[]</code></td>
</tr>
</table>

</div>

### Install.Locations.Add (client request)



<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p><span class="tag">Optional</span> identifier of the new install location.
if not specified, will be generated.</p>
</td>
</tr>
<tr>
<td><code>path</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>path of the new install location</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>installLocation</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#InstallLocationSummary__TypeHint">InstallLocationSummary</span></code></td>
<td></td>
</tr>
</table>


<div id="InstallLocationsAddParams__TypeHint" class="tip-content">
<p>Install.Locations.Add (client request) <a href="#/?id=installlocationsadd-client-request">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>path</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>


<div id="InstallLocationsAddResult__TypeHint" class="tip-content">
<p>InstallLocationsAdd  <a href="#/?id=installlocationsadd-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>installLocation</code></td>
<td><code class="typename"><span class="type">InstallLocationSummary</span></code></td>
</tr>
</table>

</div>

### Install.Locations.Remove (client request)



<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>identifier of the install location to remove</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> <em>none</em>
</p>


<div id="InstallLocationsRemoveParams__TypeHint" class="tip-content">
<p>Install.Locations.Remove (client request) <a href="#/?id=installlocationsremove-client-request">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>


<div id="InstallLocationsRemoveResult__TypeHint" class="tip-content">
<p>InstallLocationsRemove  <a href="#/?id=installlocationsremove-">(Go to definition)</a></p>

</div>

### Install.Locations.GetByID (client request)



<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>identifier of the install location to remove</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>installLocation</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#InstallLocationSummary__TypeHint">InstallLocationSummary</span></code></td>
<td></td>
</tr>
</table>


<div id="InstallLocationsGetByIDParams__TypeHint" class="tip-content">
<p>Install.Locations.GetByID (client request) <a href="#/?id=installlocationsgetbyid-client-request">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>


<div id="InstallLocationsGetByIDResult__TypeHint" class="tip-content">
<p>InstallLocationsGetByID  <a href="#/?id=installlocationsgetbyid-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>installLocation</code></td>
<td><code class="typename"><span class="type">InstallLocationSummary</span></code></td>
</tr>
</table>

</div>

### Install.Locations.Scan (client request)



<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>legacyMarketPath</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p><span class="tag">Optional</span> path to a legacy marketDB</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>numFoundItems</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>numImportedItems</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
</table>


<div id="InstallLocationsScanParams__TypeHint" class="tip-content">
<p>Install.Locations.Scan (client request) <a href="#/?id=installlocationsscan-client-request">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>legacyMarketPath</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>


<div id="InstallLocationsScanResult__TypeHint" class="tip-content">
<p>InstallLocationsScan  <a href="#/?id=installlocationsscan-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>numFoundItems</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>numImportedItems</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### Install.Locations.Scan.Yield (notification)


<p>
<p>Sent during <code class="typename"><span class="type" data-tip-selector="#InstallLocationsScanParams__TypeHint">Install.Locations.Scan</span></code> whenever
a game is found.</p>

</p>

<p>
<span class="header">Payload</span> 
</p>


<table class="field-table">
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td></td>
</tr>
</table>


<div id="InstallLocationsScanYieldNotification__TypeHint" class="tip-content">
<p>Install.Locations.Scan.Yield (notification) <a href="#/?id=installlocationsscanyield-notification">(Go to definition)</a></p>

<p>
<p>Sent during <code class="typename"><span class="type">Install.Locations.Scan</span></code> whenever
a game is found.</p>

</p>

<table class="field-table">
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type">Game</span></code></td>
</tr>
</table>

</div>

### Install.Locations.Scan.ConfirmImport (client caller)


<p>
<p>Sent at the end of <code class="typename"><span class="type" data-tip-selector="#InstallLocationsScanParams__TypeHint">Install.Locations.Scan</span></code></p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>numItems</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>number of items that will be imported</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>confirm</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td></td>
</tr>
</table>


<div id="InstallLocationsScanConfirmImportParams__TypeHint" class="tip-content">
<p>Install.Locations.Scan.ConfirmImport (client caller) <a href="#/?id=installlocationsscanconfirmimport-client-caller">(Go to definition)</a></p>

<p>
<p>Sent at the end of <code class="typename"><span class="type">Install.Locations.Scan</span></code></p>

</p>

<table class="field-table">
<tr>
<td><code>numItems</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>


<div id="InstallLocationsScanConfirmImportResult__TypeHint" class="tip-content">
<p>InstallLocationsScanConfirmImport  <a href="#/?id=installlocationsscanconfirmimport-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>confirm</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>


## Downloads Category

### Downloads.Queue (client request)


<p>
<p>Queue a download that will be performed later by
<code class="typename"><span class="type" data-tip-selector="#DownloadsDriveParams__TypeHint">Downloads.Drive</span></code>.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>item</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#InstallQueueResult__TypeHint">InstallQueue</span></code></td>
<td></td>
</tr>
</table>



<p>
<span class="header">Result</span> <em>none</em>
</p>


<div id="DownloadsQueueParams__TypeHint" class="tip-content">
<p>Downloads.Queue (client request) <a href="#/?id=downloadsqueue-client-request">(Go to definition)</a></p>

<p>
<p>Queue a download that will be performed later by
<code class="typename"><span class="type">Downloads.Drive</span></code>.</p>

</p>

<table class="field-table">
<tr>
<td><code>item</code></td>
<td><code class="typename"><span class="type">InstallQueue</span></code></td>
</tr>
</table>

</div>


<div id="DownloadsQueueResult__TypeHint" class="tip-content">
<p>DownloadsQueue  <a href="#/?id=downloadsqueue-">(Go to definition)</a></p>

</div>

### Downloads.Prioritize (client request)


<p>
<p>Put a download on top of the queue.</p>

</p>

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


<div id="DownloadsPrioritizeParams__TypeHint" class="tip-content">
<p>Downloads.Prioritize (client request) <a href="#/?id=downloadsprioritize-client-request">(Go to definition)</a></p>

<p>
<p>Put a download on top of the queue.</p>

</p>

<table class="field-table">
<tr>
<td><code>downloadId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>


<div id="DownloadsPrioritizeResult__TypeHint" class="tip-content">
<p>DownloadsPrioritize  <a href="#/?id=downloadsprioritize-">(Go to definition)</a></p>

</div>

### Downloads.List (client request)


<p>
<p>List all known downloads.</p>

</p>

<p>
<span class="header">Parameters</span> <em>none</em>
</p>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>downloads</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Download__TypeHint">Download</span>[]</code></td>
<td></td>
</tr>
</table>


<div id="DownloadsListParams__TypeHint" class="tip-content">
<p>Downloads.List (client request) <a href="#/?id=downloadslist-client-request">(Go to definition)</a></p>

<p>
<p>List all known downloads.</p>

</p>
</div>


<div id="DownloadsListResult__TypeHint" class="tip-content">
<p>DownloadsList  <a href="#/?id=downloadslist-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>downloads</code></td>
<td><code class="typename"><span class="type">Download</span>[]</code></td>
</tr>
</table>

</div>

### Downloads.ClearFinished (client request)


<p>
<p>Removes all finished downloads from the queue.</p>

</p>

<p>
<span class="header">Parameters</span> <em>none</em>
</p>



<p>
<span class="header">Result</span> <em>none</em>
</p>


<div id="DownloadsClearFinishedParams__TypeHint" class="tip-content">
<p>Downloads.ClearFinished (client request) <a href="#/?id=downloadsclearfinished-client-request">(Go to definition)</a></p>

<p>
<p>Removes all finished downloads from the queue.</p>

</p>
</div>


<div id="DownloadsClearFinishedResult__TypeHint" class="tip-content">
<p>DownloadsClearFinished  <a href="#/?id=downloadsclearfinished-">(Go to definition)</a></p>

</div>

### Downloads.Drive (client request)


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


<div id="DownloadsDriveParams__TypeHint" class="tip-content">
<p>Downloads.Drive (client request) <a href="#/?id=downloadsdrive-client-request">(Go to definition)</a></p>

<p>
<p>Drive downloads, which is: perform them one at a time,
until they&rsquo;re all finished.</p>

</p>
</div>


<div id="DownloadsDriveResult__TypeHint" class="tip-content">
<p>DownloadsDrive  <a href="#/?id=downloadsdrive-">(Go to definition)</a></p>

</div>

### Downloads.Drive.Cancel (client request)


<p>
<p>Stop driving downloads gracefully.</p>

</p>

<p>
<span class="header">Parameters</span> <em>none</em>
</p>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>didCancel</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td></td>
</tr>
</table>


<div id="DownloadsDriveCancelParams__TypeHint" class="tip-content">
<p>Downloads.Drive.Cancel (client request) <a href="#/?id=downloadsdrivecancel-client-request">(Go to definition)</a></p>

<p>
<p>Stop driving downloads gracefully.</p>

</p>
</div>


<div id="DownloadsDriveCancelResult__TypeHint" class="tip-content">
<p>DownloadsDriveCancel  <a href="#/?id=downloadsdrivecancel-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>didCancel</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>

### Downloads.Retry (client request)


<p>
<p>Retries a download that has errored</p>

</p>

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


<div id="DownloadsRetryParams__TypeHint" class="tip-content">
<p>Downloads.Retry (client request) <a href="#/?id=downloadsretry-client-request">(Go to definition)</a></p>

<p>
<p>Retries a download that has errored</p>

</p>

<table class="field-table">
<tr>
<td><code>downloadId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>


<div id="DownloadsRetryResult__TypeHint" class="tip-content">
<p>DownloadsRetry  <a href="#/?id=downloadsretry-">(Go to definition)</a></p>

</div>

### Downloads.Discard (client request)


<p>
<p>Attempts to discard a download</p>

</p>

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


<div id="DownloadsDiscardParams__TypeHint" class="tip-content">
<p>Downloads.Discard (client request) <a href="#/?id=downloadsdiscard-client-request">(Go to definition)</a></p>

<p>
<p>Attempts to discard a download</p>

</p>

<table class="field-table">
<tr>
<td><code>downloadId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>


<div id="DownloadsDiscardResult__TypeHint" class="tip-content">
<p>DownloadsDiscard  <a href="#/?id=downloadsdiscard-">(Go to definition)</a></p>

</div>


## Update Category

### CheckUpdate (client request)


<p>
<p>Looks for game updates.</p>

<p>If a list of cave identifiers is passed, will only look for
updates for these caves <em>and will ignore snooze</em>.</p>

<p>Otherwise, will look for updates for all games, respecting snooze.</p>

<p>Updates found are regularly sent via <code class="typename"><span class="type" data-tip-selector="#GameUpdateAvailableNotification__TypeHint">GameUpdateAvailable</span></code>, and
then all at once in the result.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>caveIds</code></td>
<td><code class="typename"><span class="type builtin-type">string</span>[]</code></td>
<td><p><span class="tag">Optional</span> If specified, will only look for updates to these caves</p>
</td>
</tr>
<tr>
<td><code>verbose</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span> If specified, will log information even when we have no warnings/errors</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>updates</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#GameUpdate__TypeHint">GameUpdate</span>[]</code></td>
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


<div id="CheckUpdateParams__TypeHint" class="tip-content">
<p>CheckUpdate (client request) <a href="#/?id=checkupdate-client-request">(Go to definition)</a></p>

<p>
<p>Looks for game updates.</p>

<p>If a list of cave identifiers is passed, will only look for
updates for these caves <em>and will ignore snooze</em>.</p>

<p>Otherwise, will look for updates for all games, respecting snooze.</p>

<p>Updates found are regularly sent via <code class="typename"><span class="type">GameUpdateAvailable</span></code>, and
then all at once in the result.</p>

</p>

<table class="field-table">
<tr>
<td><code>caveIds</code></td>
<td><code class="typename"><span class="type builtin-type">string</span>[]</code></td>
</tr>
<tr>
<td><code>verbose</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>


<div id="CheckUpdateResult__TypeHint" class="tip-content">
<p>CheckUpdate  <a href="#/?id=checkupdate-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>updates</code></td>
<td><code class="typename"><span class="type">GameUpdate</span>[]</code></td>
</tr>
<tr>
<td><code>warnings</code></td>
<td><code class="typename"><span class="type builtin-type">string</span>[]</code></td>
</tr>
</table>

</div>

### GameUpdateAvailable (notification)


<p>
<p>Sent during <code class="typename"><span class="type" data-tip-selector="#CheckUpdateParams__TypeHint">CheckUpdate</span></code>, every time butler
finds an update for a game. Can be safely ignored if displaying
updates as they are found is not a requirement for the client.</p>

</p>

<p>
<span class="header">Payload</span> 
</p>


<table class="field-table">
<tr>
<td><code>update</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#GameUpdate__TypeHint">GameUpdate</span></code></td>
<td></td>
</tr>
</table>


<div id="GameUpdateAvailableNotification__TypeHint" class="tip-content">
<p>GameUpdateAvailable (notification) <a href="#/?id=gameupdateavailable-notification">(Go to definition)</a></p>

<p>
<p>Sent during <code class="typename"><span class="type">CheckUpdate</span></code>, every time butler
finds an update for a game. Can be safely ignored if displaying
updates as they are found is not a requirement for the client.</p>

</p>

<table class="field-table">
<tr>
<td><code>update</code></td>
<td><code class="typename"><span class="type">GameUpdate</span></code></td>
</tr>
</table>

</div>

### GameUpdate (struct)


<p>
<p>Describes an available update for a particular game install.</p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>caveId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Cave we found an update for</p>
</td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td><p>Game we found an update for</p>
</td>
</tr>
<tr>
<td><code>direct</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>True if this is a direct update, ie. we&rsquo;re on
a channel that still exists, and there&rsquo;s a new build
False if it&rsquo;s an indirect update, for example a new
upload that appeared after we installed, but we&rsquo;re
not sure if it&rsquo;s an upgrade or other additional content</p>
</td>
</tr>
<tr>
<td><code>choices</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#GameUpdateChoice__TypeHint">GameUpdateChoice</span>[]</code></td>
<td><p>Available choice of updates</p>
</td>
</tr>
</table>


<div id="GameUpdate__TypeHint" class="tip-content">
<p>GameUpdate (struct) <a href="#/?id=gameupdate-struct">(Go to definition)</a></p>

<p>
<p>Describes an available update for a particular game install.</p>

</p>

<table class="field-table">
<tr>
<td><code>caveId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type">Game</span></code></td>
</tr>
<tr>
<td><code>direct</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>choices</code></td>
<td><code class="typename"><span class="type">GameUpdateChoice</span>[]</code></td>
</tr>
</table>

</div>

### SnoozeCave (client request)


<p>
<p>Snoozing a cave means we ignore all new uploads (that would
be potential updates) between the cave&rsquo;s last install operation
and now.</p>

<p>This can be undone by calling <code class="typename"><span class="type" data-tip-selector="#CheckUpdateParams__TypeHint">CheckUpdate</span></code> with this specific
cave identifier.</p>

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
<span class="header">Result</span> <em>none</em>
</p>


<div id="SnoozeCaveParams__TypeHint" class="tip-content">
<p>SnoozeCave (client request) <a href="#/?id=snoozecave-client-request">(Go to definition)</a></p>

<p>
<p>Snoozing a cave means we ignore all new uploads (that would
be potential updates) between the cave&rsquo;s last install operation
and now.</p>

<p>This can be undone by calling <code class="typename"><span class="type">CheckUpdate</span></code> with this specific
cave identifier.</p>

</p>

<table class="field-table">
<tr>
<td><code>caveId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>


<div id="SnoozeCaveResult__TypeHint" class="tip-content">
<p>SnoozeCave  <a href="#/?id=snoozecave-">(Go to definition)</a></p>

</div>


## update Category

### GameUpdateChoice (struct)


<p>
<p>One possible upload/build choice to upgrade a cave</p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Upload__TypeHint">Upload</span></code></td>
<td><p>Upload to be installed</p>
</td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Build__TypeHint">Build</span></code></td>
<td><p>Build to be installed (may be nil)</p>
</td>
</tr>
<tr>
<td><code>confidence</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>How confident we are that this is the right upgrade</p>
</td>
</tr>
</table>


<div id="GameUpdateChoice__TypeHint" class="tip-content">
<p>GameUpdateChoice (struct) <a href="#/?id=gameupdatechoice-struct">(Go to definition)</a></p>

<p>
<p>One possible upload/build choice to upgrade a cave</p>

</p>

<table class="field-table">
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type">Upload</span></code></td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type">Build</span></code></td>
</tr>
<tr>
<td><code>confidence</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>


## Launch Category

### Launch (client request)


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
<td><p><span class="tag">Optional</span> Enable sandbox (regardless of manifest opt-in)</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> <em>none</em>
</p>


<div id="LaunchParams__TypeHint" class="tip-content">
<p>Launch (client request) <a href="#/?id=launch-client-request">(Go to definition)</a></p>

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


<div id="LaunchResult__TypeHint" class="tip-content">
<p>Launch  <a href="#/?id=launch-">(Go to definition)</a></p>

</div>

### LaunchRunning (notification)


<p>
<p>Sent during <code class="typename"><span class="type" data-tip-selector="#LaunchParams__TypeHint">Launch</span></code>, when the game is configured, prerequisites are installed
sandbox is set up (if enabled), and the game is actually running.</p>

</p>

<p>
<span class="header">Payload</span> <em>none</em>
</p>


<div id="LaunchRunningNotification__TypeHint" class="tip-content">
<p>LaunchRunning (notification) <a href="#/?id=launchrunning-notification">(Go to definition)</a></p>

<p>
<p>Sent during <code class="typename"><span class="type">Launch</span></code>, when the game is configured, prerequisites are installed
sandbox is set up (if enabled), and the game is actually running.</p>

</p>
</div>

### LaunchExited (notification)


<p>
<p>Sent during <code class="typename"><span class="type" data-tip-selector="#LaunchParams__TypeHint">Launch</span></code>, when the game has actually exited.</p>

</p>

<p>
<span class="header">Payload</span> <em>none</em>
</p>


<div id="LaunchExitedNotification__TypeHint" class="tip-content">
<p>LaunchExited (notification) <a href="#/?id=launchexited-notification">(Go to definition)</a></p>

<p>
<p>Sent during <code class="typename"><span class="type">Launch</span></code>, when the game has actually exited.</p>

</p>
</div>

### AcceptLicense (client caller)


<p>
<p>Sent during <code class="typename"><span class="type" data-tip-selector="#LaunchParams__TypeHint">Launch</span></code> if the game/application comes with a service license
agreement.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>text</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>The full text of the license agreement, in its default
language, which is usually English.</p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>accept</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>true if the user accepts the terms of the license, false otherwise.
Note that false will cancel the launch.</p>
</td>
</tr>
</table>


<div id="AcceptLicenseParams__TypeHint" class="tip-content">
<p>AcceptLicense (client caller) <a href="#/?id=acceptlicense-client-caller">(Go to definition)</a></p>

<p>
<p>Sent during <code class="typename"><span class="type">Launch</span></code> if the game/application comes with a service license
agreement.</p>

</p>

<table class="field-table">
<tr>
<td><code>text</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>


<div id="AcceptLicenseResult__TypeHint" class="tip-content">
<p>AcceptLicense  <a href="#/?id=acceptlicense-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>accept</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>

### PickManifestAction (client caller)


<p>
<p>Sent during <code class="typename"><span class="type" data-tip-selector="#LaunchParams__TypeHint">Launch</span></code>, ask the user to pick a manifest action to launch.</p>

<p>See <a href="https://itch.io/docs/itch/integrating/manifest.html">itch app manifests</a>.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>actions</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Action__TypeHint">Action</span>[]</code></td>
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


<div id="PickManifestActionParams__TypeHint" class="tip-content">
<p>PickManifestAction (client caller) <a href="#/?id=pickmanifestaction-client-caller">(Go to definition)</a></p>

<p>
<p>Sent during <code class="typename"><span class="type">Launch</span></code>, ask the user to pick a manifest action to launch.</p>

<p>See <a href="https://itch.io/docs/itch/integrating/manifest.html">itch app manifests</a>.</p>

</p>

<table class="field-table">
<tr>
<td><code>actions</code></td>
<td><code class="typename"><span class="type">Action</span>[]</code></td>
</tr>
</table>

</div>


<div id="PickManifestActionResult__TypeHint" class="tip-content">
<p>PickManifestAction  <a href="#/?id=pickmanifestaction-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>index</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### ShellLaunch (client caller)


<p>
<p>Ask the client to perform a shell launch, ie. open an item
with the operating system&rsquo;s default handler (File explorer).</p>

<p>Sent during <code class="typename"><span class="type" data-tip-selector="#LaunchParams__TypeHint">Launch</span></code>.</p>

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


<div id="ShellLaunchParams__TypeHint" class="tip-content">
<p>ShellLaunch (client caller) <a href="#/?id=shelllaunch-client-caller">(Go to definition)</a></p>

<p>
<p>Ask the client to perform a shell launch, ie. open an item
with the operating system&rsquo;s default handler (File explorer).</p>

<p>Sent during <code class="typename"><span class="type">Launch</span></code>.</p>

</p>

<table class="field-table">
<tr>
<td><code>itemPath</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>


<div id="ShellLaunchResult__TypeHint" class="tip-content">
<p>ShellLaunch  <a href="#/?id=shelllaunch-">(Go to definition)</a></p>

</div>

### HTMLLaunch (client caller)


<p>
<p>Ask the client to perform an HTML launch, ie. open an HTML5
game, ideally in an embedded browser.</p>

<p>Sent during <code class="typename"><span class="type" data-tip-selector="#LaunchParams__TypeHint">Launch</span></code>.</p>

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
<td><code class="typename"><span class="type builtin-type">{ [key: string]: string }</span></code></td>
<td><p>Environment variables, to pass as <code>global.Itch.env</code></p>
</td>
</tr>
</table>



<p>
<span class="header">Result</span> <em>none</em>
</p>


<div id="HTMLLaunchParams__TypeHint" class="tip-content">
<p>HTMLLaunch (client caller) <a href="#/?id=htmllaunch-client-caller">(Go to definition)</a></p>

<p>
<p>Ask the client to perform an HTML launch, ie. open an HTML5
game, ideally in an embedded browser.</p>

<p>Sent during <code class="typename"><span class="type">Launch</span></code>.</p>

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
<td><code class="typename"><span class="type builtin-type">{ [key: string]: string }</span></code></td>
</tr>
</table>

</div>


<div id="HTMLLaunchResult__TypeHint" class="tip-content">
<p>HTMLLaunch  <a href="#/?id=htmllaunch-">(Go to definition)</a></p>

</div>

### URLLaunch (client caller)


<p>
<p>Ask the client to perform an URL launch, ie. open an address
with the system browser or appropriate.</p>

<p>Sent during <code class="typename"><span class="type" data-tip-selector="#LaunchParams__TypeHint">Launch</span></code>.</p>

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


<div id="URLLaunchParams__TypeHint" class="tip-content">
<p>URLLaunch (client caller) <a href="#/?id=urllaunch-client-caller">(Go to definition)</a></p>

<p>
<p>Ask the client to perform an URL launch, ie. open an address
with the system browser or appropriate.</p>

<p>Sent during <code class="typename"><span class="type">Launch</span></code>.</p>

</p>

<table class="field-table">
<tr>
<td><code>url</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>


<div id="URLLaunchResult__TypeHint" class="tip-content">
<p>URLLaunch  <a href="#/?id=urllaunch-">(Go to definition)</a></p>

</div>

### AllowSandboxSetup (client caller)


<p>
<p>Ask the user to allow sandbox setup. Will be followed by
a UAC prompt (on Windows) or a pkexec dialog (on Linux) if
the user allows.</p>

<p>Sent during <code class="typename"><span class="type" data-tip-selector="#LaunchParams__TypeHint">Launch</span></code>.</p>

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


<div id="AllowSandboxSetupParams__TypeHint" class="tip-content">
<p>AllowSandboxSetup (client caller) <a href="#/?id=allowsandboxsetup-client-caller">(Go to definition)</a></p>

<p>
<p>Ask the user to allow sandbox setup. Will be followed by
a UAC prompt (on Windows) or a pkexec dialog (on Linux) if
the user allows.</p>

<p>Sent during <code class="typename"><span class="type">Launch</span></code>.</p>

</p>
</div>


<div id="AllowSandboxSetupResult__TypeHint" class="tip-content">
<p>AllowSandboxSetup  <a href="#/?id=allowsandboxsetup-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>allow</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>

### PrereqsStarted (notification)


<p>
<p>Sent during <code class="typename"><span class="type" data-tip-selector="#LaunchParams__TypeHint">Launch</span></code>, when some prerequisites are about to be installed.</p>

<p>This is a good time to start showing a UI element with the state of prereq
tasks.</p>

<p>Updates are regularly provided via <code class="typename"><span class="type" data-tip-selector="#PrereqsTaskStateNotification__TypeHint">PrereqsTaskState</span></code>.</p>

</p>

<p>
<span class="header">Payload</span> 
</p>


<table class="field-table">
<tr>
<td><code>tasks</code></td>
<td><code class="typename"><span class="type builtin-type">{ [key: string]: PrereqTask }</span></code></td>
<td><p>A list of prereqs that need to be tended to</p>
</td>
</tr>
</table>


<div id="PrereqsStartedNotification__TypeHint" class="tip-content">
<p>PrereqsStarted (notification) <a href="#/?id=prereqsstarted-notification">(Go to definition)</a></p>

<p>
<p>Sent during <code class="typename"><span class="type">Launch</span></code>, when some prerequisites are about to be installed.</p>

<p>This is a good time to start showing a UI element with the state of prereq
tasks.</p>

<p>Updates are regularly provided via <code class="typename"><span class="type">PrereqsTaskState</span></code>.</p>

</p>

<table class="field-table">
<tr>
<td><code>tasks</code></td>
<td><code class="typename"><span class="type builtin-type">{ [key: string]: PrereqTask }</span></code></td>
</tr>
</table>

</div>

### PrereqTask (struct)


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


<div id="PrereqTask__TypeHint" class="tip-content">
<p>PrereqTask (struct) <a href="#/?id=prereqtask-struct">(Go to definition)</a></p>

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

### PrereqsTaskState (notification)


<p>
<p>Current status of a prerequisite task</p>

<p>Sent during <code class="typename"><span class="type" data-tip-selector="#LaunchParams__TypeHint">Launch</span></code>, after <code class="typename"><span class="type" data-tip-selector="#PrereqsStartedNotification__TypeHint">PrereqsStarted</span></code>, repeatedly
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
<td><code class="typename"><span class="type" data-tip-selector="#PrereqStatus__TypeHint">PrereqStatus</span></code></td>
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


<div id="PrereqsTaskStateNotification__TypeHint" class="tip-content">
<p>PrereqsTaskState (notification) <a href="#/?id=prereqstaskstate-notification">(Go to definition)</a></p>

<p>
<p>Current status of a prerequisite task</p>

<p>Sent during <code class="typename"><span class="type">Launch</span></code>, after <code class="typename"><span class="type">PrereqsStarted</span></code>, repeatedly
until all prereq tasks are done.</p>

</p>

<table class="field-table">
<tr>
<td><code>name</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>status</code></td>
<td><code class="typename"><span class="type">PrereqStatus</span></code></td>
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

### PrereqStatus (enum)



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


<div id="PrereqStatus__TypeHint" class="tip-content">
<p>PrereqStatus (enum) <a href="#/?id=prereqstatus-enum">(Go to definition)</a></p>


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

### PrereqsEnded (notification)


<p>
<p>Sent during <code class="typename"><span class="type" data-tip-selector="#LaunchParams__TypeHint">Launch</span></code>, when all prereqs have finished installing (successfully or not)</p>

<p>After this is received, it&rsquo;s safe to close any UI element showing prereq task state.</p>

</p>

<p>
<span class="header">Payload</span> <em>none</em>
</p>


<div id="PrereqsEndedNotification__TypeHint" class="tip-content">
<p>PrereqsEnded (notification) <a href="#/?id=prereqsended-notification">(Go to definition)</a></p>

<p>
<p>Sent during <code class="typename"><span class="type">Launch</span></code>, when all prereqs have finished installing (successfully or not)</p>

<p>After this is received, it&rsquo;s safe to close any UI element showing prereq task state.</p>

</p>
</div>

### PrereqsFailed (client caller)


<p>
<p>Sent during <code class="typename"><span class="type" data-tip-selector="#LaunchParams__TypeHint">Launch</span></code>, when one or more prerequisites have failed to install.
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


<div id="PrereqsFailedParams__TypeHint" class="tip-content">
<p>PrereqsFailed (client caller) <a href="#/?id=prereqsfailed-client-caller">(Go to definition)</a></p>

<p>
<p>Sent during <code class="typename"><span class="type">Launch</span></code>, when one or more prerequisites have failed to install.
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


<div id="PrereqsFailedResult__TypeHint" class="tip-content">
<p>PrereqsFailed  <a href="#/?id=prereqsfailed-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>continue</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>


## Clean Downloads Category

### CleanDownloads.Search (client request)


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
<td><code class="typename"><span class="type" data-tip-selector="#CleanDownloadsEntry__TypeHint">CleanDownloadsEntry</span>[]</code></td>
<td><p>Entries we found that could use some cleaning (with path and size information)</p>
</td>
</tr>
</table>


<div id="CleanDownloadsSearchParams__TypeHint" class="tip-content">
<p>CleanDownloads.Search (client request) <a href="#/?id=cleandownloadssearch-client-request">(Go to definition)</a></p>

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


<div id="CleanDownloadsSearchResult__TypeHint" class="tip-content">
<p>CleanDownloadsSearch  <a href="#/?id=cleandownloadssearch-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>entries</code></td>
<td><code class="typename"><span class="type">CleanDownloadsEntry</span>[]</code></td>
</tr>
</table>

</div>

### CleanDownloadsEntry (struct)



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


<div id="CleanDownloadsEntry__TypeHint" class="tip-content">
<p>CleanDownloadsEntry (struct) <a href="#/?id=cleandownloadsentry-struct">(Go to definition)</a></p>


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

### CleanDownloads.Apply (client request)


<p>
<p>Remove the specified entries from disk, freeing up disk space.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>entries</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#CleanDownloadsEntry__TypeHint">CleanDownloadsEntry</span>[]</code></td>
<td></td>
</tr>
</table>



<p>
<span class="header">Result</span> <em>none</em>
</p>


<div id="CleanDownloadsApplyParams__TypeHint" class="tip-content">
<p>CleanDownloads.Apply (client request) <a href="#/?id=cleandownloadsapply-client-request">(Go to definition)</a></p>

<p>
<p>Remove the specified entries from disk, freeing up disk space.</p>

</p>

<table class="field-table">
<tr>
<td><code>entries</code></td>
<td><code class="typename"><span class="type">CleanDownloadsEntry</span>[]</code></td>
</tr>
</table>

</div>


<div id="CleanDownloadsApplyResult__TypeHint" class="tip-content">
<p>CleanDownloadsApply  <a href="#/?id=cleandownloadsapply-">(Go to definition)</a></p>

</div>


## System Category

### System.StatFS (client request)


<p>
<p>Get information on a filesystem.</p>

</p>

<p>
<span class="header">Parameters</span> 
</p>


<table class="field-table">
<tr>
<td><code>path</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
</table>



<p>
<span class="header">Result</span> 
</p>


<table class="field-table">
<tr>
<td><code>freeSize</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>totalSize</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
</table>


<div id="SystemStatFSParams__TypeHint" class="tip-content">
<p>System.StatFS (client request) <a href="#/?id=systemstatfs-client-request">(Go to definition)</a></p>

<p>
<p>Get information on a filesystem.</p>

</p>

<table class="field-table">
<tr>
<td><code>path</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>


<div id="SystemStatFSResult__TypeHint" class="tip-content">
<p>SystemStatFS  <a href="#/?id=systemstatfs-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>freeSize</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>totalSize</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>


## Test Category

### Test.DoubleTwice (client request)


<p>
<p>Test request: asks butler to double a number twice.
First by calling <code class="typename"><span class="type" data-tip-selector="#TestDoubleParams__TypeHint">Test.Double</span></code>, then by
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


<div id="TestDoubleTwiceParams__TypeHint" class="tip-content">
<p>Test.DoubleTwice (client request) <a href="#/?id=testdoubletwice-client-request">(Go to definition)</a></p>

<p>
<p>Test request: asks butler to double a number twice.
First by calling <code class="typename"><span class="type">Test.Double</span></code>, then by
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


<div id="TestDoubleTwiceResult__TypeHint" class="tip-content">
<p>TestDoubleTwice  <a href="#/?id=testdoubletwice-">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>number</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### Test.Double (client caller)


<p>
<p>Test request: return a number, doubled. Implement that to
use <code class="typename"><span class="type" data-tip-selector="#TestDoubleTwiceParams__TypeHint">Test.DoubleTwice</span></code> in your testing.</p>

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


<div id="TestDoubleParams__TypeHint" class="tip-content">
<p>Test.Double (client caller) <a href="#/?id=testdouble-client-caller">(Go to definition)</a></p>

<p>
<p>Test request: return a number, doubled. Implement that to
use <code class="typename"><span class="type">Test.DoubleTwice</span></code> in your testing.</p>

</p>

<table class="field-table">
<tr>
<td><code>number</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>


<div id="TestDoubleResult__TypeHint" class="tip-content">
<p>TestDouble  <a href="#/?id=testdouble-">(Go to definition)</a></p>

<p>
<p>Result for Test.Double</p>

</p>

<table class="field-table">
<tr>
<td><code>number</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>


## Miscellaneous Category

### Profile (struct)


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
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
<td><p>Timestamp the user last connected at (to the client)</p>
</td>
</tr>
<tr>
<td><code>user</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#User__TypeHint">User</span></code></td>
<td><p>User information</p>
</td>
</tr>
</table>


<div id="Profile__TypeHint" class="tip-content">
<p>Profile (struct) <a href="#/?id=profile-struct">(Go to definition)</a></p>

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
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
</tr>
<tr>
<td><code>user</code></td>
<td><code class="typename"><span class="type">User</span></code></td>
</tr>
</table>

</div>

### GameRecord (struct)



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Game ID</p>
</td>
</tr>
<tr>
<td><code>title</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Game title</p>
</td>
</tr>
<tr>
<td><code>cover</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Game cover</p>
</td>
</tr>
<tr>
<td><code>owned</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>True if owned</p>
</td>
</tr>
<tr>
<td><code>installedAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
<td><p>Non-nil if installed (has caves)</p>
</td>
</tr>
</table>


<div id="GameRecord__TypeHint" class="tip-content">
<p>GameRecord (struct) <a href="#/?id=gamerecord-struct">(Go to definition)</a></p>


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
<td><code>cover</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>owned</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>installedAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
</tr>
</table>

</div>

### GameRecordsSource (enum)



<p>
<span class="header">Values</span> 
</p>


<table class="field-table">
<tr>
<td><code>"owned"</code></td>
<td><p>Games for which the profile has a download key</p>
</td>
</tr>
<tr>
<td><code>"installed"</code></td>
<td><p>Games for which a cave exists (regardless of the profile)</p>
</td>
</tr>
<tr>
<td><code>"profile"</code></td>
<td><p>Games authored by profile, or for whom profile is an admin of</p>
</td>
</tr>
<tr>
<td><code>"collection"</code></td>
<td><p>Games from a collection</p>
</td>
</tr>
</table>


<div id="GameRecordsSource__TypeHint" class="tip-content">
<p>GameRecordsSource (enum) <a href="#/?id=gamerecordssource-enum">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>"owned"</code></td>
</tr>
<tr>
<td><code>"installed"</code></td>
</tr>
<tr>
<td><code>"profile"</code></td>
</tr>
<tr>
<td><code>"collection"</code></td>
</tr>
</table>

</div>

### GameRecordsFilters (struct)



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>classification</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#GameClassification__TypeHint">GameClassification</span></code></td>
<td><p><span class="tag">Optional</span></p>
</td>
</tr>
<tr>
<td><code>installed</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span></p>
</td>
</tr>
<tr>
<td><code>owned</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p><span class="tag">Optional</span></p>
</td>
</tr>
</table>


<div id="GameRecordsFilters__TypeHint" class="tip-content">
<p>GameRecordsFilters (struct) <a href="#/?id=gamerecordsfilters-struct">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>classification</code></td>
<td><code class="typename"><span class="type">GameClassification</span></code></td>
</tr>
<tr>
<td><code>installed</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>owned</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>

### FetchDownloadKeysFilter (struct)



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>gameId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p><span class="tag">Optional</span> Return only download keys for given game</p>
</td>
</tr>
</table>


<div id="FetchDownloadKeysFilter__TypeHint" class="tip-content">
<p>FetchDownloadKeysFilter (struct) <a href="#/?id=fetchdownloadkeysfilter-struct">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>gameId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### CollectionGamesFilters (struct)



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>installed</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td></td>
</tr>
<tr>
<td><code>classification</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#GameClassification__TypeHint">GameClassification</span></code></td>
<td></td>
</tr>
</table>


<div id="CollectionGamesFilters__TypeHint" class="tip-content">
<p>CollectionGamesFilters (struct) <a href="#/?id=collectiongamesfilters-struct">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>installed</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>classification</code></td>
<td><code class="typename"><span class="type">GameClassification</span></code></td>
</tr>
</table>

</div>

### ProfileGameFilters (struct)



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>visibility</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
<tr>
<td><code>paidStatus</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
</table>


<div id="ProfileGameFilters__TypeHint" class="tip-content">
<p>ProfileGameFilters (struct) <a href="#/?id=profilegamefilters-struct">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>visibility</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>paidStatus</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### ProfileGame (struct)



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td></td>
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


<div id="ProfileGame__TypeHint" class="tip-content">
<p>ProfileGame (struct) <a href="#/?id=profilegame-struct">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type">Game</span></code></td>
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

### ProfileOwnedKeysFilters (struct)



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>installed</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td></td>
</tr>
<tr>
<td><code>classification</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#GameClassification__TypeHint">GameClassification</span></code></td>
<td></td>
</tr>
</table>


<div id="ProfileOwnedKeysFilters__TypeHint" class="tip-content">
<p>ProfileOwnedKeysFilters (struct) <a href="#/?id=profileownedkeysfilters-struct">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>installed</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>classification</code></td>
<td><code class="typename"><span class="type">GameClassification</span></code></td>
</tr>
</table>

</div>

### DownloadKeySummary (struct)



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
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
<td><p>Date this key was created at (often coincides with purchase time)</p>
</td>
</tr>
</table>


<div id="DownloadKeySummary__TypeHint" class="tip-content">
<p>DownloadKeySummary (struct) <a href="#/?id=downloadkeysummary-struct">(Go to definition)</a></p>


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
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
</tr>
</table>

</div>

### CaveSummary (struct)



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
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
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


<div id="CaveSummary__TypeHint" class="tip-content">
<p>CaveSummary (struct) <a href="#/?id=cavesummary-struct">(Go to definition)</a></p>


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
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
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

### Cave (struct)


<p>
<p>A Cave corresponds to an &ldquo;installed item&rdquo; for a game.</p>

<p>It maps one-to-one with an upload. There might be 0, 1, or several
caves for a given game. Multiple caves for a single game is a rare-ish
case (single-page bundles, bonus content) but one that should be handled.</p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Unique identifier of this cave (UUID)</p>
</td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td><p>Game that&rsquo;s installed in this cave</p>
</td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Upload__TypeHint">Upload</span></code></td>
<td><p>Upload that&rsquo;s installed in this cave</p>
</td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Build__TypeHint">Build</span></code></td>
<td><p><span class="tag">Optional</span> Build that&rsquo;s installed in this cave, if the upload is wharf-powered</p>
</td>
</tr>
<tr>
<td><code>stats</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#CaveStats__TypeHint">CaveStats</span></code></td>
<td><p>Stats about cave usage and first install</p>
</td>
</tr>
<tr>
<td><code>installInfo</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#CaveInstallInfo__TypeHint">CaveInstallInfo</span></code></td>
<td><p>Information about where the cave is installed, how much space it takes up etc.</p>
</td>
</tr>
</table>


<div id="Cave__TypeHint" class="tip-content">
<p>Cave (struct) <a href="#/?id=cave-struct">(Go to definition)</a></p>

<p>
<p>A Cave corresponds to an &ldquo;installed item&rdquo; for a game.</p>

<p>It maps one-to-one with an upload. There might be 0, 1, or several
caves for a given game. Multiple caves for a single game is a rare-ish
case (single-page bundles, bonus content) but one that should be handled.</p>

</p>

<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type">Game</span></code></td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type">Upload</span></code></td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type">Build</span></code></td>
</tr>
<tr>
<td><code>stats</code></td>
<td><code class="typename"><span class="type">CaveStats</span></code></td>
</tr>
<tr>
<td><code>installInfo</code></td>
<td><code class="typename"><span class="type">CaveInstallInfo</span></code></td>
</tr>
</table>

</div>

### CaveStats (struct)


<p>
<p>CaveStats contains stats about cave usage and first install</p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>installedAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
<td><p>Time the cave was first installed</p>
</td>
</tr>
<tr>
<td><code>lastTouchedAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
<td></td>
</tr>
<tr>
<td><code>secondsRun</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
</table>


<div id="CaveStats__TypeHint" class="tip-content">
<p>CaveStats (struct) <a href="#/?id=cavestats-struct">(Go to definition)</a></p>

<p>
<p>CaveStats contains stats about cave usage and first install</p>

</p>

<table class="field-table">
<tr>
<td><code>installedAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
</tr>
<tr>
<td><code>lastTouchedAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
</tr>
<tr>
<td><code>secondsRun</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### CaveInstallInfo (struct)


<p>
<p>CaveInstallInfo contains information about where the cave is installed, how
much space it takes up, etc.</p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>installedSize</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Size the cave takes up - or at least, size it took up when we finished
installing it. Does not include files generated by the game in the install folder.</p>
</td>
</tr>
<tr>
<td><code>installLocation</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Name of the install location for this cave. This may change if the cave
is moved.</p>
</td>
</tr>
<tr>
<td><code>installFolder</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Absolute path to the install folder</p>
</td>
</tr>
<tr>
<td><code>pinned</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>If true, this cave is ignored while checking for updates</p>
</td>
</tr>
</table>


<div id="CaveInstallInfo__TypeHint" class="tip-content">
<p>CaveInstallInfo (struct) <a href="#/?id=caveinstallinfo-struct">(Go to definition)</a></p>

<p>
<p>CaveInstallInfo contains information about where the cave is installed, how
much space it takes up, etc.</p>

</p>

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
<tr>
<td><code>pinned</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>

### InstallLocationSummary (struct)



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Unique identifier for this install location</p>
</td>
</tr>
<tr>
<td><code>path</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Absolute path on disk for this install location</p>
</td>
</tr>
<tr>
<td><code>sizeInfo</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#InstallLocationSizeInfo__TypeHint">InstallLocationSizeInfo</span></code></td>
<td><p>Information about the size used and available at this install location</p>
</td>
</tr>
</table>


<div id="InstallLocationSummary__TypeHint" class="tip-content">
<p>InstallLocationSummary (struct) <a href="#/?id=installlocationsummary-struct">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>path</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>sizeInfo</code></td>
<td><code class="typename"><span class="type">InstallLocationSizeInfo</span></code></td>
</tr>
</table>

</div>

### InstallLocationSizeInfo (struct)



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>installedSize</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Number of bytes used by caves installed in this location</p>
</td>
</tr>
<tr>
<td><code>freeSize</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Free space at this location (depends on the partition/disk on which
it is), or a negative value if we can&rsquo;t find it</p>
</td>
</tr>
<tr>
<td><code>totalSize</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Total space of this location (depends on the partition/disk on which
it is), or a negative value if we can&rsquo;t find it</p>
</td>
</tr>
</table>


<div id="InstallLocationSizeInfo__TypeHint" class="tip-content">
<p>InstallLocationSizeInfo (struct) <a href="#/?id=installlocationsizeinfo-struct">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>installedSize</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>freeSize</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>totalSize</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### CavesFilters (struct)



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>classification</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#GameClassification__TypeHint">GameClassification</span></code></td>
<td><p><span class="tag">Optional</span></p>
</td>
</tr>
<tr>
<td><code>gameId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p><span class="tag">Optional</span></p>
</td>
</tr>
<tr>
<td><code>installLocationId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p><span class="tag">Optional</span></p>
</td>
</tr>
</table>


<div id="CavesFilters__TypeHint" class="tip-content">
<p>CavesFilters (struct) <a href="#/?id=cavesfilters-struct">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>classification</code></td>
<td><code class="typename"><span class="type">GameClassification</span></code></td>
</tr>
<tr>
<td><code>gameId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>installLocationId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### InstallPlanInfo (struct)



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Upload__TypeHint">Upload</span></code></td>
<td></td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Build__TypeHint">Build</span></code></td>
<td></td>
</tr>
<tr>
<td><code>type</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
<tr>
<td><code>diskUsage</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#DiskUsageInfo__TypeHint">DiskUsageInfo</span></code></td>
<td></td>
</tr>
<tr>
<td><code>error</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
<tr>
<td><code>errorMessage</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
<tr>
<td><code>errorCode</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
</table>


<div id="InstallPlanInfo__TypeHint" class="tip-content">
<p>InstallPlanInfo (struct) <a href="#/?id=installplaninfo-struct">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type">Upload</span></code></td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type">Build</span></code></td>
</tr>
<tr>
<td><code>type</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>diskUsage</code></td>
<td><code class="typename"><span class="type">DiskUsageInfo</span></code></td>
</tr>
<tr>
<td><code>error</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>errorMessage</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>errorCode</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### DiskUsageInfo (struct)



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>finalDiskUsage</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>neededFreeSpace</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>accuracy</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
</table>


<div id="DiskUsageInfo__TypeHint" class="tip-content">
<p>DiskUsageInfo (struct) <a href="#/?id=diskusageinfo-struct">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>finalDiskUsage</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>neededFreeSpace</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>accuracy</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### InstallEvent (struct)



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>type</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#InstallEventType__TypeHint">InstallEventType</span></code></td>
<td></td>
</tr>
<tr>
<td><code>timestamp</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
<td></td>
</tr>
<tr>
<td><code>heal</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#HealInstallEvent__TypeHint">HealInstallEvent</span></code></td>
<td></td>
</tr>
<tr>
<td><code>install</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#InstallInstallEvent__TypeHint">InstallInstallEvent</span></code></td>
<td></td>
</tr>
<tr>
<td><code>upgrade</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#UpgradeInstallEvent__TypeHint">UpgradeInstallEvent</span></code></td>
<td></td>
</tr>
<tr>
<td><code>ghostBusting</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#GhostBustingInstallEvent__TypeHint">GhostBustingInstallEvent</span></code></td>
<td></td>
</tr>
<tr>
<td><code>patching</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#PatchingInstallEvent__TypeHint">PatchingInstallEvent</span></code></td>
<td></td>
</tr>
<tr>
<td><code>problem</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#ProblemInstallEvent__TypeHint">ProblemInstallEvent</span></code></td>
<td></td>
</tr>
<tr>
<td><code>fallback</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#FallbackInstallEvent__TypeHint">FallbackInstallEvent</span></code></td>
<td></td>
</tr>
</table>


<div id="InstallEvent__TypeHint" class="tip-content">
<p>InstallEvent (struct) <a href="#/?id=installevent-struct">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>type</code></td>
<td><code class="typename"><span class="type">InstallEventType</span></code></td>
</tr>
<tr>
<td><code>timestamp</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
</tr>
<tr>
<td><code>heal</code></td>
<td><code class="typename"><span class="type">HealInstallEvent</span></code></td>
</tr>
<tr>
<td><code>install</code></td>
<td><code class="typename"><span class="type">InstallInstallEvent</span></code></td>
</tr>
<tr>
<td><code>upgrade</code></td>
<td><code class="typename"><span class="type">UpgradeInstallEvent</span></code></td>
</tr>
<tr>
<td><code>ghostBusting</code></td>
<td><code class="typename"><span class="type">GhostBustingInstallEvent</span></code></td>
</tr>
<tr>
<td><code>patching</code></td>
<td><code class="typename"><span class="type">PatchingInstallEvent</span></code></td>
</tr>
<tr>
<td><code>problem</code></td>
<td><code class="typename"><span class="type">ProblemInstallEvent</span></code></td>
</tr>
<tr>
<td><code>fallback</code></td>
<td><code class="typename"><span class="type">FallbackInstallEvent</span></code></td>
</tr>
</table>

</div>

### InstallEventType (enum)



<p>
<span class="header">Values</span> 
</p>


<table class="field-table">
<tr>
<td><code>"resume"</code></td>
<td><p>Started for the first time or resumed after a pause
or butler exit or whatever</p>
</td>
</tr>
<tr>
<td><code>"stop"</code></td>
<td><p>Stopped explicitly (pausing downloads), can&rsquo;t rely
on this being present because BRÜTAL PÖWER LÖSS will
not announce itself 🔥</p>
</td>
</tr>
<tr>
<td><code>"install"</code></td>
<td><p>Regular install from archive or naked file</p>
</td>
</tr>
<tr>
<td><code>"heal"</code></td>
<td><p>Reverting to previous version or re-installing
wharf-powered upload</p>
</td>
</tr>
<tr>
<td><code>"upgrade"</code></td>
<td><p>Applying one or more wharf patches</p>
</td>
</tr>
<tr>
<td><code>"patching"</code></td>
<td><p>Applying a single wharf patch</p>
</td>
</tr>
<tr>
<td><code>"ghostBusting"</code></td>
<td><p>Cleaning up ghost files</p>
</td>
</tr>
<tr>
<td><code>"problem"</code></td>
<td><p>Any kind of step failing</p>
</td>
</tr>
<tr>
<td><code>"fallback"</code></td>
<td><p>Any operation we do as a result of another one failing,
but in a case where we&rsquo;re still expecting a favorable
outcome eventually.</p>
</td>
</tr>
</table>


<div id="InstallEventType__TypeHint" class="tip-content">
<p>InstallEventType (enum) <a href="#/?id=installeventtype-enum">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>"resume"</code></td>
</tr>
<tr>
<td><code>"stop"</code></td>
</tr>
<tr>
<td><code>"install"</code></td>
</tr>
<tr>
<td><code>"heal"</code></td>
</tr>
<tr>
<td><code>"upgrade"</code></td>
</tr>
<tr>
<td><code>"patching"</code></td>
</tr>
<tr>
<td><code>"ghostBusting"</code></td>
</tr>
<tr>
<td><code>"problem"</code></td>
</tr>
<tr>
<td><code>"fallback"</code></td>
</tr>
</table>

</div>

### InstallInstallEvent (struct)



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>manager</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
</table>


<div id="InstallInstallEvent__TypeHint" class="tip-content">
<p>InstallInstallEvent (struct) <a href="#/?id=installinstallevent-struct">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>manager</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### HealInstallEvent (struct)



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>totalCorrupted</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>appliedCaseFixes</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td></td>
</tr>
</table>


<div id="HealInstallEvent__TypeHint" class="tip-content">
<p>HealInstallEvent (struct) <a href="#/?id=healinstallevent-struct">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>totalCorrupted</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>appliedCaseFixes</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>

### UpgradeInstallEvent (struct)



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>numPatches</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
</table>


<div id="UpgradeInstallEvent__TypeHint" class="tip-content">
<p>UpgradeInstallEvent (struct) <a href="#/?id=upgradeinstallevent-struct">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>numPatches</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### ProblemInstallEvent (struct)



<p>
<span class="header">Fields</span> 
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
<td><p>Longer error</p>
</td>
</tr>
</table>


<div id="ProblemInstallEvent__TypeHint" class="tip-content">
<p>ProblemInstallEvent (struct) <a href="#/?id=probleminstallevent-struct">(Go to definition)</a></p>


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

### FallbackInstallEvent (struct)



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>attempted</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Name of the operation we were trying to do</p>
</td>
</tr>
<tr>
<td><code>problem</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#ProblemInstallEvent__TypeHint">ProblemInstallEvent</span></code></td>
<td><p>Problem encountered while trying &ldquo;attempted&rdquo;</p>
</td>
</tr>
<tr>
<td><code>nowTrying</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Name of the operation we&rsquo;re falling back to</p>
</td>
</tr>
</table>


<div id="FallbackInstallEvent__TypeHint" class="tip-content">
<p>FallbackInstallEvent (struct) <a href="#/?id=fallbackinstallevent-struct">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>attempted</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>problem</code></td>
<td><code class="typename"><span class="type">ProblemInstallEvent</span></code></td>
</tr>
<tr>
<td><code>nowTrying</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### PatchingInstallEvent (struct)



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>buildID</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Build we patched to</p>
</td>
</tr>
<tr>
<td><code>subtype</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>&ldquo;default&rdquo; or &ldquo;optimized&rdquo; (for the +bsdiff variant)</p>
</td>
</tr>
</table>


<div id="PatchingInstallEvent__TypeHint" class="tip-content">
<p>PatchingInstallEvent (struct) <a href="#/?id=patchinginstallevent-struct">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>buildID</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>subtype</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### GhostBustingInstallEvent (struct)



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>operation</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Operation that requested the ghost busting (install, upgrade, heal)</p>
</td>
</tr>
<tr>
<td><code>found</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Number of ghost files found</p>
</td>
</tr>
<tr>
<td><code>removed</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>Number of ghost files removed</p>
</td>
</tr>
</table>


<div id="GhostBustingInstallEvent__TypeHint" class="tip-content">
<p>GhostBustingInstallEvent (struct) <a href="#/?id=ghostbustinginstallevent-struct">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>operation</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>found</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>removed</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### GameCredentials (struct)


<p>
<p>GameCredentials contains all the credentials required to make API requests
including the download key if any.</p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
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


<div id="GameCredentials__TypeHint" class="tip-content">
<p>GameCredentials (struct) <a href="#/?id=gamecredentials-struct">(Go to definition)</a></p>

<p>
<p>GameCredentials contains all the credentials required to make API requests
including the download key if any.</p>

</p>

<table class="field-table">
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

### Downloads.Drive.Progress (notification)



<p>
<span class="header">Payload</span> 
</p>


<table class="field-table">
<tr>
<td><code>download</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Download__TypeHint">Download</span></code></td>
<td></td>
</tr>
<tr>
<td><code>progress</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#DownloadProgress__TypeHint">DownloadProgress</span></code></td>
<td></td>
</tr>
<tr>
<td><code>speedHistory</code></td>
<td><code class="typename"><span class="type builtin-type">number</span>[]</code></td>
<td><p>BPS values for the last minute</p>
</td>
</tr>
</table>


<div id="DownloadsDriveProgressNotification__TypeHint" class="tip-content">
<p>Downloads.Drive.Progress (notification) <a href="#/?id=downloadsdriveprogress-notification">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>download</code></td>
<td><code class="typename"><span class="type">Download</span></code></td>
</tr>
<tr>
<td><code>progress</code></td>
<td><code class="typename"><span class="type">DownloadProgress</span></code></td>
</tr>
<tr>
<td><code>speedHistory</code></td>
<td><code class="typename"><span class="type builtin-type">number</span>[]</code></td>
</tr>
</table>

</div>

### Downloads.Drive.Started (notification)



<p>
<span class="header">Payload</span> 
</p>


<table class="field-table">
<tr>
<td><code>download</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Download__TypeHint">Download</span></code></td>
<td></td>
</tr>
</table>


<div id="DownloadsDriveStartedNotification__TypeHint" class="tip-content">
<p>Downloads.Drive.Started (notification) <a href="#/?id=downloadsdrivestarted-notification">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>download</code></td>
<td><code class="typename"><span class="type">Download</span></code></td>
</tr>
</table>

</div>

### Downloads.Drive.Errored (notification)



<p>
<span class="header">Payload</span> 
</p>


<table class="field-table">
<tr>
<td><code>download</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Download__TypeHint">Download</span></code></td>
<td><p>The download that errored. It contains all the error
information: a short message, a full stack trace,
and a butlerd error code.</p>
</td>
</tr>
</table>


<div id="DownloadsDriveErroredNotification__TypeHint" class="tip-content">
<p>Downloads.Drive.Errored (notification) <a href="#/?id=downloadsdriveerrored-notification">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>download</code></td>
<td><code class="typename"><span class="type">Download</span></code></td>
</tr>
</table>

</div>

### Downloads.Drive.Finished (notification)



<p>
<span class="header">Payload</span> 
</p>


<table class="field-table">
<tr>
<td><code>download</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Download__TypeHint">Download</span></code></td>
<td></td>
</tr>
</table>


<div id="DownloadsDriveFinishedNotification__TypeHint" class="tip-content">
<p>Downloads.Drive.Finished (notification) <a href="#/?id=downloadsdrivefinished-notification">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>download</code></td>
<td><code class="typename"><span class="type">Download</span></code></td>
</tr>
</table>

</div>

### Downloads.Drive.Discarded (notification)



<p>
<span class="header">Payload</span> 
</p>


<table class="field-table">
<tr>
<td><code>download</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Download__TypeHint">Download</span></code></td>
<td></td>
</tr>
</table>


<div id="DownloadsDriveDiscardedNotification__TypeHint" class="tip-content">
<p>Downloads.Drive.Discarded (notification) <a href="#/?id=downloadsdrivediscarded-notification">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>download</code></td>
<td><code class="typename"><span class="type">Download</span></code></td>
</tr>
</table>

</div>

### Downloads.Drive.NetworkStatus (notification)


<p>
<p>Sent during <code class="typename"><span class="type" data-tip-selector="#DownloadsDriveParams__TypeHint">Downloads.Drive</span></code> to inform on network
status changes.</p>

</p>

<p>
<span class="header">Payload</span> 
</p>


<table class="field-table">
<tr>
<td><code>status</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#NetworkStatus__TypeHint">NetworkStatus</span></code></td>
<td><p>The current network status</p>
</td>
</tr>
</table>


<div id="DownloadsDriveNetworkStatusNotification__TypeHint" class="tip-content">
<p>Downloads.Drive.NetworkStatus (notification) <a href="#/?id=downloadsdrivenetworkstatus-notification">(Go to definition)</a></p>

<p>
<p>Sent during <code class="typename"><span class="type">Downloads.Drive</span></code> to inform on network
status changes.</p>

</p>

<table class="field-table">
<tr>
<td><code>status</code></td>
<td><code class="typename"><span class="type">NetworkStatus</span></code></td>
</tr>
</table>

</div>

### NetworkStatus (enum)



<p>
<span class="header">Values</span> 
</p>


<table class="field-table">
<tr>
<td><code>"online"</code></td>
<td></td>
</tr>
<tr>
<td><code>"offline"</code></td>
<td></td>
</tr>
</table>


<div id="NetworkStatus__TypeHint" class="tip-content">
<p>NetworkStatus (enum) <a href="#/?id=networkstatus-enum">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>"online"</code></td>
</tr>
<tr>
<td><code>"offline"</code></td>
</tr>
</table>

</div>

### DownloadReason (enum)



<p>
<span class="header">Values</span> 
</p>


<table class="field-table">
<tr>
<td><code>"install"</code></td>
<td></td>
</tr>
<tr>
<td><code>"reinstall"</code></td>
<td></td>
</tr>
<tr>
<td><code>"update"</code></td>
<td></td>
</tr>
<tr>
<td><code>"version-switch"</code></td>
<td></td>
</tr>
</table>


<div id="DownloadReason__TypeHint" class="tip-content">
<p>DownloadReason (enum) <a href="#/?id=downloadreason-enum">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>"install"</code></td>
</tr>
<tr>
<td><code>"reinstall"</code></td>
</tr>
<tr>
<td><code>"update"</code></td>
</tr>
<tr>
<td><code>"version-switch"</code></td>
</tr>
</table>

</div>

### Download (struct)


<p>
<p>Represents a download queued, which will be
performed whenever <code class="typename"><span class="type" data-tip-selector="#DownloadsDriveParams__TypeHint">Downloads.Drive</span></code> is called.</p>

</p>

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
<td><code>error</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
<tr>
<td><code>errorMessage</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
<tr>
<td><code>errorCode</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>reason</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#DownloadReason__TypeHint">DownloadReason</span></code></td>
<td></td>
</tr>
<tr>
<td><code>position</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>caveId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td></td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Upload__TypeHint">Upload</span></code></td>
<td></td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Build__TypeHint">Build</span></code></td>
<td></td>
</tr>
<tr>
<td><code>startedAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
<td></td>
</tr>
<tr>
<td><code>finishedAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
<td></td>
</tr>
<tr>
<td><code>stagingFolder</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
</table>


<div id="Download__TypeHint" class="tip-content">
<p>Download (struct) <a href="#/?id=download-struct">(Go to definition)</a></p>

<p>
<p>Represents a download queued, which will be
performed whenever <code class="typename"><span class="type">Downloads.Drive</span></code> is called.</p>

</p>

<table class="field-table">
<tr>
<td><code>id</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>error</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>errorMessage</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>errorCode</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>reason</code></td>
<td><code class="typename"><span class="type">DownloadReason</span></code></td>
</tr>
<tr>
<td><code>position</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>caveId</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type">Game</span></code></td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type">Upload</span></code></td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type">Build</span></code></td>
</tr>
<tr>
<td><code>startedAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
</tr>
<tr>
<td><code>finishedAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
</tr>
<tr>
<td><code>stagingFolder</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### DownloadProgress (struct)



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


<div id="DownloadProgress__TypeHint" class="tip-content">
<p>DownloadProgress (struct) <a href="#/?id=downloadprogress-struct">(Go to definition)</a></p>


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

### Log (notification)


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
<td><code class="typename"><span class="type" data-tip-selector="#LogLevel__TypeHint">LogLevel</span></code></td>
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


<div id="LogNotification__TypeHint" class="tip-content">
<p>Log (notification) <a href="#/?id=log-notification">(Go to definition)</a></p>

<p>
<p>Sent any time butler needs to send a log message. The client should
relay them in their own stdout / stderr, and collect them so they
can be part of an issue report if something goes wrong.</p>

</p>

<table class="field-table">
<tr>
<td><code>level</code></td>
<td><code class="typename"><span class="type">LogLevel</span></code></td>
</tr>
<tr>
<td><code>message</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### LogLevel (enum)



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


<div id="LogLevel__TypeHint" class="tip-content">
<p>LogLevel (enum) <a href="#/?id=loglevel-enum">(Go to definition)</a></p>


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

### Code (enum)


<p>
<p>butlerd JSON-RPC 2.0 error codes</p>

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
<tr>
<td><code>404</code></td>
<td><p>We tried to launch something, but the install folder just wasn&rsquo;t there</p>
</td>
</tr>
<tr>
<td><code>2001</code></td>
<td><p>We tried to install something, but could not find compatible uploads</p>
</td>
</tr>
<tr>
<td><code>3000</code></td>
<td><p>This title is packaged in a way that is not supported.</p>
</td>
</tr>
<tr>
<td><code>3001</code></td>
<td><p>This title is hosted on an incompatible third-party website</p>
</td>
</tr>
<tr>
<td><code>5000</code></td>
<td><p>Nothing that can be launched was found</p>
</td>
</tr>
<tr>
<td><code>6000</code></td>
<td><p>Java Runtime Environment is required to launch this title.</p>
</td>
</tr>
<tr>
<td><code>9000</code></td>
<td><p>There is no Internet connection</p>
</td>
</tr>
<tr>
<td><code>12000</code></td>
<td><p>API error</p>
</td>
</tr>
<tr>
<td><code>16000</code></td>
<td><p>The database is busy</p>
</td>
</tr>
<tr>
<td><code>18000</code></td>
<td><p>An install location could not be removed because it has active downloads</p>
</td>
</tr>
</table>


<div id="Code__TypeHint" class="tip-content">
<p>Code (enum) <a href="#/?id=code-enum">(Go to definition)</a></p>

<p>
<p>butlerd JSON-RPC 2.0 error codes</p>

</p>

<table class="field-table">
<tr>
<td><code>499</code></td>
</tr>
<tr>
<td><code>410</code></td>
</tr>
<tr>
<td><code>404</code></td>
</tr>
<tr>
<td><code>2001</code></td>
</tr>
<tr>
<td><code>3000</code></td>
</tr>
<tr>
<td><code>3001</code></td>
</tr>
<tr>
<td><code>5000</code></td>
</tr>
<tr>
<td><code>6000</code></td>
</tr>
<tr>
<td><code>9000</code></td>
</tr>
<tr>
<td><code>12000</code></td>
</tr>
<tr>
<td><code>16000</code></td>
</tr>
<tr>
<td><code>18000</code></td>
</tr>
</table>

</div>

### Cursor 


Type alias for string

<div id="Cursor__TypeHint" class="tip-content">
<p>Cursor  <a href="#/?id=cursor-">(Go to definition)</a></p>
</div>

### LaunchTarget (struct)



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>action</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Action__TypeHint">Action</span></code></td>
<td><p>The manifest action corresponding to this launch target.
For implicit launch targets, a minimal one will be generated.</p>
</td>
</tr>
<tr>
<td><code>host</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Host__TypeHint">Host</span></code></td>
<td><p>Host this launch target was found for</p>
</td>
</tr>
<tr>
<td><code>strategy</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#StrategyResult__TypeHint">Strategy</span></code></td>
<td><p>Detailed launch strategy</p>
</td>
</tr>
</table>


<div id="LaunchTarget__TypeHint" class="tip-content">
<p>LaunchTarget (struct) <a href="#/?id=launchtarget-struct">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>action</code></td>
<td><code class="typename"><span class="type">Action</span></code></td>
</tr>
<tr>
<td><code>host</code></td>
<td><code class="typename"><span class="type">Host</span></code></td>
</tr>
<tr>
<td><code>strategy</code></td>
<td><code class="typename"><span class="type">Strategy</span></code></td>
</tr>
</table>

</div>

### LaunchStrategy (enum)



<p>
<span class="header">Values</span> 
</p>


<table class="field-table">
<tr>
<td><code>""</code></td>
<td></td>
</tr>
<tr>
<td><code>"native"</code></td>
<td></td>
</tr>
<tr>
<td><code>"html"</code></td>
<td></td>
</tr>
<tr>
<td><code>"url"</code></td>
<td></td>
</tr>
<tr>
<td><code>"shell"</code></td>
<td></td>
</tr>
</table>


<div id="LaunchStrategy__TypeHint" class="tip-content">
<p>LaunchStrategy (enum) <a href="#/?id=launchstrategy-enum">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>""</code></td>
</tr>
<tr>
<td><code>"native"</code></td>
</tr>
<tr>
<td><code>"html"</code></td>
</tr>
<tr>
<td><code>"url"</code></td>
</tr>
<tr>
<td><code>"shell"</code></td>
</tr>
</table>

</div>

### Host (struct)



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>runtime</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Runtime__TypeHint">Runtime</span></code></td>
<td><p>os + arch, e.g. windows-i386, linux-amd64</p>
</td>
</tr>
<tr>
<td><code>wrapper</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Wrapper__TypeHint">Wrapper</span></code></td>
<td><p>wrapper tool (wine, etc.) that butler can launch itself</p>
</td>
</tr>
<tr>
<td><code>remoteLaunchName</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td></td>
</tr>
</table>


<div id="Host__TypeHint" class="tip-content">
<p>Host (struct) <a href="#/?id=host-struct">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>runtime</code></td>
<td><code class="typename"><span class="type">Runtime</span></code></td>
</tr>
<tr>
<td><code>wrapper</code></td>
<td><code class="typename"><span class="type">Wrapper</span></code></td>
</tr>
<tr>
<td><code>remoteLaunchName</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### Wrapper (struct)



<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>beforeTarget</code></td>
<td><code class="typename"><span class="type builtin-type">string</span>[]</code></td>
<td><p>wrapper {HERE} game.exe &ndash;launch-editor</p>
</td>
</tr>
<tr>
<td><code>betweenTargetAndArgs</code></td>
<td><code class="typename"><span class="type builtin-type">string</span>[]</code></td>
<td><p>wrapper game.exe {HERE} &ndash;launch-editor</p>
</td>
</tr>
<tr>
<td><code>afterArgs</code></td>
<td><code class="typename"><span class="type builtin-type">string</span>[]</code></td>
<td><p>wrapper game.exe &ndash;launch-editor {HERE}</p>
</td>
</tr>
<tr>
<td><code>wrapperBinary</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>full path to the wrapper, like &ldquo;wine&rdquo;</p>
</td>
</tr>
<tr>
<td><code>env</code></td>
<td><code class="typename"><span class="type builtin-type">{ [key: string]: string }</span></code></td>
<td><p>additional environment variables</p>
</td>
</tr>
<tr>
<td><code>needRelativeTarget</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>When this is true, the wrapper can&rsquo;t function like this:</p>

<p>$ wine /path/to/game.exe</p>

<p>It needs to function like this:</p>

<p>$ cd /path/to
$ wine game.exe</p>

<p>This is at least true for wine, which cannot find required DLLs
otherwise. This might be true for other wrappers, so it&rsquo;s an option here.</p>
</td>
</tr>
</table>


<div id="Wrapper__TypeHint" class="tip-content">
<p>Wrapper (struct) <a href="#/?id=wrapper-struct">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>beforeTarget</code></td>
<td><code class="typename"><span class="type builtin-type">string</span>[]</code></td>
</tr>
<tr>
<td><code>betweenTargetAndArgs</code></td>
<td><code class="typename"><span class="type builtin-type">string</span>[]</code></td>
</tr>
<tr>
<td><code>afterArgs</code></td>
<td><code class="typename"><span class="type builtin-type">string</span>[]</code></td>
</tr>
<tr>
<td><code>wrapperBinary</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
<tr>
<td><code>env</code></td>
<td><code class="typename"><span class="type builtin-type">{ [key: string]: string }</span></code></td>
</tr>
<tr>
<td><code>needRelativeTarget</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>

### Manifest (struct)


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
<td><code class="typename"><span class="type" data-tip-selector="#Actions__TypeHint">Actions</span></code></td>
<td><p>Actions are a list of options to give the user when launching a game.</p>
</td>
</tr>
<tr>
<td><code>prereqs</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Prereq__TypeHint">Prereq</span>[]</code></td>
<td><p>Prereqs describe libraries or frameworks that must be installed
prior to launching a game</p>
</td>
</tr>
</table>


<div id="Manifest__TypeHint" class="tip-content">
<p>Manifest (struct) <a href="#/?id=manifest-struct">(Go to definition)</a></p>

<p>
<p>A Manifest describes prerequisites (dependencies) and actions that
can be taken while launching a game.</p>

</p>

<table class="field-table">
<tr>
<td><code>actions</code></td>
<td><code class="typename"><span class="type">Actions</span></code></td>
</tr>
<tr>
<td><code>prereqs</code></td>
<td><code class="typename"><span class="type">Prereq</span>[]</code></td>
</tr>
</table>

</div>

### Action (struct)


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
<td><code class="typename"><span class="type" data-tip-selector="#Platform__TypeHint">Platform</span></code></td>
<td><p>platform to restrict this action to</p>
</td>
</tr>
<tr>
<td><code>locales</code></td>
<td><code class="typename"><span class="type builtin-type">{ [key: string]: ActionLocale }</span></code></td>
<td><p>localized action name</p>
</td>
</tr>
</table>


<div id="Action__TypeHint" class="tip-content">
<p>Action (struct) <a href="#/?id=action-struct">(Go to definition)</a></p>

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
<td><code class="typename"><span class="type">Platform</span></code></td>
</tr>
<tr>
<td><code>locales</code></td>
<td><code class="typename"><span class="type builtin-type">{ [key: string]: ActionLocale }</span></code></td>
</tr>
</table>

</div>

### Prereq (struct)



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


<div id="Prereq__TypeHint" class="tip-content">
<p>Prereq (struct) <a href="#/?id=prereq-struct">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>name</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### ActionLocale (struct)



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


<div id="ActionLocale__TypeHint" class="tip-content">
<p>ActionLocale (struct) <a href="#/?id=actionlocale-struct">(Go to definition)</a></p>


<table class="field-table">
<tr>
<td><code>name</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
</tr>
</table>

</div>

### Platform (enum)



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


<div id="Platform__TypeHint" class="tip-content">
<p>Platform (enum) <a href="#/?id=platform-enum">(Go to definition)</a></p>


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

### Runtime (struct)


<p>
<p>Runtime describes an os-arch combo in a convenient way</p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>platform</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Platform__TypeHint">Platform</span></code></td>
<td></td>
</tr>
<tr>
<td><code>is64</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td></td>
</tr>
</table>


<div id="Runtime__TypeHint" class="tip-content">
<p>Runtime (struct) <a href="#/?id=runtime-struct">(Go to definition)</a></p>

<p>
<p>Runtime describes an os-arch combo in a convenient way</p>

</p>

<table class="field-table">
<tr>
<td><code>platform</code></td>
<td><code class="typename"><span class="type">Platform</span></code></td>
</tr>
<tr>
<td><code>is64</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
</table>

</div>

### User (struct)


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


<div id="User__TypeHint" class="tip-content">
<p>User (struct) <a href="#/?id=user-struct">(Go to definition)</a></p>

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

### Game (struct)


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
<td><code class="typename"><span class="type" data-tip-selector="#GameType__TypeHint">GameType</span></code></td>
<td><p>Downloadable game, html game, etc.</p>
</td>
</tr>
<tr>
<td><code>classification</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#GameClassification__TypeHint">GameClassification</span></code></td>
<td><p>Classification: game, tool, comic, etc.</p>
</td>
</tr>
<tr>
<td><code>embed</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#GameEmbedData__TypeHint">GameEmbedData</span></code></td>
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
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
<td><p>Date the game was created</p>
</td>
</tr>
<tr>
<td><code>publishedAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
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
<td><p>Are payments accepted?</p>
</td>
</tr>
<tr>
<td><code>hasDemo</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>Does this game have a demo available?</p>
</td>
</tr>
<tr>
<td><code>inPressSystem</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>Is this game part of the itch.io press system?</p>
</td>
</tr>
<tr>
<td><code>platforms</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Platforms__TypeHint">Platforms</span></code></td>
<td><p>Platforms this game is available for</p>
</td>
</tr>
<tr>
<td><code>user</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#User__TypeHint">User</span></code></td>
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
<td><code class="typename"><span class="type" data-tip-selector="#Sale__TypeHint">Sale</span></code></td>
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


<div id="Game__TypeHint" class="tip-content">
<p>Game (struct) <a href="#/?id=game-struct">(Go to definition)</a></p>

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
<td><code class="typename"><span class="type">GameType</span></code></td>
</tr>
<tr>
<td><code>classification</code></td>
<td><code class="typename"><span class="type">GameClassification</span></code></td>
</tr>
<tr>
<td><code>embed</code></td>
<td><code class="typename"><span class="type">GameEmbedData</span></code></td>
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
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
</tr>
<tr>
<td><code>publishedAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
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
<td><code>hasDemo</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>inPressSystem</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>platforms</code></td>
<td><code class="typename"><span class="type">Platforms</span></code></td>
</tr>
<tr>
<td><code>user</code></td>
<td><code class="typename"><span class="type">User</span></code></td>
</tr>
<tr>
<td><code>userId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>sale</code></td>
<td><code class="typename"><span class="type">Sale</span></code></td>
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

### Platforms (struct)


<p>
<p>Platforms describes which OS/architectures a game or upload
is compatible with.</p>

</p>

<p>
<span class="header">Fields</span> 
</p>


<table class="field-table">
<tr>
<td><code>windows</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Architectures__TypeHint">Architectures</span></code></td>
<td></td>
</tr>
<tr>
<td><code>linux</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Architectures__TypeHint">Architectures</span></code></td>
<td></td>
</tr>
<tr>
<td><code>osx</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Architectures__TypeHint">Architectures</span></code></td>
<td></td>
</tr>
</table>


<div id="Platforms__TypeHint" class="tip-content">
<p>Platforms (struct) <a href="#/?id=platforms-struct">(Go to definition)</a></p>

<p>
<p>Platforms describes which OS/architectures a game or upload
is compatible with.</p>

</p>

<table class="field-table">
<tr>
<td><code>windows</code></td>
<td><code class="typename"><span class="type">Architectures</span></code></td>
</tr>
<tr>
<td><code>linux</code></td>
<td><code class="typename"><span class="type">Architectures</span></code></td>
</tr>
<tr>
<td><code>osx</code></td>
<td><code class="typename"><span class="type">Architectures</span></code></td>
</tr>
</table>

</div>

### Architectures (enum)


<p>
<p>Architectures describes a set of processor architectures (mostly 32-bit vs 64-bit)</p>

</p>

<p>
<span class="header">Values</span> 
</p>


<table class="field-table">
<tr>
<td><code>"all"</code></td>
<td><p>ArchitecturesAll represents any processor architecture</p>
</td>
</tr>
<tr>
<td><code>"386"</code></td>
<td><p>Architectures386 represents 32-bit processor architectures</p>
</td>
</tr>
<tr>
<td><code>"amd64"</code></td>
<td><p>ArchitecturesAmd64 represents 64-bit processor architectures</p>
</td>
</tr>
</table>


<div id="Architectures__TypeHint" class="tip-content">
<p>Architectures (enum) <a href="#/?id=architectures-enum">(Go to definition)</a></p>

<p>
<p>Architectures describes a set of processor architectures (mostly 32-bit vs 64-bit)</p>

</p>

<table class="field-table">
<tr>
<td><code>"all"</code></td>
</tr>
<tr>
<td><code>"386"</code></td>
</tr>
<tr>
<td><code>"amd64"</code></td>
</tr>
</table>

</div>

### GameType (enum)


<p>
<p>GameType is the type of an itch.io game page, mostly related to
how it should be presented on web (downloadable or embed)</p>

</p>

<p>
<span class="header">Values</span> 
</p>


<table class="field-table">
<tr>
<td><code>"default"</code></td>
<td><p>GameTypeDefault is downloadable games</p>
</td>
</tr>
<tr>
<td><code>"flash"</code></td>
<td><p>GameTypeFlash is for .swf (legacy)</p>
</td>
</tr>
<tr>
<td><code>"unity"</code></td>
<td><p>GameTypeUnity is for .unity3d (legacy)</p>
</td>
</tr>
<tr>
<td><code>"java"</code></td>
<td><p>GameTypeJava is for .jar (legacy)</p>
</td>
</tr>
<tr>
<td><code>"html"</code></td>
<td><p>GameTypeHTML is for .html (thriving)</p>
</td>
</tr>
</table>


<div id="GameType__TypeHint" class="tip-content">
<p>GameType (enum) <a href="#/?id=gametype-enum">(Go to definition)</a></p>

<p>
<p>GameType is the type of an itch.io game page, mostly related to
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

### GameClassification (enum)


<p>
<p>GameClassification is the creator-picked classification for a page</p>

</p>

<p>
<span class="header">Values</span> 
</p>


<table class="field-table">
<tr>
<td><code>"game"</code></td>
<td><p>GameClassificationGame is something you can play</p>
</td>
</tr>
<tr>
<td><code>"tool"</code></td>
<td><p>GameClassificationTool includes all software pretty much</p>
</td>
</tr>
<tr>
<td><code>"assets"</code></td>
<td><p>GameClassificationAssets includes assets: graphics, sounds, etc.</p>
</td>
</tr>
<tr>
<td><code>"game_mod"</code></td>
<td><p>GameClassificationGameMod are game mods (no link to game, purely creator tagging)</p>
</td>
</tr>
<tr>
<td><code>"physical_game"</code></td>
<td><p>GameClassificationPhysicalGame is for a printable / board / card game</p>
</td>
</tr>
<tr>
<td><code>"soundtrack"</code></td>
<td><p>GameClassificationSoundtrack is a bunch of music files</p>
</td>
</tr>
<tr>
<td><code>"other"</code></td>
<td><p>GameClassificationOther is anything that creators think don&rsquo;t fit in any other category</p>
</td>
</tr>
<tr>
<td><code>"comic"</code></td>
<td><p>GameClassificationComic is a comic book (pdf, jpg, specific comic formats, etc.)</p>
</td>
</tr>
<tr>
<td><code>"book"</code></td>
<td><p>GameClassificationBook is a book (pdf, jpg, specific e-book formats, etc.)</p>
</td>
</tr>
</table>


<div id="GameClassification__TypeHint" class="tip-content">
<p>GameClassification (enum) <a href="#/?id=gameclassification-enum">(Go to definition)</a></p>

<p>
<p>GameClassification is the creator-picked classification for a page</p>

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

### GameEmbedData (struct)


<p>
<p>GameEmbedData contains presentation information for embed games</p>

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


<div id="GameEmbedData__TypeHint" class="tip-content">
<p>GameEmbedData (struct) <a href="#/?id=gameembeddata-struct">(Go to definition)</a></p>

<p>
<p>GameEmbedData contains presentation information for embed games</p>

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

### Sale (struct)


<p>
<p>Sale describes a discount for a game.</p>

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
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
<td><p>Timestamp the sale started at</p>
</td>
</tr>
<tr>
<td><code>endDate</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
<td><p>Timestamp the sale ends at</p>
</td>
</tr>
</table>


<div id="Sale__TypeHint" class="tip-content">
<p>Sale (struct) <a href="#/?id=sale-struct">(Go to definition)</a></p>

<p>
<p>Sale describes a discount for a game.</p>

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
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
</tr>
<tr>
<td><code>endDate</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
</tr>
</table>

</div>

### Upload (struct)


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
<td><code>storage</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#UploadStorage__TypeHint">UploadStorage</span></code></td>
<td><p>Storage (hosted, external, etc.)</p>
</td>
</tr>
<tr>
<td><code>host</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
<td><p>Host (if external storage)</p>
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
<td><code class="typename"><span class="type" data-tip-selector="#Build__TypeHint">Build</span></code></td>
<td><p>Latest build for this upload, if it&rsquo;s a wharf-enabled upload</p>
</td>
</tr>
<tr>
<td><code>buildId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td><p>ID of the latest build for this upload, if it&rsquo;s a wharf-enabled upload</p>
</td>
</tr>
<tr>
<td><code>type</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#UploadType__TypeHint">UploadType</span></code></td>
<td><p>Upload type: default, soundtrack, etc.</p>
</td>
</tr>
<tr>
<td><code>preorder</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>Is this upload a pre-order placeholder?</p>
</td>
</tr>
<tr>
<td><code>demo</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
<td><p>Is this upload a free demo?</p>
</td>
</tr>
<tr>
<td><code>platforms</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Platforms__TypeHint">Platforms</span></code></td>
<td><p>Platforms this upload is compatible with</p>
</td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
<td><p>Date this upload was created at</p>
</td>
</tr>
<tr>
<td><code>updatedAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
<td><p>Date this upload was last updated at (order changed, display name set, etc.)</p>
</td>
</tr>
</table>


<div id="Upload__TypeHint" class="tip-content">
<p>Upload (struct) <a href="#/?id=upload-struct">(Go to definition)</a></p>

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
<td><code>storage</code></td>
<td><code class="typename"><span class="type">UploadStorage</span></code></td>
</tr>
<tr>
<td><code>host</code></td>
<td><code class="typename"><span class="type builtin-type">string</span></code></td>
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
<td><code class="typename"><span class="type">Build</span></code></td>
</tr>
<tr>
<td><code>buildId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>type</code></td>
<td><code class="typename"><span class="type">UploadType</span></code></td>
</tr>
<tr>
<td><code>preorder</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>demo</code></td>
<td><code class="typename"><span class="type builtin-type">boolean</span></code></td>
</tr>
<tr>
<td><code>platforms</code></td>
<td><code class="typename"><span class="type">Platforms</span></code></td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
</tr>
<tr>
<td><code>updatedAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
</tr>
</table>

</div>

### UploadStorage (enum)


<p>
<p>UploadStorage describes where an upload file is stored.</p>

</p>

<p>
<span class="header">Values</span> 
</p>


<table class="field-table">
<tr>
<td><code>"hosted"</code></td>
<td><p>UploadStorageHosted is a classic upload (web) - no versioning</p>
</td>
</tr>
<tr>
<td><code>"build"</code></td>
<td><p>UploadStorageBuild is a wharf upload (butler)</p>
</td>
</tr>
<tr>
<td><code>"external"</code></td>
<td><p>UploadStorageExternal is an external upload - alllllllll bets are off.</p>
</td>
</tr>
</table>


<div id="UploadStorage__TypeHint" class="tip-content">
<p>UploadStorage (enum) <a href="#/?id=uploadstorage-enum">(Go to definition)</a></p>

<p>
<p>UploadStorage describes where an upload file is stored.</p>

</p>

<table class="field-table">
<tr>
<td><code>"hosted"</code></td>
</tr>
<tr>
<td><code>"build"</code></td>
</tr>
<tr>
<td><code>"external"</code></td>
</tr>
</table>

</div>

### UploadType (enum)


<p>
<p>UploadType describes what&rsquo;s in an upload - an executable,
a web game, some music, etc.</p>

</p>

<p>
<span class="header">Values</span> 
</p>


<table class="field-table">
<tr>
<td><code>"default"</code></td>
<td><p>UploadTypeDefault is for executables</p>
</td>
</tr>
<tr>
<td><code>"flash"</code></td>
<td><p>UploadTypeFlash is for .swf files</p>
</td>
</tr>
<tr>
<td><code>"unity"</code></td>
<td><p>UploadTypeUnity is for .unity3d files</p>
</td>
</tr>
<tr>
<td><code>"java"</code></td>
<td><p>UploadTypeJava is for .jar files</p>
</td>
</tr>
<tr>
<td><code>"html"</code></td>
<td><p>UploadTypeHTML is for .html files</p>
</td>
</tr>
<tr>
<td><code>"soundtrack"</code></td>
<td><p>UploadTypeSoundtrack is for archives with .mp3/.ogg/.flac/etc files</p>
</td>
</tr>
<tr>
<td><code>"book"</code></td>
<td><p>UploadTypeBook is for books (epubs, pdfs, etc.)</p>
</td>
</tr>
<tr>
<td><code>"video"</code></td>
<td><p>UploadTypeVideo is for videos</p>
</td>
</tr>
<tr>
<td><code>"documentation"</code></td>
<td><p>UploadTypeDocumentation is for documentation (pdf, maybe uhh doxygen?)</p>
</td>
</tr>
<tr>
<td><code>"mod"</code></td>
<td><p>UploadTypeMod is a bunch of loose files with no clear instructions how to apply them to a game</p>
</td>
</tr>
<tr>
<td><code>"audio_assets"</code></td>
<td><p>UploadTypeAudioAssets is a bunch of .ogg/.wav files</p>
</td>
</tr>
<tr>
<td><code>"graphical_assets"</code></td>
<td><p>UploadTypeGraphicalAssets is a bunch of .png/.svg/.gif files, maybe some .objs thrown in there</p>
</td>
</tr>
<tr>
<td><code>"sourcecode"</code></td>
<td><p>UploadTypeSourcecode is for source code. No further comments.</p>
</td>
</tr>
<tr>
<td><code>"other"</code></td>
<td><p>UploadTypeOther is for literally anything that isn&rsquo;t an existing category,
or for stuff that isn&rsquo;t tagged properly.</p>
</td>
</tr>
</table>


<div id="UploadType__TypeHint" class="tip-content">
<p>UploadType (enum) <a href="#/?id=uploadtype-enum">(Go to definition)</a></p>

<p>
<p>UploadType describes what&rsquo;s in an upload - an executable,
a web game, some music, etc.</p>

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
<tr>
<td><code>"soundtrack"</code></td>
</tr>
<tr>
<td><code>"book"</code></td>
</tr>
<tr>
<td><code>"video"</code></td>
</tr>
<tr>
<td><code>"documentation"</code></td>
</tr>
<tr>
<td><code>"mod"</code></td>
</tr>
<tr>
<td><code>"audio_assets"</code></td>
</tr>
<tr>
<td><code>"graphical_assets"</code></td>
</tr>
<tr>
<td><code>"sourcecode"</code></td>
</tr>
<tr>
<td><code>"other"</code></td>
</tr>
</table>

</div>

### Collection (struct)


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
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
<td><p>Date this collection was created at</p>
</td>
</tr>
<tr>
<td><code>updatedAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
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
<td><code>collectionGames</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#CollectionGame__TypeHint">CollectionGame</span>[]</code></td>
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
<td><code class="typename"><span class="type" data-tip-selector="#User__TypeHint">User</span></code></td>
<td></td>
</tr>
</table>


<div id="Collection__TypeHint" class="tip-content">
<p>Collection (struct) <a href="#/?id=collection-struct">(Go to definition)</a></p>

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
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
</tr>
<tr>
<td><code>updatedAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
</tr>
<tr>
<td><code>gamesCount</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>collectionGames</code></td>
<td><code class="typename"><span class="type">CollectionGame</span>[]</code></td>
</tr>
<tr>
<td><code>userId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>user</code></td>
<td><code class="typename"><span class="type">User</span></code></td>
</tr>
</table>

</div>

### CollectionGame (struct)


<p>
<p>CollectionGame represents a game&rsquo;s membership for a collection.</p>

</p>

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
<td><code class="typename"><span class="type" data-tip-selector="#Collection__TypeHint">Collection</span></code></td>
<td></td>
</tr>
<tr>
<td><code>gameId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td></td>
</tr>
<tr>
<td><code>position</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
<td></td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
<td></td>
</tr>
<tr>
<td><code>updatedAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
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


<div id="CollectionGame__TypeHint" class="tip-content">
<p>CollectionGame (struct) <a href="#/?id=collectiongame-struct">(Go to definition)</a></p>

<p>
<p>CollectionGame represents a game&rsquo;s membership for a collection.</p>

</p>

<table class="field-table">
<tr>
<td><code>collectionId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>collection</code></td>
<td><code class="typename"><span class="type">Collection</span></code></td>
</tr>
<tr>
<td><code>gameId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>game</code></td>
<td><code class="typename"><span class="type">Game</span></code></td>
</tr>
<tr>
<td><code>position</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
</tr>
<tr>
<td><code>updatedAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
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

### DownloadKey (struct)


<p>
<p>A DownloadKey is often generated when a purchase is made, it
allows downloading uploads for a game that are not available
for free. It can also be generated by other means.</p>

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
<td><code class="typename"><span class="type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td><p>Game to which this download key grants access</p>
</td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
<td><p>Date this key was created at (often coincides with purchase time)</p>
</td>
</tr>
<tr>
<td><code>updatedAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
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


<div id="DownloadKey__TypeHint" class="tip-content">
<p>DownloadKey (struct) <a href="#/?id=downloadkey-struct">(Go to definition)</a></p>

<p>
<p>A DownloadKey is often generated when a purchase is made, it
allows downloading uploads for a game that are not available
for free. It can also be generated by other means.</p>

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
<td><code class="typename"><span class="type">Game</span></code></td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
</tr>
<tr>
<td><code>updatedAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
</tr>
<tr>
<td><code>ownerId</code></td>
<td><code class="typename"><span class="type builtin-type">number</span></code></td>
</tr>
</table>

</div>

### Build (struct)


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
<td><code class="typename"><span class="type" data-tip-selector="#BuildState__TypeHint">BuildState</span></code></td>
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
<td><code class="typename"><span class="type" data-tip-selector="#BuildFile__TypeHint">BuildFile</span>[]</code></td>
<td><p>Files associated with this build - often at least an archive,
a signature, and a patch. Some might be missing while the build
is still processing or if processing has failed.</p>
</td>
</tr>
<tr>
<td><code>user</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#User__TypeHint">User</span></code></td>
<td><p>User who pushed the build</p>
</td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
<td><p>Timestamp the build was created at</p>
</td>
</tr>
<tr>
<td><code>updatedAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
<td><p>Timestamp the build was last updated at</p>
</td>
</tr>
</table>


<div id="Build__TypeHint" class="tip-content">
<p>Build (struct) <a href="#/?id=build-struct">(Go to definition)</a></p>

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
<td><code class="typename"><span class="type">BuildState</span></code></td>
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
<td><code class="typename"><span class="type">BuildFile</span>[]</code></td>
</tr>
<tr>
<td><code>user</code></td>
<td><code class="typename"><span class="type">User</span></code></td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
</tr>
<tr>
<td><code>updatedAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
</tr>
</table>

</div>

### BuildState (enum)


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


<div id="BuildState__TypeHint" class="tip-content">
<p>BuildState (enum) <a href="#/?id=buildstate-enum">(Go to definition)</a></p>

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

### BuildFile (struct)


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
<td><code class="typename"><span class="type" data-tip-selector="#BuildFileState__TypeHint">BuildFileState</span></code></td>
<td><p>State of this file: created, uploading, uploaded, etc.</p>
</td>
</tr>
<tr>
<td><code>type</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#BuildFileType__TypeHint">BuildFileType</span></code></td>
<td><p>Type of this build file: archive, signature, patch, etc.</p>
</td>
</tr>
<tr>
<td><code>subType</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#BuildFileSubType__TypeHint">BuildFileSubType</span></code></td>
<td><p>Subtype of this build file, usually indicates compression</p>
</td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
<td><p>Date this build file was created at</p>
</td>
</tr>
<tr>
<td><code>updatedAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
<td><p>Date this build file was last updated at</p>
</td>
</tr>
</table>


<div id="BuildFile__TypeHint" class="tip-content">
<p>BuildFile (struct) <a href="#/?id=buildfile-struct">(Go to definition)</a></p>

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
<td><code class="typename"><span class="type">BuildFileState</span></code></td>
</tr>
<tr>
<td><code>type</code></td>
<td><code class="typename"><span class="type">BuildFileType</span></code></td>
</tr>
<tr>
<td><code>subType</code></td>
<td><code class="typename"><span class="type">BuildFileSubType</span></code></td>
</tr>
<tr>
<td><code>createdAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
</tr>
<tr>
<td><code>updatedAt</code></td>
<td><code class="typename"><span class="type builtin-type">RFCDate</span></code></td>
</tr>
</table>

</div>

### BuildFileState (enum)


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


<div id="BuildFileState__TypeHint" class="tip-content">
<p>BuildFileState (enum) <a href="#/?id=buildfilestate-enum">(Go to definition)</a></p>

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

### BuildFileType (enum)


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


<div id="BuildFileType__TypeHint" class="tip-content">
<p>BuildFileType (enum) <a href="#/?id=buildfiletype-enum">(Go to definition)</a></p>

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

### BuildFileSubType (enum)


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


<div id="BuildFileSubType__TypeHint" class="tip-content">
<p>BuildFileSubType (enum) <a href="#/?id=buildfilesubtype-enum">(Go to definition)</a></p>

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

### Verdict (struct)


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
<td><code class="typename"><span class="type" data-tip-selector="#Candidate__TypeHint">Candidate</span>[]</code></td>
<td><p>Candidates is a list of potentially interesting files, with a lot of additional info</p>
</td>
</tr>
</table>


<div id="Verdict__TypeHint" class="tip-content">
<p>Verdict (struct) <a href="#/?id=verdict-struct">(Go to definition)</a></p>

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
<td><code class="typename"><span class="type">Candidate</span>[]</code></td>
</tr>
</table>

</div>

### Candidate (struct)


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
<td><code class="typename"><span class="type" data-tip-selector="#Flavor__TypeHint">Flavor</span></code></td>
<td><p>Flavor is the type of a candidate - native, html, jar etc.</p>
</td>
</tr>
<tr>
<td><code>arch</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Arch__TypeHint">Arch</span></code></td>
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
<td><p><span class="tag">Optional</span> Spell contains raw output from <a href="https://github.com/itchio/wizardry">https://github.com/itchio/wizardry</a></p>
</td>
</tr>
<tr>
<td><code>windowsInfo</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#WindowsInfo__TypeHint">WindowsInfo</span></code></td>
<td><p><span class="tag">Optional</span> WindowsInfo contains information specific to native Windows candidates</p>
</td>
</tr>
<tr>
<td><code>linuxInfo</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#LinuxInfo__TypeHint">LinuxInfo</span></code></td>
<td><p><span class="tag">Optional</span> LinuxInfo contains information specific to native Linux candidates</p>
</td>
</tr>
<tr>
<td><code>macosInfo</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#MacosInfo__TypeHint">MacosInfo</span></code></td>
<td><p><span class="tag">Optional</span> MacosInfo contains information specific to native macOS candidates</p>
</td>
</tr>
<tr>
<td><code>loveInfo</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#LoveInfo__TypeHint">LoveInfo</span></code></td>
<td><p><span class="tag">Optional</span> LoveInfo contains information specific to Love2D bundles (<code>.love</code> files)</p>
</td>
</tr>
<tr>
<td><code>scriptInfo</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#ScriptInfo__TypeHint">ScriptInfo</span></code></td>
<td><p><span class="tag">Optional</span> ScriptInfo contains information specific to shell scripts (<code>.sh</code>, <code>.bat</code> etc.)</p>
</td>
</tr>
<tr>
<td><code>jarInfo</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#JarInfo__TypeHint">JarInfo</span></code></td>
<td><p><span class="tag">Optional</span> JarInfo contains information specific to Java archives (<code>.jar</code> files)</p>
</td>
</tr>
</table>


<div id="Candidate__TypeHint" class="tip-content">
<p>Candidate (struct) <a href="#/?id=candidate-struct">(Go to definition)</a></p>

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
<td><code class="typename"><span class="type">Flavor</span></code></td>
</tr>
<tr>
<td><code>arch</code></td>
<td><code class="typename"><span class="type">Arch</span></code></td>
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
<td><code class="typename"><span class="type">WindowsInfo</span></code></td>
</tr>
<tr>
<td><code>linuxInfo</code></td>
<td><code class="typename"><span class="type">LinuxInfo</span></code></td>
</tr>
<tr>
<td><code>macosInfo</code></td>
<td><code class="typename"><span class="type">MacosInfo</span></code></td>
</tr>
<tr>
<td><code>loveInfo</code></td>
<td><code class="typename"><span class="type">LoveInfo</span></code></td>
</tr>
<tr>
<td><code>scriptInfo</code></td>
<td><code class="typename"><span class="type">ScriptInfo</span></code></td>
</tr>
<tr>
<td><code>jarInfo</code></td>
<td><code class="typename"><span class="type">JarInfo</span></code></td>
</tr>
</table>

</div>

### Flavor (enum)


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
<tr>
<td><code>"msi"</code></td>
<td><p>Microsoft installer packages</p>
</td>
</tr>
</table>


<div id="Flavor__TypeHint" class="tip-content">
<p>Flavor (enum) <a href="#/?id=flavor-enum">(Go to definition)</a></p>

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
<tr>
<td><code>"msi"</code></td>
</tr>
</table>

</div>

### Arch (enum)


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


<div id="Arch__TypeHint" class="tip-content">
<p>Arch (enum) <a href="#/?id=arch-enum">(Go to definition)</a></p>

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

### WindowsInfo (struct)


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
<td><code class="typename"><span class="type" data-tip-selector="#WindowsInstallerType__TypeHint">WindowsInstallerType</span></code></td>
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


<div id="WindowsInfo__TypeHint" class="tip-content">
<p>WindowsInfo (struct) <a href="#/?id=windowsinfo-struct">(Go to definition)</a></p>

<p>
<p>Contains information specific to native windows executables
or installer packages.</p>

</p>

<table class="field-table">
<tr>
<td><code>installerType</code></td>
<td><code class="typename"><span class="type">WindowsInstallerType</span></code></td>
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

### WindowsInstallerType (enum)


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


<div id="WindowsInstallerType__TypeHint" class="tip-content">
<p>WindowsInstallerType (enum) <a href="#/?id=windowsinstallertype-enum">(Go to definition)</a></p>

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

### MacosInfo (struct)


<p>
<p>Contains information specific to native macOS executables
or app bundles.</p>

</p>

<p>
<span class="header">Fields</span> <em>none</em>
</p>


<div id="MacosInfo__TypeHint" class="tip-content">
<p>MacosInfo (struct) <a href="#/?id=macosinfo-struct">(Go to definition)</a></p>

<p>
<p>Contains information specific to native macOS executables
or app bundles.</p>

</p>
</div>

### LinuxInfo (struct)


<p>
<p>Contains information specific to native Linux executables</p>

</p>

<p>
<span class="header">Fields</span> <em>none</em>
</p>


<div id="LinuxInfo__TypeHint" class="tip-content">
<p>LinuxInfo (struct) <a href="#/?id=linuxinfo-struct">(Go to definition)</a></p>

<p>
<p>Contains information specific to native Linux executables</p>

</p>
</div>

### LoveInfo (struct)


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


<div id="LoveInfo__TypeHint" class="tip-content">
<p>LoveInfo (struct) <a href="#/?id=loveinfo-struct">(Go to definition)</a></p>

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

### ScriptInfo (struct)


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


<div id="ScriptInfo__TypeHint" class="tip-content">
<p>ScriptInfo (struct) <a href="#/?id=scriptinfo-struct">(Go to definition)</a></p>

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

### JarInfo (struct)


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


<div id="JarInfo__TypeHint" class="tip-content">
<p>JarInfo (struct) <a href="#/?id=jarinfo-struct">(Go to definition)</a></p>

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

### Receipt (struct)


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
<td><code class="typename"><span class="type" data-tip-selector="#Game__TypeHint">Game</span></code></td>
<td><p>The itch.io game installed at this location</p>
</td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Upload__TypeHint">Upload</span></code></td>
<td><p>The itch.io upload installed at this location</p>
</td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type" data-tip-selector="#Build__TypeHint">Build</span></code></td>
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


<div id="Receipt__TypeHint" class="tip-content">
<p>Receipt (struct) <a href="#/?id=receipt-struct">(Go to definition)</a></p>

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
<td><code class="typename"><span class="type">Game</span></code></td>
</tr>
<tr>
<td><code>upload</code></td>
<td><code class="typename"><span class="type">Upload</span></code></td>
</tr>
<tr>
<td><code>build</code></td>
<td><code class="typename"><span class="type">Build</span></code></td>
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


