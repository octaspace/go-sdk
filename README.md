# OctaSpace Go SDK

Go client library for the [OctaSpace](https://octa.space) API — a decentralized marketplace for GPU compute, VPN relays, and rendering.

- Zero external dependencies (standard library only)
- Resource-oriented API: `client.Nodes.List(ctx, nil)`, `client.Services.MR.Create(ctx, params)`
- Typed error hierarchy with `errors.As` support
- Automatic retry with exponential backoff on transient errors (429, 5xx)
- `context.Context` support on every method — timeouts and cancellation work out of the box

## Requirements

Go 1.20 or later.

## Installation

```bash
go get github.com/octaspace/go-sdk
```

## Quick start

```go
package main

import (
    "context"
    "fmt"
    "os"

    octaspace "github.com/octaspace/go-sdk"
)

func main() {
    ctx := context.Background()
    client := octaspace.NewClient(os.Getenv("OCTA_API_KEY"))

    // Public endpoint — no API key required
    info, err := client.Network.Info(ctx)
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
    fmt.Printf("Nodes online: %d\n", info["nodes"])

    // Authenticated endpoint
    account, err := client.Accounts.Profile(ctx)
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
    fmt.Printf("Logged in as %s\n", account.Email)
}
```

## Creating a client

```go
// With API key — access to all endpoints
client := octaspace.NewClient(os.Getenv("OCTA_API_KEY"))

// Without API key — only public endpoints (e.g. Network.Info)
client := octaspace.NewClient("")
```

The API key is sent as the `Authorization` header. When the key is empty the header is omitted entirely — public endpoints respond normally, authenticated endpoints return HTTP 401.

### Options

All options are passed as variadic arguments to `NewClient`:

```go
client := octaspace.NewClient(apiKey,
    octaspace.WithBaseURL("https://api.octa.space"), // default
    octaspace.WithTimeout(15*time.Second),            // per-request timeout (default: 30s)
    octaspace.WithMaxRetries(3),                      // retries on 429/5xx (default: 2)
    octaspace.WithLogger(myLogger),                   // request/response logging
    octaspace.WithHTTPClient(myHTTPClient),           // custom http.Client
    octaspace.WithUserAgent("my-app/1.0"),            // override User-Agent
)
```

#### Logging

`WithLogger` accepts any value that satisfies the `Logger` interface:

```go
type Logger interface {
    Debugf(format string, v ...any)
    Errorf(format string, v ...any)
}
```

The standard library's `*log.Logger` satisfies this interface directly:

```go
import "log"

l := log.New(os.Stderr, "[octaspace] ", log.LstdFlags)
client := octaspace.NewClient(apiKey, octaspace.WithLogger(l))
```

## Resources

The client is organized into resource groups that mirror the API structure. Every method takes `context.Context` as the first argument.

### Network (public)

```go
// GET /network — no authentication required
info, err := client.Network.Info(ctx)
// info is map[string]any — contains blockchain stats, marketplace summary, node counts, etc.
```

### Accounts

```go
// GET /accounts
account, err := client.Accounts.Profile(ctx)
// account.AccountUUID, account.Email, account.Wallet, account.Balance (BigInt)

// GET /accounts/balance — returns *big.Int denominated in Wei
balance, err := client.Accounts.Balance(ctx)
fmt.Printf("Balance: %s wei\n", balance.String())

// POST /accounts — generate a new wallet for the account
account, err := client.Accounts.GenerateWallet(ctx)
```

### Nodes

```go
// GET /nodes — all nodes
nodes, err := client.Nodes.List(ctx, nil)

// GET /nodes?country=US — filter by country ISO code
country := "US"
nodes, err := client.Nodes.List(ctx, &octaspace.ListNodesParams{Country: &country})

// GET /nodes?app_uuid=<uuid> — filter by compatible application
appUUID := "3f2a1..."
nodes, err := client.Nodes.List(ctx, &octaspace.ListNodesParams{AppUUID: &appUUID})

// GET /nodes/:id
node, err := client.Nodes.Find(ctx, 42)

// GET /nodes/:id/ident — returns identity file as []byte
identBytes, err := client.Nodes.DownloadIdent(ctx, 42)

// GET /nodes/:id/logs — returns log archive as []byte
logBytes, err := client.Nodes.DownloadLogs(ctx, 42)

// PATCH /nodes/:id/prices — update your node's pricing
err := client.Nodes.UpdatePrices(ctx, 42, map[string]float64{
    "gpu_hour": 0.5,
    "cpu_hour": 0.1,
})

// GET /nodes/:id/reboot
err := client.Nodes.Reboot(ctx, 42)
```

**Node struct highlights:**

```go
type Node struct {
    ID       int64  // node identifier
    IP       string
    State    string // "idle", "busy", etc.
    OSN      string // OctaSpace Node agent version
    Uptime   int64  // seconds
    Location struct {
        City      string
        Country   string
        Latitude  float64
        Longitude float64
    }
    Prices struct {
        Base    int64 // Wei (node-owner-configured)
        Storage int64
        Traffic int64
    }
    System struct {
        CPUModelName   string
        CPUCores       int
        CPULoadPercent float64
        OSVersion      string
        Disk  struct{ Free, Size, Used int64; UsedPercent float64 }
        Memory struct{ Free, Size int64 }
        GPUs []struct {
            Model          string
            MemTotalMB     int
            GPUTemperature int
            GPUUtilization int
            // ... and more telemetry fields
        }
    }
}
```

### Sessions

Sessions represent active or recent service deployments.

```go
// GET /sessions — active sessions
sessions, err := client.Sessions.List(ctx, nil)

// GET /sessions?recent=true — include recently finished sessions
recent := true
sessions, err := client.Sessions.List(ctx, &octaspace.ListSessionsParams{Recent: &recent})
```

**Note:** The `/sessions` and `/sessions?recent=true` endpoints return overlapping but different field sets. The `recent=true` response includes `FinishedAt`, `RX`, `TX`, `TerminationReason` and returns `Duration` as a quoted string. The SDK handles both transparently via `FlexInt64`.

### Services — Machine Rental (compute)

```go
// GET /services/mr — available machines for rent
machines, err := client.Services.MR.List(ctx, nil)

// With query filters (raw url.Values)
import "net/url"
params := url.Values{"country": []string{"DE"}}
machines, err := client.Services.MR.List(ctx, params)

// POST /services/mr — create a machine rental session
resp, err := client.Services.MR.Create(ctx, &octaspace.MachineRentalCreateParams{
    NodeID:   42,
    DiskSize: 50,   // GB
    Image:    "ubuntu:22.04",
    App:      "",   // empty = custom image; or use App.UUID from Apps.List
    Envs: map[string]string{
        "JUPYTER_TOKEN": "mysecret",
    },
    Ports:        []int{8888},
    HTTPPorts:    []int{8080},
    StartCommand: "",
    Entrypoint:   "",
})
// resp.UUID is the session UUID — use it with Services.Session(uuid)
```

**MachineRental pricing fields** (all `base_usd` / `storage_usd` / `traffic_usd` are scaled ×10 000):

```go
// base_usd: 1750 → $0.175/hr
pricePerHour := float64(m.BaseUSD) / 10_000

// Or use the pre-computed field:
fmt.Printf("$%.4f/hr\n", m.TotalPriceUSD)
```

**Network speed fields** on `MachineRental` and `VPNRelay` require conversion — raw values > 100 000 are in bytes/sec, not Mbit/s. Use the provided helpers:

```go
fmt.Printf("Down: %.0f Mbps\n", machine.NetDownMbps())
fmt.Printf("Up:   %.0f Mbps\n", machine.NetUpMbps())
```

### Services — VPN

```go
// GET /services/vpn — available VPN relay nodes
relays, err := client.Services.VPN.List(ctx, nil)
for _, r := range relays {
    fmt.Printf("[%d] %s %s  ↓%.0f Mbps  $%.4f/GB\n",
        r.NodeID, r.Country, r.City, r.DownloadMbps(), r.PricePerGB)
}

// POST /services/vpn — create a VPN session
// SubKind: "wg" (WireGuard), "openvpn", "ss" (Shadowsocks), "v2ray"
resp, err := client.Services.VPN.Create(ctx, &octaspace.VPNCreateParams{
    NodeID:  247,
    SubKind: "wg",
})
// resp.UUID → use with Services.Session(uuid)
```

### Services — Render

```go
// GET /services/render — active/available render jobs
jobs, err := client.Services.Render.List(ctx, nil)
// Each job is map[string]any — schema varies

// POST /services/render
resp, err := client.Services.Render.Create(ctx, map[string]any{
    "node_id": 42,
    "image":   "blender:4.0",
})
```

### Services — Per-session operations

Once you have a session UUID (from `MR.Create`, `VPN.Create`, etc.), use `Services.Session(uuid)` to operate on it:

```go
proxy := client.Services.Session("550e8400-e29b-41d4-a716-446655440000")

// GET /services/:uuid/info
info, err := proxy.Info(ctx)
fmt.Println(info.VPNConfig) // WireGuard/OpenVPN config — paste into wg0.conf

// GET /services/:uuid/logs
logs, err := proxy.Logs(ctx)

// POST /services/:uuid/stop
err := proxy.Stop(ctx, nil)

// Stop with a quality score (1–5)
score := 5
err := proxy.Stop(ctx, &octaspace.StopParams{Score: &score})
```

### Apps

```go
// GET /apps — application catalog (GPU apps, OS images, mining software, etc.)
apps, err := client.Apps.List(ctx)
for _, a := range apps {
    fmt.Printf("%-30s  category=%-15s  image=%s\n", a.Name, a.Category, a.Image)
}
// Use a.UUID when creating machine rental sessions with a specific application
```

### Idle Jobs

```go
// GET /idle_jobs/:node_id/:job_id
job, err := client.IdleJobs.Find(ctx, nodeID, jobID)

// GET /idle_jobs/:node_id/:job_id/logs — returns raw bytes
logBytes, err := client.IdleJobs.Logs(ctx, nodeID, jobID)
```

## Error handling

All errors are typed and form a hierarchy rooted at `*APIError`. Use `errors.As` to inspect them:

```go
nodes, err := client.Nodes.List(ctx, nil)
if err != nil {
    var authErr *octaspace.AuthenticationError
    var notFound *octaspace.NotFoundError
    var rateLimit *octaspace.RateLimitError
    var serverErr *octaspace.ServerError
    var netErr *octaspace.NetworkError

    switch {
    case errors.As(err, &authErr):
        // HTTP 401 — check your API key
        fmt.Println("invalid or expired API key")
    case errors.As(err, &notFound):
        // HTTP 404
        fmt.Println("resource not found")
    case errors.As(err, &rateLimit):
        // HTTP 429 — RetryAfter is populated when the server sends Retry-After header
        fmt.Printf("rate limited, retry after %ds\n", rateLimit.RetryAfter)
    case errors.As(err, &serverErr):
        // HTTP 5xx
        fmt.Printf("server error (request-id: %s)\n", serverErr.RequestID)
    case errors.As(err, &netErr):
        // transport failure (DNS, TLS, connection refused)
        fmt.Printf("network error: %v\n", netErr.Err)
    default:
        fmt.Println(err)
    }
}
```

**Convenience helpers** for the most common cases:

```go
octaspace.IsAuthError(err)  // true for *AuthenticationError
octaspace.IsNotFound(err)   // true for *NotFoundError
octaspace.IsRateLimit(err)  // true for *RateLimitError
```

**All error subtypes embed `*APIError`**, so you can always unwrap to it:

```go
var apiErr *octaspace.APIError
if errors.As(err, &apiErr) {
    fmt.Printf("HTTP %d, request-id: %s\n", apiErr.StatusCode, apiErr.RequestID)
}
```

### Error types reference

| Type | HTTP status | Description |
|---|---|---|
| `*NetworkError` | — | Transport failure (DNS, TLS, timeout, refused) |
| `*APIError` | any 4xx/5xx | Base type; all subtypes embed it |
| `*AuthenticationError` | 401 | Invalid or missing API key |
| `*PermissionError` | 403 | API key lacks permission for this operation |
| `*NotFoundError` | 404 | Resource does not exist |
| `*ValidationError` | 422 | Request parameters failed server-side validation |
| `*RateLimitError` | 429 | Too many requests; check `RetryAfter` |
| `*ServerError` | 5xx | Internal server error |

## Retries

The client retries idempotent requests (GET) on transient errors (HTTP 429, 500, 502, 503, 504) using exponential backoff with ±25% jitter:

| Attempt | Base delay |
|---|---|
| 1st retry | 500ms |
| 2nd retry | 1s |
| 3rd retry | 2s |

When the server includes a `Retry-After` header, its value is used instead.

POST and PATCH requests are **never** retried automatically.

Configure retry behaviour:

```go
// disable retries entirely
client := octaspace.NewClient(apiKey, octaspace.WithMaxRetries(0))

// up to 5 retries
client := octaspace.NewClient(apiKey, octaspace.WithMaxRetries(5))
```

## Special types

### BigInt

Account balances are denominated in Wei and can exceed the range of `int64` (~9.2×10¹⁸). The SDK uses `BigInt`, a thin wrapper around `*big.Int`, that marshals and unmarshals as a bare JSON integer:

```go
balance, err := client.Accounts.Balance(ctx)
// balance is *big.Int
fmt.Printf("%s wei\n", balance.String())

// Convert to ETH (1 ETH = 10^18 Wei)
eth := new(big.Float).Quo(
    new(big.Float).SetInt(balance),
    new(big.Float).SetFloat64(1e18),
)
fmt.Printf("%.6f OCTA\n", eth)
```

### FlexInt64

Some fields (e.g. `Session.Duration`, `Session.StartedAt`) are returned as bare integers by one endpoint and as quoted strings by another (a known API inconsistency). `FlexInt64` deserializes both forms transparently. Access the underlying value with `.Int64()`.

### Network speed conversion

`MachineRental.NetDownMbits` / `NetUpMbits` and `VPNRelay.NetDownMbits` / `NetUpMbits` store raw API values. When the raw value exceeds 100 000, it is in bytes/sec and must be divided by 125 000 to obtain Mbit/s. Use the provided helpers — they apply this logic automatically:

```go
fmt.Printf("Down: %.0f Mbps\n", machine.NetDownMbps())
fmt.Printf("Up:   %.0f Mbps\n", machine.NetUpMbps())

fmt.Printf("Down: %.0f Mbps\n", relay.DownloadMbps())
fmt.Printf("Up:   %.0f Mbps\n", relay.UploadMbps())
```

## Running examples

A complete working example is in `examples/basic`:

```bash
# Public endpoints only
go run ./examples/basic

# All endpoints
OCTA_API_KEY=<your-key> go run ./examples/basic
```

## Smoke tests

`cmd/smoke` verifies the SDK against the live API and reports pass/fail for every endpoint:

```bash
# Public endpoints only
go run ./cmd/smoke

# All endpoints
OCTA_API_KEY=<your-key> go run ./cmd/smoke

# Machine-readable JSON output
OCTA_API_KEY=<your-key> go run ./cmd/smoke -json

# Run a single check
go run ./cmd/smoke -only network.info

# Custom base URL and timeout
go run ./cmd/smoke -base-url https://api.octa.space -timeout 10s
```

## Running tests

Tests use `httptest.Server` — no API key or network access required.

```bash
go test ./...

# With race detector
go test -race ./...

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## Project layout

```
octaspace/
├── client.go               — NewClient, options, HTTP transport, retry logic
├── errors.go               — typed error hierarchy
├── bigint.go               — BigInt and FlexInt64 types
├── accounts.go             — Accounts resource
├── apps.go                 — Apps resource
├── idle_jobs.go            — IdleJobs resource
├── network.go              — Network resource
├── nodes.go                — Nodes resource
├── services.go             — ServicesResource (aggregates MR, VPN, Render, Session)
├── services_mr.go          — Machine Rental sub-resource
├── services_render.go      — Render sub-resource
├── services_session_proxy.go — per-session operations (Info, Logs, Stop)
├── services_vpn.go         — VPN sub-resource
├── sessions.go             — Sessions resource
├── examples/basic/         — runnable usage example
└── cmd/smoke/              — live API smoke test tool
```

## License

MIT — see [LICENSE](LICENSE).
