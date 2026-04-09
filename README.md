# go-sdk

Go client for the [OctaSpace](https://octa.space) API.

## Installation

```bash
go get github.com/octaspace/go-sdk
```

## Quick start

```go
import sdk "github.com/octaspace/go-sdk"

client := sdk.NewClient("your-api-key")
```

## What's included

| Area | Methods |
|---|---|
| **Account** | `GetAccount`, `GetBalance`, `ValidateToken` |
| **Nodes** | `ListNodes`, `ListNodesRaw` |
| **Apps** | `ListApps`, `ListAppsRaw` |
| **Compute** | `ListMachinesForRent`, `ListMachinesForRentRaw`, `DeployMachine` |
| **Sessions** | `ListSessions`, `ListSessionsRaw`, `GetSessionInfo`, `GetSessionInfoRaw`, `GetSessionLogs`, `StopSession` |
| **VPN** | `ListVPNRelays`, `ListVPNRelaysRaw`, `ConnectVPN` |

Methods ending in `Raw` return the response body as `[]byte` — useful when you want to forward JSON downstream without deserializing it.

## Examples

Usage examples for every method are in [`example_test.go`](example_test.go). A few highlights:

**Deploy a machine**

```go
resp, err := client.DeployMachine(sdk.DeployRequest{
    App:      "app-uuid",  // from ListApps
    NodeID:   42,          // from ListMachinesForRent
    DiskSize: 50,          // GB
    Envs: map[string]string{
        "JUPYTER_TOKEN": "secret",
    },
})
// resp.UUID is the session identifier
```

**Connect to VPN and get WireGuard config**

```go
resp, err := client.ConnectVPN(7, "wireguard")

info, err := client.GetSessionInfo(resp.UUID)
fmt.Println(info.VPNConfig) // paste into wg0.conf
```

**List active sessions**

```go
sessions, err := client.ListSessions()
for _, s := range sessions {
    fmt.Printf("[%s] node=%d  ready=%v  ip=%s\n",
        s.UUID, s.NodeID, s.IsReady, s.PublicIP)
}
```

**Stop a session**

```go
err := client.StopSession("session-uuid")
```

## Balance type

Account balances are represented as `BigInt` (a thin wrapper around `*big.Int`) because OctaSpace balances are denominated in Wei and can exceed the range of `int64`.

```go
balance, err := client.GetBalance()
fmt.Printf("%s wei\n", balance.String())
```

## Running tests

```bash
go test ./...
```

Tests use `httptest.Server` — no API key or network access required.
