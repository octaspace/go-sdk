# Sessions

`client.Sessions` works with the account's running sessions, both compute and
render. They share the `Session` type; `Service` tells them apart
(`octaspace.ServiceCompute`, `octaspace.ServiceRender`).

## List and get

```go
sessions, err := client.Sessions.List(ctx) // running sessions, oldest first

session, err := client.Sessions.Get(ctx, uuid)
if octaspace.IsNotFound(err) {
	// the session has stopped, or the UUID is unknown
}
```

Only running sessions are returned; a stopped session is no longer available.

## Wait until ready

A session is `SessionStarting` while its container is set up, then
`SessionReady`. `WaitReady` polls it until it is ready:

```go
ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
defer cancel()

session, err := client.Sessions.WaitReady(ctx, uuid, &octaspace.WaitOptions{
	OnProgress: func(s *octaspace.Session) {
		fmt.Println(s.Status, s.Progress, s.PullPercent)
	},
})
if errors.Is(err, octaspace.ErrSessionStopped) {
	// the session stopped while starting, for example because SSH could not be set up
}
```

It checks every 3 seconds (`WaitOptions.Interval`). Limit the wait with the
context.

## Connect

Once a session is ready:

- `SSH.Direct` and `SSH.Proxy` are the SSH endpoints (`Host`, `Port`, `User`),
  and `SSH.Web` is the web terminal. They are empty unless SSH was requested.
- `Ports` maps each exposed container port to its public port on the node.
- `URLs` maps each HTTP port to its public URL, and `FM` is the web file
  manager URL if it was started.

## Logs

```go
logs, err := client.Sessions.Logs(ctx, uuid, &octaspace.LogsParams{Tail: 100})
if logs.Container != nil {
	fmt.Print(*logs.Container)
}
for _, e := range logs.System {
	fmt.Println(e.Time(), e.Msg)
}
```

Logs are available for running compute sessions. `Tail` is the number of
container output lines, 1 to 1000 (200 by default). `Container` is `nil` when
the node cannot be reached, which differs from an empty log;
`ContainerTruncated` reports that the output was cut to its last 1 MiB.

## Stop

```go
_, err := client.Sessions.Stop(ctx, uuid)
```

Stopping takes a few seconds, during which `Get` may still return the session;
after that it returns a `*NotFoundError`. Stopping a session twice is safe.
A session stopped before its `MinRentMinutes` is charged for that long.

## Session fields

- Prices (`BasePrice`, `StoragePrice`, `TrafficPrice`) are fixed when the
  session starts; `ChargeAmount` is what it has cost so far.
- `Duration` is in seconds. `StartedAt` is in Unix milliseconds; use
  `StartedTime()` for a `time.Time`.
- `NodeHW` describes the node: CPU, memory, GPUs, the rented disk in GB and the
  network speed in Mbit/s.
- While a session is starting, its prices are zero and `NodeHW` is `nil`.
