# Getting started

Create one client and reuse it concurrently. Every SDK method accepts a
`context.Context` and returns a typed result plus an error.

```go
ctx := context.Background()
client := octaspace.NewClient(os.Getenv("OCTA_API_KEY"))

account, err := client.Accounts.Profile(ctx)
if err != nil {
	return err
}
fmt.Println(account.Email)
```

An empty key is valid for public endpoints such as `Network.Info`. The client
omits the `Authorization` header in that case.

## Configuration

```go
client := octaspace.NewClient(apiKey,
	octaspace.WithBaseURL("https://api.octa.space"),
	octaspace.WithTimeout(15*time.Second),
	octaspace.WithMaxRetries(2),
	octaspace.WithUserAgent("my-service/1.0"),
	octaspace.WithLogger(octaspace.NewStdLogger(log.Default())),
)
```

`WithBaseURLs` accepts ordered fallback hosts. Failover happens only for
network failures; API responses, including 5xx, are returned from the first
reachable host. GET requests retry 429 and 5xx responses; POST and PATCH are
not retried, so session-creation calls are not duplicated.

For API-specific examples, start with the [resource reference](resources.md).
