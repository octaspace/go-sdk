# Lossless raw JSON

Typed methods are the stable programming interface. Their models deliberately
normalize inconsistent fields, so a typed decode followed by `json.Marshal`
is not a lossless copy of a server response.

Use resource-scoped raw methods for JSON passthrough, diagnostics, or when the
server has added a field before the SDK models it:

```go
machines, err := client.Compute.ListRaw(ctx)
renderNodes, err := client.Render.ListRaw(ctx)
sessions, err := client.Sessions.ListRaw(ctx)
info, err := client.Sessions.GetRaw(ctx, uuid)
nodes, err := client.Hosting.NodesRaw(ctx)
network, err := client.Network.InfoRaw(ctx)
```

Raw bodies are returned as the server sent them, including values the API
sends as the string `"null"` (see [resources](resources.md#hosting)).

Raw methods use the same authentication, timeout, retry, failover, error
mapping, URL escaping, and structured query encoding as their typed GET
counterparts. They return the exact response bytes after a successful HTTP
status. There is intentionally no unrestricted `RawGet` method.
