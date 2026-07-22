# API contract inventory

This inventory was refreshed on 2026-07-22 from the public Swagger document,
sanitized live key/type probes, Cube normalizers, and the sibling JavaScript
SDK. Representative, non-secret payloads live in `testdata/contracts`.

Legend: **typed** means the Go model exposes the field; **raw** means exact
server bytes remain available even when the field is intentionally not typed.

## Applications

| JSON field | Swagger | Live | Cube/other SDK | Go decision |
| --- | --- | --- | --- | --- |
| `uuid`, `name`, `image`, `category`, `description`, `personal` | yes | yes | yes | typed |
| `envs` | object | string and non-string values | template defaults | `map[string]any` |
| `ports`, `http_ports` | arrays | arrays; older captures used JSON-encoded strings | template defaults | tolerant decoder |
| `start_command` | yes | yes | deploy default | typed and regression-tested |
| `extra` | object | yes | `min_disk_size` | `map[string]any` |
| `entrypoint` | no | absent | Cube can carry a user request value | not added to `App`; no API evidence |

## Sessions and session info

| JSON field/group | Swagger | Live | Cube/JS SDK | Go decision |
| --- | --- | --- | --- | --- |
| `uuid`, `service`, `node_id`, `duration`, `is_ready`, `started_at` | yes | yes | yes | typed |
| `started_at` representation | integer milliseconds | numeric in active, recent, and `/info` probes | numeric | `FlexInt64`; ISO strings intentionally rejected |
| `account_id` | no | active and `/info` | tolerated | typed after live evidence |
| `kind`, `ip`, `country_iso`, `progress` | partial/no | service-dependent | yes | typed |
| `status`, `state` | no | absent in probes | tolerant reader only | not added without API evidence |
| `ports_matrix` | integer values | numeric values | Cube has also seen strings | `map[string]FlexInt64` |
| `user_envs` | object | object | string/number/boolean | `map[string]any` |
| `config` / `vpn_config` | Swagger: `vpn_config` | live services use `config` | both | accept both; marshal `config` |
| SSH, URLs, node hardware, prices | yes/partial | service-dependent | yes | named typed models |
| camelCase aliases | no | not observed | Cube accepts internally | not accepted by SDK without wire evidence |

## Machine-rental prices

The similarly named price families are not interchangeable.

| Family | Observed meaning | Go fields |
| --- | --- | --- |
| `base`, `storage`, `traffic`, `total_price_wei` | live integer Wei values | `BigInt` |
| `base_ether`, `storage_ether`, `traffic_ether`, `total_price_ether` | live whole-token decimal values | `float64` |
| `base_usd`, `storage_usd`, `traffic_usd` | live USD components scaled by 10,000 | `int` |
| `total_price_usd` | live hourly USD decimal | `float64` |
| `base_price`, `storage_price`, `traffic_price`, `total_price` | Swagger-only legacy integer names; absent from live probe | separate `BigInt` fields |

Live `gpu` is grouped by `nvidia`, `amd`, `intel`, and `undefined`; `gpus` is
the flat form. Both remain typed. Live reliability is an object, while Swagger
still describes a number; `MachineReliability` accepts both and stores the
numeric form in `LegacyScore`.

## VPN relays

Swagger uses `net_down_mbs`/`net_up_mbs`; live uses
`net_down_mbits`/`net_up_mbits`. `VPNRelay` accepts both spellings and both
integer/fractional JSON numbers while preserving the existing `int64` API.
Typed marshal uses the live `*_mbits` names.

## Remaining resource audit

| Resource | Swagger/live result | Go decision |
| --- | --- | --- |
| Account profile | seven documented/live fields agree | typed; `BigInt` balance |
| Wallet generation | Swagger returns `{address}` rather than a full account | source-compatible `Account.Address` added |
| Nodes | live wrapper has `id/system/state/ip/location/uptime/prices/vrf_dt/osn`; Swagger describes inner node fields | live typed shape retained; raw escape hatch; missing memory usage fields added |
| Recent sessions | reduced field set with numeric counters/timestamps | merged `Session` model with tolerant numerics |
| Logs | `container:string`, `system:[{ts,msg}]` in Swagger and live | typed `SessionLogs` |
| Create response | object 201 and production array per-item forms | separate `CreateSingle`; tolerant array decoder with presence-aware status |

## Raw JSON boundary

Typed models are stable and normalized. Resource-scoped raw methods preserve
unknown fields and exact JSON. Tests cover exact bytes, auth, structured query
encoding, retry, failover, timeout, cancellation, API errors, and UUID path
escaping. No unrestricted arbitrary-path method is exported.
