# Changelog

All notable changes to this project will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

Every resource now calls the current OctaSpace API endpoints; resources the API
no longer has are removed.

### Changed

- **Breaking:** the API key is sent as `Authorization: Bearer <key>` (RFC 6750),
  as the new API requires; it used to be sent bare. A key passed with its
  `Bearer ` prefix is accepted and not doubled.
- **Breaking:** the default base URL is `BaseURLDev`
  (`https://api-dev.octa.space`).
- **Breaking:** `client.Services.Render` (an untyped `map[string]any` wrapper of
  `/services/render`) is replaced by the typed `client.Render`.
- **Breaking:** sessions move to the new API. `client.Sessions` gains `Get`,
  `GetRaw`, `Logs` and `Stop` taking the session UUID, and
  `client.Services.Session(uuid)` (`SessionProxy`) is removed. `Session` is the
  new model shared by compute and render sessions (`Service`, `Status`,
  `Progress`, `PullPercent`, `NodeNumber`, `MicroUSD` prices, `SSH`,
  `NodeHW`); `SessionInfo`, `SessionPrices`, `SessionNodeLocation`,
  `ListSessionsParams` and `StopParams` are removed. `List` no longer takes
  params and `LogsParams.Recent` is replaced by `Tail`: the API keeps no session
  history. `SessionLogs.Container` is a `*string` (nil when the node cannot be
  reached). `Stop` returns the session UUID.
- **Breaking:** `Accounts.Balance` returns `*AccountBalance` (`Balance` in
  `MicroUSD`, `Currency`, `Decimals`) instead of a `*big.Int` in Wei, as the
  new API reports it.
- **Breaking:** `Network.Info` returns the typed `*NetworkInfo` instead of
  `map[string]any`.
- **Breaking:** the legacy service constants `ServiceMR` and `ServiceVPN` are
  replaced by `ServiceCompute` and `ServiceRender`.
- **Breaking:** HTTP 400 responses are returned as `*ValidationError` (was a
  bare `*APIError`); the new API reports validation errors with 400.
- GET retries no longer wait out a `Retry-After` above the 60-second cap: the
  `RateLimitError` is returned at once, since the retry would be refused again.
- A 202/204 response with an empty body no longer fails JSON decoding.

### Added

- `client.Compute` for the new compute API: `List`/`ListRaw`
  (`ComputeMachine`), `Start`, `StartBatch` (`ComputeBatchItem`, `BatchResult`)
  and `Restart` (`ComputeRestartParams`: params with any field set replace the
  session's parameters as a whole; nil or all-zero params keep them). Its doc
  lists the default each left-out field falls back to, e.g. `DiskSize` → 1 GB,
  not the session's current size.
- `client.Render` for the new render API: `List`/`ListRaw` (`RenderMachine`),
  `Start`, `StartBatch` (`RenderBatchItem`).
- `Sessions.WaitReady` polls a session until it is ready, reporting progress
  through `WaitOptions.OnProgress`; it returns `ErrSessionStopped` if the
  session stops meanwhile.
- Opt-in live compute integration tests (`OCTA_RUN_COMPUTE_INTEGRATION`)
  replace the legacy machine-rental ones.
- Opt-in live hosting test (`OCTA_RUN_HOSTING_INTEGRATION`, `OCTA_HOSTING_NODE`):
  changes a node's traffic price, waits until `GET /hosting/nodes` shows it
  (a 204 from the PATCH alone does not prove that), and restores it.
- `client.Hosting`: `Nodes`/`NodesRaw` (`HostingNode`) and `SetPrices`
  (`NodePrices`, with the `Price` and `Int` helpers) for the account's nodes.
- `client.Stats`: `GPUCurrent`, `GPUHistory` (with `GPUHistory.Points`) and
  `GPULocations` for the public GPU market.
- `Network.InfoRaw`.
- `Stats` and `Hosting` read values sent as the string `"null"` as null.
- Network speed of machines and session nodes is `NetDownMbps` / `NetUpMbps`
  (`net_down_mbps` / `net_up_mbps`): whole Mbit/s, measured by the API. The
  legacy `net_*_mbits` fields in bytes per second and their conversion helpers
  are gone.
- `MicroUSD` money type (`Dollars()`, `String()`), `MachineGPU`, `Reliability`
  and the `Bool` helper for optional `*bool` parameters.
- `MachineGPU` tolerates optional integers sent as the string `"null"`, as
  some nodes (e.g. Intel Arc) report them.
- `ConflictError` (HTTP 409) and `IsConflict`; `IsValidation`.
- `RateLimit` (`RateLimit-Limit` / `-Remaining` / `-Reset`), exposed as
  `APIError.RateLimit` and `Client.RateLimit()` for the latest response.
- `BaseURLDev` constant.

### Removed

- **Breaking:** resources without a counterpart in the API:
  `client.Nodes` (list, find, ident, logs, reboot, price updates by ID),
  `client.Apps`, `client.IdleJobs`, `client.Services` (`MR`, `R`, `VPN`) and
  `Accounts.Profile` / `ProfileRaw` / `GenerateWallet`, with their types.
  Hosted nodes and their prices are in `client.Hosting`.
- **Breaking:** the helper types `BigInt`, `FlexInt64` and `FlexString`, which
  only the legacy models used.

---

## [0.5.0] - 2026-07-22

### Fixed

- `Session.ContainerID` and `SessionInfo.ContainerID` now decode compute (MR)
  sessions correctly. The live API returns `container_id` as a JSON array for
  compute sessions (absent entirely for VPN sessions), but the field was typed
  as a bare `string`; any account with an active compute rental hit a decode
  error on the *entire* `/sessions` list and `/services/:uuid/info`, not just
  that one field. Verified against a live account with an active session.

### Changed

- **Breaking:** `Session.ContainerID` and `SessionInfo.ContainerID` are now
  `FlexString` instead of `string`. `FlexString` behaves like a normal string
  (`.String()`, comparable to string literals) but accepts both the legacy
  bare-string and the current array-of-strings shape, joining multiple
  container IDs with `", "`. Callers holding these fields in an explicitly
  `string`-typed variable need `.String()` or an explicit conversion.

---

## [0.4.0] - 2026-07-22

### Fixed

- `NodesResource.Find` now decodes the real `GET /nodes/:id` response. That
  endpoint uses a different JSON envelope than `GET /nodes` (system block nested
  under `data`, memory as `total_memory`/`free_memory`, agent version as
  `data.version`), so decoding it into `Node` silently zeroed CPU, RAM, disk,
  GPU, uptime, and agent-version fields. Verified against the live API (node 65).

### Changed

- **Breaking:** `NodesResource.Find` now returns `*NodeDetail` instead of
  `*Node`. `NodeDetail` faithfully models the detail envelope and adds
  `reliability`, `account_id`, and richer `location`/memory fields. Node prices
  are not part of this endpoint — use `NodesResource.List` for pricing.

---

## [0.3.0] - 2026-07-22

### Added

- `MachineRentalResource.CreateSingle` for the documented object-form
  machine-rental request, without changing the existing batch-array `Create`.
- Scoped lossless JSON methods for apps, account profile, nodes, sessions,
  machine rentals, VPN relays, and per-session info.
- Documentation guides under `docs/` for setup, resources, machine rentals,
  sessions, raw JSON, reliability, and testing.

### Fixed

- `App` now retains `start_command` while decoding application templates.
- `SessionInfo` models connection, lifecycle, pricing, hardware, and both
  `config` and `vpn_config` response keys.
- Machine-rental prices in Wei, vendor-grouped GPUs, and reliability fields are
  retained by typed decoding.
- Wallet-generation `address`, node memory usage, and live session `account_id`
  are retained by typed decoding.
- Service-create results retain the presence-aware `status` field, including 0.
- VPN relay decoding accepts both `net_*_mbits` and legacy `net_*_mbs` keys.

## 0.2.0 - 2026-07-21

### Added

- `client.Services.R` for the distinct `/services/r` render-worker marketplace,
  including typed multi-node create requests and per-item results.
- `ServiceCreateResponse.Results` and `ServiceCreateResult` for service APIs
  that return a result for every requested node.
- Optional organization and project identifiers on machine-rental create
  requests.

### Fixed

- `Sessions.List` now accepts numeric as well as quoted-string `rx` and `tx`
  counters from recent-session responses.
- Machine-rental creation now decodes the production per-item array response;
  the first successful UUID remains available through `response.UUID` for
  single-node callers.
- The smoke command no longer prints scalar account values or account fields;
  it reports only types, counts, and field names.
- Updated the reflection kind spelling to satisfy the configured Go linter.

## [0.1.0] — 2026-04-15

Initial public release.

### Added

**Client**
- `NewClient(apiKey, ...Option)` — authenticated client with functional options
- `WithBaseURL` / `WithBaseURLs` — single or multi-URL configuration
- `WithTimeout` — per-request timeout applied via `context.WithTimeout`
- `WithMaxRetries` — exponential backoff + jitter on 429/5xx (GET/HEAD only)
- `WithHTTPClient` — bring your own `*http.Client`
- `WithLogger` / `NewStdLogger` — structured diagnostic logging
- `WithUserAgent` — custom `User-Agent` header
- Multi-URL failover: automatically switches to the next base URL on network-level errors (connection refused, DNS failure, TLS error); HTTP errors are never failed over
- Connection pooling: tuned `http.Transport` with `MaxIdleConnsPerHost=10`

**API coverage**
- `client.Accounts` — profile, balance, wallet generation
- `client.Apps` — application catalog (handles both array and string-encoded `ports`/`http_ports`)
- `client.Network` — network statistics
- `client.Nodes` — list, find, download ident/logs, update prices, reboot
- `client.Sessions` — list active and recent sessions
- `client.Services.MR` — machine rental list and create
- `client.Services.VPN` — VPN relay list and create (supports `wg`, `openvpn`, `ss`, `v2ray` + `Extra` map for forward-compatible fields)
- `client.Services.Render` — render job list and create
- `client.Services.Session` — per-session info, logs (active and recent), stop
- `client.IdleJobs` — idle job inspection

**Types**
- `FlexInt64` — JSON int-or-quoted-string unmarshaler (used for `Duration`, `StartedAt`, etc.)
- `BigInt` — `*big.Int` wrapper for Wei-denominated values; handles null, bare number, and quoted string
- `GPU` — named type shared between `Node.System.GPUs` and `MachineRental.GPUs`
- `SSHEndpoint` — named type for `Session.SSHDirect` / `Session.SSHProxy`
- Service kind constants: `ServiceMR`, `ServiceVPN`, `ServiceRender`

**Error handling**
- `APIError` — HTTP status, message, `Retry-After`
- `NetworkError` — transport-level failures (wraps underlying error)
- `IsNotFound`, `IsRateLimit`, `IsAuthError` helpers
- Error messages truncated at 512 characters to prevent HTML pages polluting logs

**Other**
- `go test -race` clean on Go 1.20–1.23
- `golangci-lint` clean (errcheck, govet, staticcheck, nilerr, misspell, errorlint, …)
- GitHub Actions CI: test matrix Go 1.20–1.23 × ubuntu/macos, lint, live smoke (opt-in)
- `cmd/smoke` — standalone smoke tool with `--json` output and per-check skip/pass/fail reporting
- godoc `Example*` functions for key types and options

[0.2.0]: https://github.com/octaspace/go-sdk/releases/tag/v0.2.0
[0.3.0]: https://github.com/octaspace/go-sdk/releases/tag/v0.3.0
