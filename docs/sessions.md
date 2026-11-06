# Sessions

[← Back to README](../README.md)

**On this page:** [Lifecycle](#lifecycle) · [List and get](#list-and-get) ·
[Wait until ready](#wait-until-ready) · [Connect](#connect) · [Logs](#logs) ·
[Stop](#stop) · [Session fields](#session-fields)

`client.Sessions` works with your running sessions, both compute and render.
They share the `Session` type; `Service` tells them apart
(`octaspace.ServiceCompute`, `octaspace.ServiceRender`).

## Lifecycle

```text
Compute.Start / Render.Start ──► starting ──► ready ──► Sessions.Stop
                                    ▲           │
                                    └─ Restart ─┘
```

| `Status` | Meaning |
| --- | --- |
| `SessionStarting` | The container is being set up; see `Progress` and `PullPercent` |
| `SessionReady` | The session is running and can be used |

A stopped session is no longer returned by the API.

## List and get

```go
sessions, err := client.Sessions.List(ctx) // running sessions, oldest first

session, err := client.Sessions.Get(ctx, uuid)
if octaspace.IsNotFound(err) {
	// the session has stopped, or the UUID is unknown
}
```

## Wait until ready

`WaitReady` polls a session until it is ready:

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

It checks every 3 seconds; change this with `WaitOptions.Interval`. Limit the
wait with the context.

## Connect

When a compute session is ready, these fields tell you how to reach it:

| Field | Contains |
| --- | --- |
| `SSH.Direct`, `SSH.Proxy` | SSH endpoints: `Host`, `Port` and the `User` to log in as |
| `SSH.Web` | URL of the web terminal |
| `FM` | URL of the web file browser |
| `Ports` | Each exposed container port mapped to its public port on the node |
| `URLs` | Each HTTP port mapped to its public URL |

The SSH fields are empty unless SSH was turned on when the session started,
and `FM` is empty unless the file browser was.

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

| Field | Contains |
| --- | --- |
| `Container` | The latest container output; `Tail` sets the number of lines, 1 – 1000 (200 by default). `nil` when the node cannot be reached, which differs from an empty log. |
| `ContainerTruncated` | `true` if the output was cut to its last 1 MiB |
| `System` | Platform events of the session, oldest first |

Logs are available for running compute sessions.

## Stop

```go
_, err := client.Sessions.Stop(ctx, uuid)
```

Stopping takes a few seconds, during which `Get` can still return the session;
after that it returns a `*NotFoundError`. Stopping a session twice is safe.

> [!NOTE]
> A session stopped before its minimum rent (`MinRentMinutes`) is charged as
> if it ran that long.

## Session fields

| Field | Meaning |
| --- | --- |
| `BasePrice`, `StoragePrice`, `TrafficPrice` | Prices, fixed when the session starts |
| `ChargeAmount` | What the session has cost so far |
| `Duration` | Seconds since the start |
| `StartedAt` | Unix milliseconds; `StartedTime()` returns a `time.Time` |
| `NodeHW` | The node: CPU, memory, GPUs, the rented disk in GB, network speed |

While a session is starting, its prices are zero and `NodeHW` is `nil`.
