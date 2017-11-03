# buse

buse (butler service) is a [JSON-RPC 2.0](http://www.jsonrpc.org/specification) service over TCP that allows
using butler for long-running tasks (called operations) or one-off requests.

### Usage

Start butler in service mode:

```
butler service --json
```

It'll output a line of JSON similar to this one:

```
{"time":1509722356,"type":"result","value":{"address":"127.0.0.1:50890","type":"server-listening"}}
```

Dial the `address` to establish a connection, then send `\n`-separated valid JSON-RPC 2.0 requests
or notifications. The connection is bidirectional, so butler may send requests and notifications
the other way.

*Note: most JSON-RPC 2.0 implementations assume a unidirectional use-case, ie. client/server. These
won't work with buse.*

When you're done, just kill the butler process.

### Methods

There is no human documentation for buse, save for this README.

All requests, notifications and results can be found in the `types.go` file. 

### Client libraries

While JSON-RPC 2.0 and TCP are simple (unlike, say, [grpc](https://grpc.io/)), it's sometimes
more convenient to use client libraries and get straight to the point.

#### node.js client library

[node-buse](https://github.com/itchio/node-buse) is used by the [itch.io
app](https://github.com/itchio/itch) to access buse from the node.js runtime. It has very few dependencies
and ships with TypeScript typings so that all requests/notifications/results
are type-checked by the compiler.
