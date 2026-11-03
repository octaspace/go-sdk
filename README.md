# OctaSpace Go SDK

[![CI](https://github.com/octaspace/go-sdk/actions/workflows/ci.yml/badge.svg)](https://github.com/octaspace/go-sdk/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/octaspace/go-sdk.svg)](https://pkg.go.dev/github.com/octaspace/go-sdk)

Typed Go client for the [OctaSpace API](https://octa.space):
renting compute machines and render nodes, running sessions, hosted nodes and
their prices, the account balance, and public network and GPU market
statistics. It uses only the Go standard library and provides context
cancellation, typed errors, rate-limit awareness, retry for safe requests, and
optional host failover.

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
(`https://api-dev.octa.space`). See [getting started](docs/getting-started.md)
for client options, timeouts, logging, retries, and failover.

## Example

```bash
# Omit OCTA_API_KEY to call public endpoints only.
OCTA_API_KEY=<key> go run ./examples/basic
```

## Resources

| Resource | Main operations | Guide |
| --- | --- | --- |
| Compute | list machines, start, batch start, restart | [Compute](docs/resources.md#compute) |
| Render | list render nodes, start, batch start | [Render](docs/resources.md#render) |
| Sessions | running sessions, info, logs, stop, wait until ready | [Sessions](docs/sessions.md) |
| Hosting | the account's nodes, set prices | [Hosting](docs/resources.md#hosting) |
| Accounts | balance | [Accounts](docs/resources.md#accounts) |
| Network | public platform statistics | [Network](docs/resources.md#network) |
| Stats | public GPU market: now, over time, by location | [Stats](docs/resources.md#stats) |

## Documentation

- [Getting started and client configuration](docs/getting-started.md)
- [Resource reference](docs/resources.md)
- [Sessions](docs/sessions.md)
- [Lossless raw JSON](docs/raw-json.md)
- [Errors, rate limits, retries, and failover](docs/errors-and-reliability.md)
- [Change history](CHANGELOG.md)

## License

[MIT](LICENSE).
