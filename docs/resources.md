# Resources

Each resource is a field of `Client`. Money is `octaspace.MicroUSD`, an
integer number of millionths of a US dollar: `String()` formats it as
`$0.250000`, `Dollars()` converts it to `float64`. Nodes are identified by
their number, such as `7KQ4-M2XD-4`.

Sessions are described in [their own guide](sessions.md).

## Compute

### List machines

```go
machines, err := client.Compute.List(ctx)
for _, m := range machines {
	fmt.Println(m.NodeNumber, m.CPUCores, m.BasePrice, m.NetDownMbps)
}
```

A `ComputeMachine` has its location, CPU, memory, disk, GPUs, network speed
(`NetDownMbps` / `NetUpMbps`, Mbit/s), reliability and prices: `BasePrice` per
hour, `StoragePrice` per GB of disk per hour, `TrafficPrice` per GB, and
`MinRentMinutes`, the shortest time a session is billed for.

The listing works without an API key, but is then refreshed only about once an
hour; with a key it is at most 10 seconds old.

### Start a session

```go
uuid, err := client.Compute.Start(ctx, &octaspace.ComputeStartParams{
	NodeNumber: "7KQ4-M2XD-4",
	Image:      "ubuntu:24.04",
	DiskSize:   20, // GB
	Entrypoint: "sh -c",
	Args:       `"sleep infinity"`,
	SSH:        octaspace.Bool(true),
})
```

`NodeNumber` and `Image` are required. Fields left zero are not sent and take
the defaults listed in the `ComputeStartParams` reference. Use
`octaspace.Bool(false)` to turn off SSH or the file manager that an app would
enable.

`Start` returns as soon as the session is created; the container is still being
set up. Wait for it with [`Sessions.WaitReady`](sessions.md#wait-until-ready).
A rejected request returns a `*ValidationError` with the reason.

### Start several sessions

```go
results, err := client.Compute.StartBatch(ctx, []octaspace.ComputeBatchItem{
	{ID: "a", ComputeStartParams: octaspace.ComputeStartParams{NodeNumber: "7KQ4-M2XD-4", Image: "ubuntu:24.04"}},
	{ID: "b", ComputeStartParams: octaspace.ComputeStartParams{NodeNumber: "3HW9-P6TR-8", Image: "ubuntu:24.04"}},
})
for _, r := range results {
	if r.OK() {
		fmt.Println(r.ID, "started", r.UUID)
	} else {
		fmt.Println(r.ID, "failed:", r.Reason)
	}
}
```

Each item is checked on its own: one that fails does not stop the others.

### Restart a session

```go
_, err := client.Compute.Restart(ctx, uuid, nil) // same parameters
```

`Restart` recreates the container on the same node. With `nil` it keeps the
parameters the session was started with. Parameters you pass **replace** the
old ones as a whole, so include everything the session needs: `Args` without
the `Entrypoint` they belong to, for example, can make the session stop.

A session can be restarted once in 5 minutes. An earlier attempt returns a
`*RateLimitError` whose `RetryAfter` says how long to wait; one made while the
session is still starting returns a `*ConflictError`.

## Render

```go
nodes, err := client.Render.List(ctx)

uuid, err := client.Render.Start(ctx, "7KQ4-M2XD-4")

results, err := client.Render.StartBatch(ctx, []octaspace.RenderBatchItem{
	{ID: "a", NodeNumber: "7KQ4-M2XD-4"},
	{ID: "b", NodeNumber: "3HW9-P6TR-8"},
})
```

A `RenderMachine` is a node with an NVIDIA GPU, its Blender score and
`BasePrice` per hour. The node renders for the account until the session is
stopped.

## Hosting

The nodes you host and their prices.

```go
nodes, err := client.Hosting.Nodes(ctx)
for _, n := range nodes {
	fmt.Println(n.Number, n.Status, n.Prices.Base, n.UptimePercent)
}

err = client.Hosting.SetPrices(ctx, "7KQ4-M2XD-4", octaspace.NodePrices{
	Base:           octaspace.Price(250_000), // $0.25 per hour
	MinRentMinutes: octaspace.Int(15),
})
```

`Nodes` lists the nodes that are online, with their status (`NodeIdle`,
`NodeRented`, ...), the services they are offered for, prices, hardware,
location and uptime.

`SetPrices` changes only the prices you set; zero is a valid storage price and
minimum rent. A value out of range returns a `*ValidationError`, an unknown node
a `*NotFoundError`.

## Accounts

```go
b, err := client.Accounts.Balance(ctx)
fmt.Println(b.Balance) // $12.500000
```

The balance already accounts for running sessions and is negative when they
have used more than the account has. A recent charge or deposit can take a
moment to appear.

## Network

```go
info, err := client.Network.Info(ctx)
fmt.Println(info.Nodes.Online, info.Sessions.Running, info.Time())
```

Platform statistics: nodes, capacity, sessions, users, rendered frames and VPN
relays, updated every few minutes. No API key is needed.

## Stats

GPU market statistics. No API key is needed.

```go
market, err := client.Stats.GPUCurrent(ctx, &octaspace.GPUStatsParams{
	Verified: octaspace.VerifiedYes,
})
for _, g := range market.GPUs {
	fmt.Println(g.Model, g.Available, g.Rented, g.AvailablePrice.P50)
}

history, err := client.Stats.GPUHistory(ctx, &octaspace.GPUHistoryParams{
	GPUNames: []string{"NVIDIA GeForce RTX 4090"},
	Start:    time.Now().Add(-7 * 24 * time.Hour),
	Step:     6 * time.Hour,
})
for _, p := range history["NVIDIA GeForce RTX 4090"].Points() {
	fmt.Println(p.Time, p.Rented, p.RentedPrice.P50)
}

locations, err := client.Stats.GPULocations(ctx)
```

- `GPUCurrent` gives, per GPU model, the number of GPUs (not nodes), how many are
  rented, performance and price percentiles per GPU-hour. A percentile is `nil`
  when there is no price.
- `GPUHistory` gives the same over time. `Step` is at least 5 minutes, and a
  range can have at most 2000 steps.
- `GPULocations` gives the number of GPUs by city and model.
