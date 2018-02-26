# Caution: this is a draft

!> This document is a draft! It should not be used yet for implementing
   clients for buse. The API and recommendations are still subject to change.

# Overview

buse is butler's JSON-RPC 2.0 service. It is used heavily by the [itch.io app](https://itch.io/app)
as of version v25.

## Version

{{VERSION}}

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

{{EVERYTHING}}
