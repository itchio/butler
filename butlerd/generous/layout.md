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

  * <https://broth.itch.zone/butler/windows-amd64/LATEST>

Where `windows-amd64` is one of:

  * `windows-386` - 32-bit Windows
  * `windows-amd64` - 64-bit Windows
  * `linux-amd64` - 64-bit Linux
  * `darwin-amd64` - 64-bit macOS

`LATEST` is a text file that contains a version number.

For example, if the contents of `LATEST` is `11.1.0`, then
the latest version of butler can be downloaded via:

  * <https://broth.itch.zone/butler/windows-amd64/11.1.0/butler/.zip>

Make sure when you unzip it, that the executable bit is set on Linux & macOS.

(The permissions are set properly in the zip file, but not all zip extractors
faithfully restore them)

### Friendly update deployment

See <https://github.com/itchio/itch/issues/1721>

{{EVERYTHING}}
