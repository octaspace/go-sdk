# Resource reference

Each resource is available from a `Client`. List operations accept `nil` when
no filter is required.

## Network

`client.Network.Info(ctx)` returns the public platform statistics as
`*NetworkInfo` (nodes, capacity, sessions, users, rendered frames, VPN relays)
as of the latest snapshot, taken every few minutes (`At`, `Time()`). It needs
no API key; `InfoRaw` returns the body unchanged.

## Stats

`client.Stats` is the public GPU market (no API key needed):

- `GPUCurrent(ctx, &GPUStatsParams{Verified: VerifiedYes})` → `*GPUMarket`:
  per GPU model the counts (of GPUs, not nodes), usage, TFLOPS and price
  percentiles per GPU-hour (`AvailablePrice` / `RentedPrice`, `*MicroUSD`,
  nil when there is no price).
- `GPUHistory(ctx, &GPUHistoryParams{GPUNames, Verified, Start, End, Step})` →
  `map[string]GPUHistory` per model, as the API's parallel series; `Points()`
  turns one into a value per step. `Step` is at least 5 minutes and the range
  at most 2000 points.
- `GPULocations(ctx)` → GPUs by city and model.

## Hosting

`client.Hosting` manages the nodes the account hosts:

- `Nodes(ctx)` returns the **online** nodes as `[]HostingNode` (a node that is
  down is not listed): `Status` (`NodeIdle`, `NodeRented`, ...), `Activity`
  of a running check, `Maintenance`, `Verified`, `Services` (whether the node
  is offered for each enabled service, and `Reason` if not), `Prices`,
  `System`, `Location`, `OnlineAt`, `Uptime`, `UptimePercent`.
  `System.GPUs` are `map[string]any` as the node reports them: their fields are
  not fixed by the API.
- `SetPrices(ctx, number, NodePrices{...})` changes only the prices given
  (`octaspace.Price(...)`, `octaspace.Int(...)` for the pointers); zero is a
  valid storage price or minimum rent. The API checks the ranges: an invalid
  value is a `*ValidationError`, an unknown node a `*NotFoundError`.

```go
err := client.Hosting.SetPrices(ctx, "7KQ4-M2XD-4", octaspace.NodePrices{
	Base:           octaspace.Price(250_000), // $0.25 per hour
	MinRentMinutes: octaspace.Int(15),
})
```

Some statistics and node fields arrive from the API as the string `"null"`
instead of JSON null; `Stats` and `Hosting` read them as null.

## Compute

`client.Compute` rents machines on the new API.

- `List(ctx)` returns `[]ComputeMachine` (`GET /services/compute`); `ListRaw`
  returns the body unchanged. Authentication is optional: without a key the
  listing is refreshed about hourly, with one it is at most 10 seconds old.
- Machines are identified by `NodeNumber` (e.g. `7KQ4-M2XD-4`). Prices are
  `MicroUSD` (1 = $0.000001; `String()` formats dollars): `BasePrice` per hour,
  `StoragePrice` per GB-hour, `TrafficPrice` per GB. `MinRentMinutes` is the
  shortest billed rent. `NetDownMbits`/`NetUpMbits` are Mbit/s as documented
  by the API and are not converted by the SDK.
- `Start(ctx, *ComputeStartParams)` starts one session and returns its UUID;
  a rejected request is a `*ValidationError` with the reason.
  `StartBatch(ctx, []ComputeBatchItem)` starts several and reports each in a
  `BatchResult` (`OK()`, `Reason`). Only set fields are sent; `SSH`/`FM` are
  `*bool` (`octaspace.Bool(false)` to switch off what an app enables).
- `Restart(ctx, uuid, params)` recreates the container (202). Nil params keep
  the original parameters. **Non-nil params replace them as a whole**: pass
  every parameter the session needs, or it may stop (e.g. `Args` without the
  `Entrypoint` they belong to). A restart while one is running is a
  `*ConflictError`; within the 5-minute cooldown a `*RateLimitError` whose
  `RetryAfter` says how long to wait.

Starting and restarting are never retried automatically.

```go
uuid, err := client.Compute.Start(ctx, &octaspace.ComputeStartParams{
	NodeNumber: "7KQ4-M2XD-4",
	Image:      "ubuntu:24.04",
	DiskSize:   20,
	Entrypoint: "sh -c",
	Args:       `"sleep infinity"`,
	SSH:        octaspace.Bool(true),
})
```

## Accounts

`client.Accounts.Balance(ctx)` returns `*AccountBalance`: `Balance` in
`MicroUSD` (negative while a session runs past what the account has),
`Currency` (`"USD"`) and `Decimals` (6). It is read from a database replica,
so a charge may show up a moment later. `Profile` and `GenerateWallet` belong to
the legacy API and are removed with it.

## Apps

`client.Apps.List(ctx)` returns deployment templates. `App.StartCommand`,
environment values, ports, HTTP ports, and `Extra` are preserved by the typed
model. Use `Apps.ListRaw` when forwarding server JSON unchanged.

## Nodes

`client.Nodes.List(ctx, params)` supports `Country` and `AppUUID` filters and
returns `[]Node`. `Find`, `UpdatePrices`, `Reboot`, `DownloadIdent`, and
`DownloadLogs` operate on one node. `ListRaw` offers the lossless list response.

`Find` returns a `*NodeDetail`, not a `Node`: the `GET /nodes/:id` endpoint uses
a different JSON envelope than the list endpoint (hardware under `data`, memory
as `total_memory`/`free_memory`, agent version as `data.version`, plus
`reliability`/`account_id`). Node **prices are not returned by the detail
endpoint** — read them from `List`.

## VPN

`client.Services.VPN.List(ctx, params)` lists relays; `Create` starts a VPN
session. Supported subkinds include `wg`, `openvpn`, `ss`, and `v2ray`; v2ray
uses `Protocol`. `VPNRelay` accepts both `net_*_mbits` and legacy
`net_*_mbs` response keys.

## Render

`client.Render` rents render nodes on the new API: `List(ctx)` returns
`[]RenderMachine` (nodes with an NVIDIA GPU and a Blender score, billed by
`BasePrice` per hour), `Start(ctx, nodeNumber)` and `StartBatch` work as for
compute. Render is still being migrated to the new API, so its behaviour may
change.

`client.Services.R` is the legacy render-worker marketplace (`/services/r`); it
is removed together with the legacy API.

## Idle jobs

`client.IdleJobs.Find(ctx, nodeID, jobID)` reads a job and `Logs` returns its
raw log bytes.

See [machine rentals](machine-rentals.md) and [sessions](sessions.md) for the
service flows that require more context.

Field-level Swagger/live decisions are recorded in the
[contract inventory](contracts.md).
