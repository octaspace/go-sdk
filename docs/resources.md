# Resource reference

Each resource is available from a `Client`. List operations accept `nil` when
no filter is required.

## Network

`client.Network.Info(ctx)` reads public network information and returns
`map[string]any`, because its server-defined schema can change.

## Accounts

`client.Accounts.Profile(ctx)`, `Balance(ctx)`, and `GenerateWallet(ctx)` cover
account access. Balances are represented with `*big.Int`/`BigInt` in Wei.

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

`client.Services.R` is the render-worker marketplace (`/services/r`) with
multi-node requests and per-item results. `client.Services.Render` maps to the
separate render-job API (`/services/render`), whose payload is intentionally
`map[string]any` because it has no stable typed schema.

## Idle jobs

`client.IdleJobs.Find(ctx, nodeID, jobID)` reads a job and `Logs` returns its
raw log bytes.

See [machine rentals](machine-rentals.md) and [sessions](sessions.md) for the
service flows that require more context.

Field-level Swagger/live decisions are recorded in the
[contract inventory](contracts.md).
