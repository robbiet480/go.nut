[![GoDoc](https://godoc.org/github.com/robbiet480/go.nut?status.svg)](https://godoc.org/github.com/robbiet480/go.nut)

# go.nut
go.nut is a Golang library for interacting with [NUT (Network UPS Tools)](https://networkupstools.org/)

# Getting started
```
import "github.com/robbiet480/go.nut"
```

Check out the examples in [`example_test.go`](example_test.go). For full documentation, see the [Godocs](https://godoc.org/github.com/robbiet480/go.nut).

# Connection lifetime and cancellation

`Connect` keeps its existing signature. `ConnectContext(ctx, host, port...)`
uses the context for dialing, the VER/NETVER handshake, and subsequent I/O.
A failed handshake returns an error and closes the connection. The context
belongs to the connection: cancelling it makes the client unusable.

Reads and writes have a two-second operation deadline, capped by the context
deadline when present. Use a context deadline to bound a whole sequence of
commands; operation deadlines alone do not provide that overall budget.
`Connect` uses a background context and does not add an overall dial deadline.

Always call `Close` or `Disconnect` when finished. `Close` releases the socket
without LOGOUT and is safe to call repeatedly. `Disconnect` attempts LOGOUT
and releases the socket even if LOGOUT fails. Commands on a single client
must not be used concurrently.

The context-aware implementation requires **Go 1.21 or later** for
`context.AfterFunc`. The existing module-less layout is retained. To run the
suite, place the checkout at `$GOPATH/src/github.com/robbiet480/go.nut` and run:

```sh
GO111MODULE=off GOWORK=off go test -race github.com/robbiet480/go.nut
GO111MODULE=off GOWORK=off go vet github.com/robbiet480/go.nut
```

# Other resources
* [Network protocol information](http://networkupstools.org/docs/developer-guide.chunked/ar01s09.html)

# Contributing

1. [Fork it](https://github.com/robbiet480/go.nut)
2. Create your feature branch (`git checkout -b my-new-feature`)
3. Make sure `golint` and `go vet` run successfully.
4. `go fmt` your code!
5. Commit your changes (`git commit -am "Add some feature"`)
6. Push to the branch (`git push origin my-new-feature`)
7. Create a new Pull Request

# License
[MIT](LICENSE)
