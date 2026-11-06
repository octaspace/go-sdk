# Raw JSON responses

[← Back to README](../README.md)

The typed methods are the recommended way to use the API. When you need the
response exactly as the API sent it, for example to store it or to read a field
the SDK does not have yet, use the `…Raw` variant:

```go
body, err := client.Compute.ListRaw(ctx)
if err != nil {
	return err
}
os.Stdout.Write(body)
```

| Method | Typed counterpart |
| --- | --- |
| `Compute.ListRaw` | `Compute.List` |
| `Render.ListRaw` | `Render.List` |
| `Sessions.ListRaw` | `Sessions.List` |
| `Sessions.GetRaw` | `Sessions.Get` |
| `Hosting.NodesRaw` | `Hosting.Nodes` |
| `Network.InfoRaw` | `Network.Info` |

Raw methods authenticate, retry and return errors the same way as the typed
ones.
