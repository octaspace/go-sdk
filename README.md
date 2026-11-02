# OctaSpace Go SDK

[![CI](https://github.com/octaspace/go-sdk/actions/workflows/ci.yml/badge.svg)](https://github.com/octaspace/go-sdk/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/octaspace/go-sdk.svg)](https://pkg.go.dev/github.com/octaspace/go-sdk)

Typed Go client for the [OctaSpace API](https://octa.space): compute rentals,
VPN relays, render services, nodes, sessions, and account operations. It uses
only the Go standard library and provides context cancellation, typed errors,
retry for safe requests, and optional host failover.

## Requirements and installation

Go 1.20 or later is required.

```bash
go get github.com/octaspace/go-sdk
```

```go
client := octaspace.NewClient(os.Getenv("OCTA_API_KEY"))
info, err := client.Network.Info(context.Background()) // public endpoint
```

Pass an empty key for public endpoints. Authenticated requests send the key as
`Authorization: Bearer <key>`; keys (`octa_ak_…`) are issued in Cube under
Profile → Credentials. The default base URL is `octaspace.BaseURLDev`
(`https://api-dev.octa.space`), the test stand of the new API, until its
production host is announced. See [getting started](docs/getting-started.md)
for client options, timeouts, logging, retries, and failover.

## Run and test

```bash
# Runnable example; omit OCTA_API_KEY to call public endpoints only.
go run ./examples/basic
OCTA_API_KEY=<key> go run ./examples/basic

# Offline unit tests.
go test ./...
go test -race ./...
go vet ./...

# Optional live smoke checks.
go run ./cmd/smoke
OCTA_API_KEY=<key> go run ./cmd/smoke -json
```

The smoke command reads live API data; it is not part of the unit-test suite.
More detail: [testing and smoke checks](docs/testing.md).

## Resources

| Resource | Main operations | Guide |
| --- | --- | --- |
| Network | `Network.Info` | [Network](docs/resources.md#network) |
| Accounts | profile, balance, wallet generation | [Accounts](docs/resources.md#accounts) |
| Apps | list application templates | [Apps](docs/resources.md#apps) |
| Nodes | list, find, price updates, reboot, downloads | [Nodes](docs/resources.md#nodes) |
| Sessions | active/recent list | [Sessions](docs/sessions.md#listing-sessions) |
| Machine Rental | list and create compute sessions | [Machine rentals](docs/machine-rentals.md) |
| VPN | list relays and create VPN sessions | [VPN](docs/resources.md#vpn) |
| Render workers/jobs | `/services/r` and `/services/render` | [Render](docs/resources.md#render) |
| Session operations | info, logs, stop | [Session operations](docs/sessions.md#session-operations) |
| Idle jobs | inspect jobs and fetch logs | [Idle jobs](docs/resources.md#idle-jobs) |

## Documentation

- [Getting started and client configuration](docs/getting-started.md)
- [Machine-rental request and response forms](docs/machine-rentals.md)
- [Sessions and session details](docs/sessions.md)
- [Resource reference](docs/resources.md)
- [Lossless raw JSON](docs/raw-json.md)
- [Swagger/live contract inventory](docs/contracts.md)
- [Errors, retries, and failover](docs/errors-and-reliability.md)
- [Testing and live smoke checks](docs/testing.md)
- [Change history](CHANGELOG.md)

## License

[MIT](LICENSE).
