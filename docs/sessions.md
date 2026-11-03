# Sessions

`client.Sessions` covers the account's **running** sessions. Compute and render
sessions share the `Session` type and are told apart by `Service`
(`octaspace.ServiceCompute` / `ServiceRender`); the compute-only fields
(`SSH`, `Ports`, `URLs`, `FM`, `Envs`, `RX`/`TX`, storage and traffic prices)
stay zero for render. A session of another kind still decodes.

## Listing sessions

```go
sessions, err := client.Sessions.List(ctx) // running sessions, oldest first
```

Stopped sessions are not listed, and there is no session history in the API.

## Session operations

```go
session, err := client.Sessions.Get(ctx, uuid)                  // GET /services/{uuid}/info
logs, err := client.Sessions.Logs(ctx, uuid, &octaspace.LogsParams{Tail: 100})
_, err = client.Sessions.Stop(ctx, uuid)                         // 202: stopping
```

- `Get` of a stopped or unknown session is a `*NotFoundError`; a malformed UUID
  is a `*ValidationError`.
- `Logs` works for running compute sessions only. `Tail` is 1..1000 (0 uses the
  API default, 200). `Container` is nil when the node cannot be reached, which
  is different from an empty log; `ContainerTruncated` marks output cut to its
  last 1 MiB. System events carry `TS` in Unix milliseconds (`Time()`).
- `Stop` is asynchronous and idempotent. The session can still be reported for
  a few seconds afterwards: treat `Get` returning `NotFoundError` as stopped.
  A session stopped before its `MinRentMinutes` is charged base and storage
  for that long.

## Waiting until a session is ready

A started or restarted session is `SessionStarting` while it is configured
(`Progress`, and `PullPercent` while the image is pulled), then
`SessionReady`.

```go
ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
defer cancel()
session, err := client.Sessions.WaitReady(ctx, uuid, &octaspace.WaitOptions{
	OnProgress: func(s *octaspace.Session) { fmt.Println(s.Status, s.Progress) },
})
if errors.Is(err, octaspace.ErrSessionStopped) {
	// the session stopped while starting, e.g. its SSH setup failed
}
```

`WaitReady` polls every 3 seconds by default (`WaitOptions.Interval`); bound
the wait with the context.

## Session fields

Many fields are unknown while a session is starting: prices are zero and
`NodeHW` is nil until it is configured.

- Prices are `MicroUSD`, fixed at start; `ChargeAmount` is what was charged so
  far.
- `Duration` is in seconds, `StartedAt` in Unix milliseconds (`StartedTime()`).
- `SSH.Direct` / `SSH.Proxy` are nil and `SSH.Web` is empty unless SSH was
  requested; `SSHEndpoint.User` is the user to log in as.
- `Ports` maps a container port to its public port on the node; `URLs` maps an
  HTTP port to its public URL.
- `NodeHW.Disk` is the rented disk in GB; `NetDownMbps`/`NetUpMbps` are the
  node's network speed in whole Mbit/s (0 if unknown).

Use `client.Sessions.GetRaw(ctx, uuid)` or `ListRaw(ctx)` when the exact server
response is needed.
