# OctaSpace Go SDK

[![CI](https://github.com/octaspace/go-sdk/actions/workflows/ci.yml/badge.svg)](https://github.com/octaspace/go-sdk/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/octaspace/go-sdk.svg)](https://pkg.go.dev/github.com/octaspace/go-sdk)

The official Go library for the [OctaSpace](https://octa.space) API. Rent
compute machines and render nodes, manage sessions and the nodes you host, and
read GPU market statistics from Go code.

**Contents:** [Installation](#installation) ·
[Quick start](#quick-start) · [Authentication](#authentication) ·
[Handling errors](#handling-errors) · [API overview](#api-overview) ·
[Documentation](#documentation) · [Versioning](#versioning)

## Installation

```bash
go get github.com/octaspace/go-sdk
```

Requires Go 1.20 or later. The library depends only on the standard library.

## Quick start

1. [Create an API key](#authentication) and export it:

   ```bash
   export OCTA_API_KEY="octa_ak_..."
   ```

2. Run this program:

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

The SDK authenticates with an **API key** of your OctaSpace account. You create
and delete API keys in the OctaSpace dashboard:

1. Sign in to the [OctaSpace dashboard](https://cube-dev.octa.space).
2. In the sidebar, open **Account → Credentials**.
3. In the **API Keys** section, click **New key**.
4. Enter a **Label** that tells you where the key is used, for example
   `billing-service`, and click **Create key**.
5. Copy the key. It starts with `octa_ak_`.

> [!IMPORTANT]
> The key is shown in full only once, right after you create it. Copy it
> before you close the dialog. If you lose it, delete it and create a new one.

Pass the key to `NewClient`. The SDK sends it in the
`Authorization: Bearer <key>` header.

```go
client := octaspace.NewClient(os.Getenv("OCTA_API_KEY"))
```

> [!WARNING]
> An API key acts on behalf of your account: it can start sessions that are
> charged to your balance. Keep it out of source code and version control. If
> a key leaks, delete it under **Account → Credentials → API Keys**; every
> program that uses it stops working at once.

The **API Keys** list shows each key's label, its first characters and when it
was last used. Use a separate key for each program or machine, so that you can
delete one without affecting the others. The number of keys per account is
limited; the counter next to **API Keys** shows how many you have.

### Calls without an API key

These methods also work without an API key. Pass an empty string to
`NewClient` to use them anonymously:

| Method | Without a key |
| --- | --- |
| `Network.Info` | Same result |
| `Stats.GPUCurrent`, `Stats.GPUHistory`, `Stats.GPULocations` | Same result |
| `Compute.List`, `Render.List` | Refreshed about once an hour instead of every 10 seconds |

Every other method requires an API key and returns an `*AuthenticationError`
without one.

## Handling errors

Every method returns a typed error. Check it with `errors.As` or a helper:

```go
uuid, err := client.Compute.Start(ctx, params)
switch {
case octaspace.IsAuthError(err):
	// the API key is missing, wrong or deleted
case octaspace.IsValidation(err):
	// the request was rejected; err.Error() says why
case octaspace.IsRateLimit(err):
	// too many requests; wait APIError.RetryAfter seconds
case err != nil:
	return err
}
```

See [Errors, rate limits and retries](docs/errors-and-reliability.md).

## API overview

| Resource | Dashboard section | Operations | Guide |
| --- | --- | --- | --- |
| `client.Compute` | Marketplace → Compute | List machines, start, batch start, restart | [Compute](docs/resources.md#compute) |
| `client.Render` | Render → Launch Nodes | List render nodes, start, batch start | [Render](docs/resources.md#render) |
| `client.Sessions` | Marketplace → Sessions, Render → Render Nodes | List, get, logs, stop, wait until ready | [Sessions](docs/sessions.md) |
| `client.Hosting` | Hosting → Nodes | Your nodes, set prices | [Hosting](docs/resources.md#hosting) |
| `client.Accounts` | Account → Billing | Balance | [Accounts](docs/resources.md#accounts) |
| `client.Network` | — | Platform statistics | [Network](docs/resources.md#network) |
| `client.Stats` | — | GPU market: current, history, locations | [Stats](docs/resources.md#stats) |

Every type and method is documented on
[pkg.go.dev](https://pkg.go.dev/github.com/octaspace/go-sdk).

## Documentation

| Guide | Covers |
| --- | --- |
| [Client configuration](docs/getting-started.md) | Options and defaults, context, logging, custom HTTP client |
| [Resources](docs/resources.md) | Compute, render, hosting, balance, network and GPU statistics |
| [Sessions](docs/sessions.md) | Waiting until ready, connecting, logs, stopping |
| [Errors, rate limits and retries](docs/errors-and-reliability.md) | Error types, limits, automatic retries |
| [Raw JSON responses](docs/raw-json.md) | Responses exactly as the API sent them |
| [Changelog](CHANGELOG.md) | Changes in each release |

## Versioning

The library follows [Semantic Versioning](https://semver.org). Until version
1.0, a minor release can contain breaking changes; the
[changelog](CHANGELOG.md) marks them as **Breaking**.

## License

[MIT](LICENSE)
