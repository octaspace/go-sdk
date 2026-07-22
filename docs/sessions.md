# Sessions

## Listing sessions

```go
active, err := client.Sessions.List(ctx, nil)

recent := true
finished, err := client.Sessions.List(ctx, &octaspace.ListSessionsParams{Recent: &recent})
```

The active and recent endpoints overlap but do not expose identical fields.
`FlexInt64` accepts numeric and quoted-string counters and timestamps.

## Session operations

```go
session := client.Services.Session(uuid)
info, err := session.Info(ctx)
logs, err := session.Logs(ctx, nil)
err = session.Stop(ctx, nil)
```

Pass `&octaspace.LogsParams{Recent: &recent}` for logs from a finished
session. `Stop` accepts an optional `StopParams` quality score.

`SessionInfo` includes identity, lifecycle, connection, pricing, hardware,
traffic, and port-mapping fields. It accepts both live `config` and Swagger
`vpn_config`; marshaling uses `config`. `PortsMatrix` accepts numeric and
string port values, and `UserEnvs` intentionally retains non-string values.

Use `session.InfoRaw(ctx)` when an exact server response is needed.
