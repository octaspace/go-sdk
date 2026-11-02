# Lossless raw JSON

Typed methods are the stable programming interface. Their models deliberately
normalize inconsistent fields, so a typed decode followed by `json.Marshal`
is not a lossless copy of a server response.

Use resource-scoped raw methods for JSON passthrough, diagnostics, or when the
server has added a field before the SDK models it:

```go
apps, err := client.Apps.ListRaw(ctx)
nodes, err := client.Nodes.ListRaw(ctx, params)
sessions, err := client.Sessions.ListRaw(ctx, params)
machines, err := client.Services.MR.ListRaw(ctx, params)
relays, err := client.Services.VPN.ListRaw(ctx, params)
info, err := client.Sessions.GetRaw(ctx, uuid)
account, err := client.Accounts.ProfileRaw(ctx)
```

Raw methods use the same authentication, timeout, retry, failover, error
mapping, URL escaping, and structured query encoding as their typed GET
counterparts. They return the exact response bytes after a successful HTTP
status. There is intentionally no unrestricted `RawGet` method.
