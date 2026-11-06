# Changelog

All notable changes to this project are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and the project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

The SDK follows the current OctaSpace API. Most resources changed; see the
[resources](docs/resources.md) and [sessions](docs/sessions.md) guides.

### Added

- `client.Compute`: `List`, `ListRaw`, `Start`, `StartBatch` and `Restart`.
- `client.Render`: `List`, `ListRaw`, `Start` and `StartBatch`.
- `client.Hosting`: `Nodes`, `NodesRaw` and `SetPrices`, with the `Price` and
  `Int` helpers.
- `client.Stats`: `GPUCurrent`, `GPUHistory` and `GPULocations`.
- `Sessions.Get`, `GetRaw`, `Logs`, `Stop` and `WaitReady`, with
  `ErrSessionStopped`.
- `Network.InfoRaw`.
- `MicroUSD` money type with `Dollars()` and `String()`.
- `Bool` helper for optional `*bool` parameters.
- `ConflictError` (HTTP 409), `IsConflict` and `IsValidation`.
- `RateLimit`, available as `Client.RateLimit()` and `APIError.RateLimit`.
- `DefaultBaseURL` constant.

### Changed

- **Breaking:** the API key is sent as `Authorization: Bearer <key>`. A key
  that already has the `Bearer ` prefix is accepted.
- **Breaking:** the default base URL is `https://api-dev.octa.space`
  (`DefaultBaseURL`).
- **Breaking:** `Session` is a new model shared by compute and render sessions,
  with `Service`, `Status`, `Progress`, `PullPercent`, `NodeNumber`, prices in
  `MicroUSD`, `SSH` and `NodeHW`. `Sessions.List` takes no parameters and
  returns running sessions only. `LogsParams.Recent` is replaced by `Tail`.
- **Breaking:** `Accounts.Balance` returns `*AccountBalance` with the balance in
  `MicroUSD`.
- **Breaking:** `Network.Info` returns `*NetworkInfo` instead of
  `map[string]any`.
- **Breaking:** HTTP 400 responses are returned as `*ValidationError`.
- **Breaking:** the service constants are `ServiceCompute` and `ServiceRender`.
- A GET request is no longer retried when `Retry-After` exceeds 60 seconds; the
  `*RateLimitError` is returned at once.
- A 202 or 204 response with an empty body no longer fails to decode.

### Removed

- **Breaking:** `client.Nodes`, `client.Apps`, `client.IdleJobs`,
  `client.Services` and `Accounts.Profile`, `ProfileRaw` and `GenerateWallet`,
  with their types. Hosted nodes and their prices are in `client.Hosting`,
  sessions in `client.Sessions`.
- **Breaking:** the `BigInt`, `FlexInt64` and `FlexString` types.

## [0.5.0] - 2026-07-22

### Fixed

- Listing sessions and getting session info no longer fail when the API returns
  `container_id` as an array.

### Changed

- **Breaking:** `Session.ContainerID` and `SessionInfo.ContainerID` are
  `FlexString` instead of `string`. `FlexString` accepts a string or an array of
  strings and joins several IDs with `", "`; use `.String()` where a `string` is
  needed.

## [0.4.0] - 2026-07-22

### Fixed

- `NodesResource.Find` now decodes the CPU, memory, disk, GPU, uptime and agent
  version of a node, which were left empty.

### Changed

- **Breaking:** `NodesResource.Find` returns `*NodeDetail` instead of `*Node`.
  `NodeDetail` adds reliability, account ID and more location and memory
  fields. Node prices are available from `NodesResource.List`.

## [0.3.0] - 2026-07-22

### Added

- `MachineRentalResource.CreateSingle` for starting one machine rental.
- Raw JSON methods for apps, the account profile, nodes, sessions, machine
  rentals, VPN relays and session info.
- Guides under `docs/`.

### Fixed

- `App` keeps `start_command`.
- `SessionInfo` includes connection, lifecycle, pricing and hardware details,
  for both `config` and `vpn_config`.
- Machine rentals keep their prices, GPUs grouped by vendor and reliability.
- The generated wallet address, node memory usage and the session `account_id`
  are decoded.
- Service-create results keep a `status` of 0.
- VPN relays decode both `net_*_mbits` and `net_*_mbs` keys.

## [0.2.0] - 2026-07-21

### Added

- `client.Services.R` for render workers, with multi-node start requests and a
  result for every node.
- `ServiceCreateResponse.Results` and `ServiceCreateResult`.
- Optional organization and project IDs when creating a machine rental.

### Fixed

- `Sessions.List` accepts `rx` and `tx` as numbers or strings.
- Creating a machine rental decodes the per-node result list;
  `response.UUID` holds the first started session.

## [0.1.0] - 2026-04-15

First release.

### Added

- `NewClient` with the options `WithBaseURL`, `WithBaseURLs`, `WithTimeout`,
  `WithMaxRetries`, `WithHTTPClient`, `WithLogger` and `WithUserAgent`.
- Failover to the next base URL when a host cannot be reached.
- Retries with exponential backoff and jitter for GET requests on 429 and 5xx.
- `client.Accounts`: profile, balance and wallet generation.
- `client.Apps`: the application catalog.
- `client.Network`: network statistics.
- `client.Nodes`: list, find, ident and log downloads, price updates and reboot.
- `client.Sessions`: active and recent sessions.
- `client.Services`: machine rentals (`MR`), VPN relays (`VPN`), render
  (`Render`) and per-session info, logs and stop (`Session`).
- `client.IdleJobs`.
- Typed errors: `APIError`, `NetworkError`, `IsNotFound`, `IsRateLimit` and
  `IsAuthError`.

[Unreleased]: https://github.com/octaspace/go-sdk/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/octaspace/go-sdk/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/octaspace/go-sdk/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/octaspace/go-sdk/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/octaspace/go-sdk/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/octaspace/go-sdk/releases/tag/v0.1.0
