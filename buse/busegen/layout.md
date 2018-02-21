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

{{REQUESTS}}

# Notifications

Notifications

{{NOTIFICATIONS}}
