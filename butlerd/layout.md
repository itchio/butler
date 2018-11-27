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

Use the `--log` command-line option to log all HTTP requests or TCP message exchanges.

## Making requests

By default, butlerd listens over HTTP and HTTPS.

When started, it will output a line of JSON to stdout with the following structure:

```
{
  "secret": "<the butlerd secret>"
  "http": {
    "address": "<host and port to make HTTP requests>"
  },
  "https": {
    "address": "<host and port to make HTTPS requests>",
    "ca": "<base64-encoded PEM block with a self-signed certificate, which is its own CA>",
  }
  "type": "butlerd/listen-notification"
}
```

butlerd supports HTTP/2 (on its http address).

> Note: butler tries to listen on port 13141 for HTTP, and 13142 for HTTPS, but
> if those are not available, it'll pick any free port.

It's important that you **do not hardcode** these ports in your client, but rather
parse butler's standard output line by line, trying to interpret each of these
as JSON, and only connecting when you get an object with `type` set to
`butlerd/listen-notification`.

butler may output lines to stdout that are not JSON - your client should not
crash if that is the case, but just ignore (or log) them.

## JSON-RPC 2.0 over TCP

It's easier to explain JSON-RPC over TCP, because it's a symmetrical connection.

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

## JSON-RPC 2.0 over HTTP

### Cheat sheet

(This only makes sense if you read the explanation below, probably)

client->server calls:

  * POST `/call/:method`
    * Body is *just* the params from a JSON-RPC request, as JSON
    * Must include `X-ID` header (the JSON-RPC request ID)
    * Must include `X-Secret` header
    * The `X-CID` header is required... (conversation ID, picked by client)
      * ...if there's going to be server->client requests
      * ...or you care about notifications
    * Status codes:
      * HTTP 200 if call was made successfully
        * ...but reply (full JSON-RPC object) might be a response with a JSON-RPC error
      * HTTP 400 if missing a header
      * HTTP 401 if the seret is wrong
      * HTTP 404 if you miss the route somehow
      * HTTP 424 (precondition failed) if
        * a server->client request couldn't be done because you weren't listening on that conversation's feed
      * HTTP 500 if you find a way to make butlerd blow up

server->client calls via Server-Sent Events (SSE):

  * GET `/feed?cid=CID&secret=SECRET`
    * Is a `text/event-stream`, long-running request
    * Lets server make server->client requests and server->client notifications
    * Each message is a proper JSON-RPC object (as JSON)
    * SSE message ids are meaningless
    * Status codes:
      * HTTP if you're missing `cid` or `secret`
      * HTTP 401 if the secret is wrong
      * HTTP 200 otherwise

client->server replies:

  * POST `/reply`
    * Must include `X-CID` and `X-Secret` headers
    * Body is a full JSON-RPC message, as JSON
    * Status codes:
      * HTTP 204 if successful

### Explanation

Now that we've seen JSON-RPC 2.0 over TCP, we can tackle HTTP.

The goal of the HTTP transport is to make it easy to see what requests
<https://itch.io/app> is making to butler, in the Chromium developer tools.


## Instances and connections

The recommended way to use butlerd is to have a **single instance**, but
**multiple connections**.

Multiple connections are useful because JSON-RPC notifications are not tied
to specific requests.

Long-running operations, like performing an install, or a launch, benefit
from having their own connection, so that their notifications can
be isolated from the rest, and show UI relevant to the item being installed
or launched.
It also lets us take advantage of Chromium's network stack - with HTTP/2,
requests are multiplexed over a single TCP connection, etc. 

However, HTTP is not symmetrical. And we don't want the client to have its own HTTP
server. Let's get started with simple requests.

Most requests are client->butlerd. The server doesn't need to emit any
notifications, it doesn't need to make butlerd->client requests, it's just
straightforward data retrieval.

Such requests can be done with a simple POST request.

(We'll use CURL for the examples)

```bash
curl -d '{}' http://127.0.0.1:13141/call/Version.Get
```

(Note that `-d` specifies the request's body *and* makes it a POST request
automatically)

This fails with HTTP 401 (Unauthorized). The secret needs to be included (in all requests!)
as the `X-Secret` HTTP header:


```bash
curl -H "X-Secret $(< secret)" -d '{}' http://127.0.0.1:13141/call/Version.Get
```

For testing, you can use the `--write-secret path/to/secret` command-line option to
write the secret to a file.

This will also fail, because it's missing the `X-ID` parameter, which must be set
to the ID of the json-rpc request.

```bash
curl -H "X-Secret $(< secret)" \
     -H "X-ID: 0" \
     -d '{}' http://127.0.0.1:13141/call/Version.Get
```

This finally succeeds with HTTP 200:

```json
{"id":0,"result":{"version":"head","versionString":"head, no build date"},"jsonrpc":"2.0"}
```

> Going forward, we'll assume all POST requests made include the proper headers

If we request something more complicated, like `Test.DoubleTwice`, we'll see
another error:

```bash
curl -d '{"number": 256}' http://127.0.0.1:13142/call/Test.DoubleTwice
```

```json
{
  "id": 0,
  "error": {
    "code": -32603,
    "message": "Server tried to call 'Test.Double', but no CID was specified ('X-CID' header is not set)"
  },
  "jsonrpc": "2.0"
}
```

CID stands for `conversation ID`, and is used to group together requests and replies when
using the HTTP transport.

Because each `client->butlerd` message is a separate HTTP request, and many concurrent
(but independent) requests can be made at the same time, we need a way to trace back these
requests to a specific `conversation`.

> Whatever TCP socket is used to carry these messages has nothing to do
> with the conversation ID. It's a logical identifier that requests belong to
> so messages can be exchanged properly.

If your application is simple enough (ie. it only ever makes one request at a time),
then you can pick a fixed CID. We'll pick `banana` for our purposes.

```bash
# along with other headers like X-Secret and X-ID
curl -H "X-CID: banana" \
     -d '{"number": 256}' http://127.0.0.1:13142/call/Test.DoubleTwice
```

This fails with another message:

```json
{
  "id": 0,
  "error": {
    "code": -32603,
    "message": "Server tried to call 'Test.Double', but nobody is listening to the feed for CID 'banana'"
  },
  "jsonrpc": "2.0"
}
```

See, normally, the flow would be:

  1. client calls 'Test.DoubleTwice' with number 256
  2. server calls 'Test.Double' with number 256
  3. client responds with number 512
  4. server responds with number 1024 (double the client's response)

But we're stuck at step 2, because the server has no way to contact the client.

It can't make calls in the response to the `POST /call/Test.DoubleTwice` because
that HTTP request must stay alive for the duration of the call, it can
only be ended by:

  * The client closing the connection (request is aborted)
  * The server closing the connection (daemon is shutting down)
  * The server responding with a result
  * The server responding with an error

Without the client listening to the feed for CID `banana`, the request would
be forever stuck. To avoid a deadlock, butlerd sends a helpful error instead.

To listen to the feed for a given CID, make a GET request:

```bash
curl -N "http://127.0.0.1:13141/feed?cid=banana&secret=(< secret)"
```

(We use `-N` here to prevent curl from buffering the response)

butlerd replies with a Server-Sent Event stream. You can look up [the
specification](https://html.spec.whatwg.org/multipage/server-sent-events.html#the-eventsource-interface)
for authoritative details, but basically:

  * The `Content-Type` of the response is set to `text/event-stream`
  * The connection stays open
  * Messages are sent in plain text, separated by two LF characters (`\n`)
  * That's it

So the feed is the means by which any server->client communication is done.

Each connection (each CID) has its own feed, so you can make several requests in parallel,
each with their own conversation.

Note that the secret and CID are passed as query parameters directly in the path -
this is because `EventSource` from the whatwg spec does not allow specifying headers.
Oh well.

When we open the feed, butlerd sends an `open` event:

```
event: open

```

Any requests or notifications from the server are sent as messages with a unique id - that
id is completely unrelated to JSON-RPC 2.0 IDs, they're required for most Server-Sent Events
clients to work, you can safely disregard them - what's important is the data.

If we make the same `Test.Double` request while listening to the feed for conversation `banana`,
we'll get these lines:

```
id: 0
data: {"method":"Test.Double","params":{"number":256},"id":0,"jsonrpc":"2.0"}

```

(Again, the `id: 0` here is a Server-Sent Events thing. It's meaningless. Disregard it.
I'm sorry, it's IDs all the way down, that's RPC for you).

Now that we know the server has made a request to us (the client), we can reply to it
by posting to `/reply` with our response:

```
# don't forget the `X-Secret` and `X-CID` headers!
# the 'id' here is the one from the JSON object in the data line of the SSE
curl -d '{"jsonrpc": "2.0", "id": 0, "result": {"number": 512}}'
```

Note that when POST-ing to `/call/:method`, the request's body is only the parameters.
But when we post to `/reply`, it's a full JSON-RPC message.

If the POST to `/reply` went well, we should get an HTTP 204 back.

> If we mess up the reply, butler prints the reason to stderr, prefixed by `jsonrpc2:`
> When developing clients, always log butler's stdout and stderr, they contain helpful
> hints.

Finally, we get our response back:

```json
{
  "id": 0,
  "result": {
    "number": 512
  },
  "jsonrpc": "2.0"
}
```

## HTTPS and HTTP/2

In order to support HTTP/2, butler generates its own self-signed certificate, which
is also its own CA (Certificate Authority).

It's included in the `butlerd/listen-notification` object when the http transport is used
(the default). You can use the `--write-cert path/to/cert` command-line flag to write
the certificate to a file for testing.

> Note: that the PEM block for the certificate (the `ca` field in the `https` field of
> the `butlerd/listen-notification` object) is **base64-encoded** (even though PEM blocks
> also contain base64-encoded content).
> This is because, technically, .pem files are not necessarily ASCII (or UTF-8) - they can
> contain any sequence of bytes before `-----BEGIN CERTIFICATE-----`. Make sure to decode
> that first.

Here's a complete example with curl:

```
butlerd daemon --json --dbpath butler.db \
  --write-cert cert.pem
  --write-secret secret

curl --cacert cert.pem \
     -H "X-Secret: $(< secret)" \
     -H "X-ID: 0" \
     -d '{}' \
     https://127.0.0.1:13142/call/Version.Get
```

> Note: the self-signed certificate is valid for `127.0.0.1` and `localhost`.
> However, you're encouraged to use 127.0.0.1 directly, to avoid any dns lookup
> shenanigans.

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

{{EVERYTHING}}
