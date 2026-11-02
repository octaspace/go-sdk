# Migrating to the next API

The `next` branch of the SDK targets the next-generation OctaSpace API. The
API changed its endpoints, formats and access model, so this is a breaking
upgrade: most code that used the legacy SDK needs changes. The full list is in
the [change history](../CHANGELOG.md).

```bash
go get github.com/octaspace/go-sdk@next
```

## Access

- **API keys.** The API takes account keys `octa_ak_…`, issued in Cube under
  Profile → Credentials, sent as `Authorization: Bearer <key>` (the SDK adds
  the scheme). Old account tokens are rejected with `401 Invalid bearer
  token`: issue a new key.
- **Base URL.** The default is `octaspace.BaseURLDev`
  (`https://api-dev.octa.space`), the stand of the new API. The legacy
  production hosts (`api.octa.space`, `api.octa.computer`) still serve the
  old API and do not work with this SDK.
- **Anonymous use.** `Network`, `Stats`, `Compute.List` and `Render.List` work
  without a key. With a key, the machine listings are at most 10 seconds old
  instead of being refreshed about hourly.

## Resources

| Legacy SDK | Next SDK |
| --- | --- |
| `Services.MR.List` | `Compute.List` → `[]ComputeMachine` |
| `Services.MR.Create`, `CreateSingle` | `Compute.Start` (one) → UUID, `Compute.StartBatch` (several) → `[]BatchResult` |
| — | `Compute.Restart` |
| `Services.Render.List` / `Create`, `Services.R.List` / `Create` | `Render.List`, `Render.Start`, `Render.StartBatch` |
| `Sessions.List(ctx, params)` | `Sessions.List(ctx)`: running sessions only |
| `Services.Session(uuid).Info` / `InfoRaw` | `Sessions.Get(ctx, uuid)` / `GetRaw` |
| `Services.Session(uuid).Logs(ctx, params)` | `Sessions.Logs(ctx, uuid, &LogsParams{Tail})` |
| `Services.Session(uuid).Stop(ctx, params)` | `Sessions.Stop(ctx, uuid)` |
| — | `Sessions.WaitReady` |
| `Nodes.List` | `Hosting.Nodes`: the account's online nodes |
| `Nodes.UpdatePrices(ctx, id, map)` | `Hosting.SetPrices(ctx, number, NodePrices{...})` |
| `Accounts.Balance` → `*big.Int` (Wei) | `Accounts.Balance` → `*AccountBalance` (`MicroUSD`) |
| `Network.Info` → `map[string]any` | `Network.Info` → `*NetworkInfo` |
| — | `Stats.GPUCurrent`, `GPUHistory`, `GPULocations` |

### Removed

These have no counterpart in the public next API and are gone from the SDK:

- `Accounts.Profile` / `ProfileRaw` / `GenerateWallet` (account profile and
  wallet);
- `Apps` (the application catalog; an app is still passed to `Compute.Start`
  by UUID);
- `Nodes.Find`, `DownloadIdent`, `DownloadLogs`, `Reboot` (node details by ID,
  ident backup, node logs, reboot);
- `Services.VPN` (VPN relays and sessions);
- `IdleJobs`;
- session history (`recent`) and the session `score` on stop;
- the helper types `BigInt`, `FlexInt64` and `FlexString`.

## Data model

- **Nodes are identified by `NodeNumber`**, a string such as `7KQ4-M2XD-4`
  (case, dashes and spaces are ignored), not by an integer ID.
- **Money is `MicroUSD`** (`int64`, 1 = $0.000001) everywhere: prices,
  charges, the balance. `String()` prints dollars, `Dollars()` converts.
  Ether, Wei and OCTA amounts are gone.
- **Prices** are `BasePrice` per hour, `StoragePrice` per GB of disk per hour
  and `TrafficPrice` per GB. `MinRentMinutes` is the shortest billed rent.
- **Sessions** have `Status` (`starting` / `ready`) instead of `IsReady`, plus
  `Progress` and `PullPercent`. Compute and render sessions share `Session`,
  told apart by `Service`.
- **Network speed** is `NetDownMbps` / `NetUpMbps` (API fields
  `net_down_mbps` / `net_up_mbps`): whole Mbit/s, 0 if not measured. The legacy
  `net_*_mbits` fields carried bytes per second; drop the old `÷125 000`
  conversions (`NetDownMbps()` / `NetUpMbps()` helpers).
- **Time** fields keep the API's units: `Session.StartedAt` and log `TS` are
  Unix milliseconds, `NetworkInfo.At` and stats timestamps are Unix seconds.
  Use the `Time()` / `StartedTime()` helpers.

## Behaviour to know

- **Starting is asynchronous.** `Compute.Start` returns a UUID while the
  session is still `starting`; use `Sessions.WaitReady`. A session that
  stops while starting (e.g. failed SSH setup) makes `WaitReady` return
  `ErrSessionStopped`.
- **`Compute.Restart` replaces parameters as a whole.** Nil params (or params
  with every field zero) restart with the original ones; any other params must
  include everything the session needs, or it may stop.
- **Stopping is asynchronous too.** A stopped session may still be reported for
  a few seconds; treat `NotFoundError` from `Get` as stopped.
- **Validation errors are HTTP 400** and come as `*ValidationError`; a restart
  while one is running is a `*ConflictError` (409).
- **Rate limits** are reported in `APIError.RateLimit` and
  `Client.RateLimit()`. Starting and restarting services has its own, stricter
  limit, and a session can be restarted once in 5 minutes: wait for
  `APIError.RetryAfter`.
- **Only set fields are sent** in start and price requests; the API rejects
  unknown fields. Optional booleans and prices are pointers (`octaspace.Bool`,
  `octaspace.Price`, `octaspace.Int`), so `false` and `0` can be sent.
