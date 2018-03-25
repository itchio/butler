# Caution: this is a draft

!> This document is a draft! It should not be used yet for implementing
   clients for butlerd. The API and recommendations are still subject to change.

# Overview

butlerd (butler daemon) is a JSON-RPC 2.0 service.

It is used heavily by the [itch.io app](https://itch.io/app) as of version 25.x.

## Starting an instance of the daemon

To start an instance, run:

```bash
butler daemon --json
```

However, without any other action on your part, it will quickly exit.

## Exchanging secrets

Each butlerd instance has a secret, which you should generate before starting it.

It can't be passed as a command-line argument or an environment variable, as both
of these can easily be snooped on.

Instead, they're passed by exchanging JSON lines over stdin/stdout.

Every json line butler prints has the following fields

  * `time` (number): A unix timestamp (second since epoch)
  * `type` (string): The message's type (`"log"`, etc.)

To request a secret, butler daemon sends a line with type `butlerd/secret-request`,
and the additional fields `minLength` (number), the minimum number of characters
the secret should have.

Here's an example `butlerd/secret-request` line, printed to stdout:

```json
{"type":"butlerd/secret-request","minLength":256,"time":12345678}
```

You should reply with a JSON line of your own (sending it to butlerd's stdin), along the lines of:

```json
{"type":"butlerd/secret-result","secret":"<your secret goes here>"}
```

> Don't re-use secrets. Generate a new one for every instance.

Once butlerd has received the secret, it will start listening on a random TCP port,
and send a JSON line of type `butlerd/listen-notification`, like this one:

```json
{"type":"butlerd/listen-notification","address":"127.0.0.1:60435","time":12345678}
```

If the secret is too short, or never sent, etc., butlerd will abort, returning a non-zero exit code
and printing the reason both to stdout as a JSON line, and to stderr:

```bash
$ butler daemon -j
stdout> {"minLength":256,"time":1521976589,"type":"butlerd/secret-request"}
stdout> {"message":"butlerd: Timed out while waiting for secret","time":1521976590,"type":"error"}
stderr> bailing out: butlerd: Timed out while waiting for secret
```

We recommend parsing all JSON lines butler prints to stdout, especially those of type `log`, with
the `level` and `message` properties, and printing them to your client's logs to help diagnose
potential connection issues.

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

## Handshake

When first connecting to butlerd's endpoint, it will send a `Handshake` request,
containing a `message`.

The client (you) should reply with a `signature`, which is simply the SHA-256 sum
of the secret concatenated with the message.

In pseudo-code, the whole flow goes like this:

```
instance = createButlerdInstance()

minSecretLength = waitForSecretRequest(instance)
secret = generateSecret(minSecretLength)
sendSecret(instance, secret)

address = waitForListenNotification(instance)
connection = connectToButler(address)

handshakeMessage = waitForHandshakeRequest(connection)
signature = computeSha256Hash(secret + handshakeMessage)
replyToHandshakeRequest(signature)
```

After replying to the Handshake with the proper signature, butlerd will
accept any other requests.

## Instances and connections

The recommended way to use butlerd is to have a **single instance**, but
**multiple connections**.

Multiple connections are useful because JSON-RPC notifications are not tied
to specific requests.

Long-running operations, like performing an install, or a launch, benefit
from having their own connection, so that their notifications can
be isolated from the rest, and show UI relevant to the item being installed
or launched.

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
