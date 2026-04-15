# Changelog

All notable changes to this project will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

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

[0.1.0]: https://github.com/octaspace/go-sdk/releases/tag/v0.1.0
