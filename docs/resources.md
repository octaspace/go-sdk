# Resources

[← Back to README](../README.md)

**On this page:** [Conventions](#conventions) · [Compute](#compute) ·
[Render](#render) · [VPN](#vpn) · [Hosting](#hosting) · [Accounts](#accounts) ·
[Network](#network) · [Stats](#stats)

Sessions have [their own guide](sessions.md).

## Conventions

| Concept | In the SDK |
| --- | --- |
| Money | `octaspace.MicroUSD`, an integer number of millionths of a US dollar. `String()` formats it as `$0.250000`; `Dollars()` returns a `float64`. |
| Node | Identified by its node number, for example `7KQ4-M2XD-4`. Case, dashes and spaces are ignored. |
| Network speed | Whole Mbit/s in `NetDownMbps` / `NetUpMbps`; `0` if not measured. |
| Optional values | Pointer fields. Set them with `octaspace.Bool`, `octaspace.Int` or `octaspace.Price`, so that `false` and `0` can be sent too. |

## Compute

Rent a machine and run your own container on it.

### List machines

```go
machines, err := client.Compute.List(ctx)
for _, m := range machines {
	fmt.Println(m.NodeNumber, m.City, m.CPUCores, m.BasePrice)
}
```

A `ComputeMachine` has its location, CPU, memory, disk, GPUs, network speed,
reliability and prices:

| Field | Meaning |
| --- | --- |
| `BasePrice` | Per hour of rent |
| `StoragePrice` | Per GB of disk per hour |
| `TrafficPrice` | Per GB of traffic |
| `MinRentMinutes` | Minimum rent, see below; `0` if none |

> [!NOTE]
> **Minimum rent.** If the node owner set a minimum rent, a session stopped
> sooner is charged as if it ran that long. It is not charged if the node
> fails.

The listing also works without an API key, but is then refreshed about once an
hour; with a key it is at most 10 seconds old.

### Apps

An app is a ready-made image with its settings. `Compute.Apps` lists the
official apps and, with an API key, your own (`Personal`):

```go
apps, err := client.Compute.Apps(ctx)
for _, app := range apps {
	if app.Name == "Ubuntu 24.04 LTS" {
		uuid, err := client.Compute.Start(ctx, &octaspace.ComputeStartParams{
			NodeNumber: "7KQ4-M2XD-4",
			App:        app.UUID,
			Image:      app.Image,
			DiskSize:   app.Extra.MinDiskSize,
		})
	}
}
```

| Field | Meaning |
| --- | --- |
| `UUID` | Pass as `App` to `Compute.Start`; `Image` is still required |
| `Ports`, `HTTPPorts` | Ports the app exposes |
| `Extra.MinDiskSize` | Disk the app needs, GB. Not enforced: a smaller `DiskSize` is accepted, and the app may run out of space |
| `StartupScript`, `Envs` | Only on your own apps |

The list also works without an API key, without your own apps.

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
their defaults, listed in the
[`ComputeStartParams` reference](https://pkg.go.dev/github.com/octaspace/go-sdk#ComputeStartParams).
Use `octaspace.Bool(false)` to turn off SSH or the file manager that an
application enables.

`Start` returns as soon as the session is created, while its container is still
being set up. Wait for it with
[`Sessions.WaitReady`](sessions.md#wait-until-ready). A rejected request
returns a `*ValidationError` with the reason.

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
_, err := client.Compute.Restart(ctx, uuid, nil) // keep the parameters
```

`Restart` recreates the container on the same node; the session keeps running.
With `nil` the session restarts with the parameters it was started with.

> [!WARNING]
> Parameters you pass **replace** the session's parameters as a whole; they
> are not merged. Include everything the session needs. For example, `Args`
> without the `Entrypoint` they belong to can make the session stop.

| Error | When |
| --- | --- |
| `*ConflictError` | The session is already starting or restarting |
| `*RateLimitError` | The session was restarted too recently; wait `RetryAfter` seconds. A session can be restarted once in 5 minutes and 6 times an hour |

## Render

Rent a render node.

```go
nodes, err := client.Render.List(ctx)

uuid, err := client.Render.Start(ctx, "7KQ4-M2XD-4")

results, err := client.Render.StartBatch(ctx, []octaspace.RenderBatchItem{
	{ID: "a", NodeNumber: "7KQ4-M2XD-4"},
	{ID: "b", NodeNumber: "3HW9-P6TR-8"},
})
```

A `RenderMachine` is a node with an NVIDIA GPU, its Blender score and
`BasePrice` per hour. The node renders for your account until you stop the
session.

## VPN

Rent a VPN on a node, billed by traffic.

```go
nodes, err := client.VPN.List(ctx)
// pick a node whose Protocols contain octaspace.VPNWireGuard
uuid, err := client.VPN.Start(ctx, octaspace.VPNStartParams{
	NodeNumber: "7KQ4-M2XD-4",
	Subkind:    octaspace.VPNWireGuard,
})
s, err := client.Sessions.WaitReady(ctx, uuid, nil)
if err == nil && s.Config != nil {
	err = os.WriteFile("octa.conf", []byte(s.Config.Text), 0o600) // import it into the client
}
```

A `VPNMachine` has its location, network speed, `TrafficPrice` per GB and
`Protocols`: the protocols that passed an end-to-end check on that node.

| `Subkind` | The session's `Config` |
| --- | --- |
| `VPNWireGuard` | `Text`: a WireGuard config file |
| `VPNAmneziaWG` | `Text`: an AmneziaWG config file |
| `VPNReality` | `Text`: a VLESS Reality link |
| `VPNHTTPProxy` | `HTTPProxy`: the proxy's `IP` and `Port` and its login (`Username`, `Password`, HTTP Basic). The proxy lets in only the IP address the `Start` request came from |

Pick a `Subkind` from the node's `Protocols`. A protocol the node did not pass
returns a `*ValidationError` (`Node's VPN did not pass the connection check`);
nothing is started or charged.

- One VPN session per node: a second one on the same node returns a
  `*ValidationError` (`You already have a VPN session on this node`).
- `Config` of a ready session is **a credential**: a WireGuard config holds
  the private key, and a proxy's has its password. Do not log it.
- VPN sessions have no logs, and a node running one stays `NodeIdle` in
  `Hosting.Nodes`.
- Stop the session with `client.Sessions.Stop`; until then it is charged for
  traffic.

The listing also works without an API key, but is then refreshed about once an
hour; with a key it is at most 10 seconds old.

## Hosting

The nodes you host, their prices and maintenance.

```go
nodes, err := client.Hosting.Nodes(ctx)
for _, n := range nodes {
	fmt.Println(n.Number, n.Status, n.Prices.Base, n.UptimePercent)
}
```

`Nodes` lists your nodes that are online, with their status (`NodeIdle`,
`NodeRented`, …), the services they are offered for, prices, hardware,
location and uptime. A node that runs only VPN sessions stays `NodeIdle`.

```go
node, err := client.Hosting.Get(ctx, "7KQ4-M2XD-4")
```

`Get` returns one of your online nodes. A node that is offline, does not exist
or is not yours returns a `*NotFoundError`.

### Set prices

```go
err := client.Hosting.SetPrices(ctx, "7KQ4-M2XD-4", octaspace.NodePrices{
	Base:           octaspace.Price(250_000), // $0.25 per hour
	MinRentMinutes: octaspace.Int(15),
})
```

Only the prices you set change.

| Field | Meaning | Range |
| --- | --- | --- |
| `Base` | Charged for every hour the node is rented | $0.0001 – $1000 |
| `Storage` | Charged per hour for each GB of disk the renter allocates | $0 – $1000 |
| `Traffic` | Per GB of traffic | $0.0001 – $1000 |
| `MinRentMinutes` | Minimum rent of a compute session | 0 – 60 minutes; `0` turns it off |

A value out of range returns a `*ValidationError`, an unknown node a
`*NotFoundError`.

### Maintenance

```go
err := client.Hosting.SetMaintenance(ctx, "7KQ4-M2XD-4", true)  // take the node off rent
err = client.Hosting.SetMaintenance(ctx, "7KQ4-M2XD-4", false)  // offer it again
```

In maintenance the node is not offered for new rentals until you take it
out; sessions already running go on and their renters are notified.
`Nodes` reports it as `Maintenance`. It works for a node that is offline
too, and repeating a call changes nothing. An unknown node returns a
`*NotFoundError`.

## Accounts

```go
b, err := client.Accounts.Balance(ctx)
fmt.Println(b.Balance) // $12.500000
```

The balance already includes what running sessions have used and becomes
negative when they use more than the account has. A recent charge or deposit
can take a moment to appear.

## Network

Platform statistics, updated every few minutes. No API key is needed.

```go
info, err := client.Network.Info(ctx)
fmt.Println(info.Nodes.Online, info.Sessions.Running, info.Time())

history, err := client.Network.History(ctx, &octaspace.NetworkHistoryParams{
	Start: time.Now().Add(-7 * 24 * time.Hour),
	Step:  6 * time.Hour,
})
for _, p := range history.Points() {
	fmt.Println(p.Time, *p.NodesOnline, p.GPUsRented)
}
```

| Method | Returns |
| --- | --- |
| `Info` | Nodes, capacity (with `GPUsRented`), sessions, users, rendered frames and VPN nodes now |
| `History` | Nodes, GPUs, TFLOPS, users and sessions over time; the last day with hourly points by default. Each value is the average over its step, so it can be fractional, and `nil` where there is none. `Step` is at least 5 minutes; a range has at most 2000 points. |

## Stats

GPU market and mining statistics. No API key is needed.

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

mining, err := client.Stats.Mining(ctx, &octaspace.MiningParams{Role: octaspace.MiningGPU, Days: 30})
for _, m := range mining.Miners {
	fmt.Println(m.Name, m.Label, m.GPUHours)
}
```

| Method | Returns |
| --- | --- |
| `GPUCurrent` | Per GPU model: the number of GPUs (not nodes), how many are rented, memory (`MemoryMB`), performance, and price percentiles per GPU-hour. A percentile is `nil` when there is no price. |
| `GPUHistory` | The same over time. `Step` is at least 5 minutes; a range has at most 2000 steps. |
| `GPULocations` | The number of GPUs by city and model |
| `Mining` | Miners, algorithms and pools of the platform's mining sessions, most used first, and the miners per day. `Role` is `MiningGPU` (default) or `MiningCPU`; `Days` is 1 – 365 (default 30). |
