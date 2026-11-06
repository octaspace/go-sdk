# OctaSpace Go SDK

[![CI](https://github.com/octaspace/go-sdk/actions/workflows/ci.yml/badge.svg)](https://github.com/octaspace/go-sdk/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/octaspace/go-sdk.svg)](https://pkg.go.dev/github.com/octaspace/go-sdk)

The official Go library for the [OctaSpace](https://octa.space) API: rent
compute machines and render nodes, manage sessions and hosted nodes, and read
GPU market statistics.

## Installation

```bash
go get github.com/octaspace/go-sdk
```

Requires Go 1.20 or later. The library has no dependencies outside the
standard library.

## Usage

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	octaspace "github.com/octaspace/go-sdk"
)

func main() {
	ctx := context.Background()
	client := octaspace.NewClient(os.Getenv("OCTA_API_KEY"))

	balance, err := client.Accounts.Balance(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Balance:", balance.Balance) // Balance: $12.500000

	machines, err := client.Compute.List(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for _, m := range machines {
		fmt.Println(m.NodeNumber, m.City, m.BasePrice, "per hour")
	}
}
```

Create one client and reuse it: it is safe for concurrent use.

## Authentication

Create an API key in your OctaSpace account under **Profile → Credentials** and
pass it to `NewClient`. Keep it out of source code; reading it from an
environment variable, as above, is the usual way.

Network and GPU market statistics and the machine listings are public: pass an
empty key to use them without an account.

## Handling errors

Every method returns a typed error. Check it with `errors.As` or a helper:

```go
uuid, err := client.Compute.Start(ctx, params)
switch {
case octaspace.IsValidation(err):
	// the request was rejected; the error message says why
case octaspace.IsRateLimit(err):
	// too many requests; see APIError.RetryAfter
case err != nil:
	return err
}
```

See [errors and retries](docs/errors-and-reliability.md).

## API

| Resource | Operations | Guide |
| --- | --- | --- |
| `client.Compute` | list machines, start, batch start, restart | [Compute](docs/resources.md#compute) |
| `client.Render` | list render nodes, start, batch start | [Render](docs/resources.md#render) |
| `client.Sessions` | list, get, logs, stop, wait until ready | [Sessions](docs/sessions.md) |
| `client.Hosting` | your hosted nodes, set prices | [Hosting](docs/resources.md#hosting) |
| `client.Accounts` | balance | [Accounts](docs/resources.md#accounts) |
| `client.Network` | platform statistics | [Network](docs/resources.md#network) |
| `client.Stats` | GPU market: current, history, locations | [Stats](docs/resources.md#stats) |

Every type and method is documented on
[pkg.go.dev](https://pkg.go.dev/github.com/octaspace/go-sdk).

## Documentation

- [Client configuration](docs/getting-started.md): timeouts, retries, logging,
  custom HTTP client
- [Resources](docs/resources.md)
- [Sessions](docs/sessions.md)
- [Errors, rate limits and retries](docs/errors-and-reliability.md)
- [Raw JSON responses](docs/raw-json.md)
- [Changelog](CHANGELOG.md)

## Versioning

The library follows [Semantic Versioning](https://semver.org). Until 1.0, a
minor release may contain breaking changes; they are marked in the
[changelog](CHANGELOG.md).

## License

[MIT](LICENSE)
