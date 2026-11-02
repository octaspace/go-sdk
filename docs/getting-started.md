# Getting started

Create one client and reuse it concurrently. Every SDK method accepts a
`context.Context` and returns a typed result plus an error.

```go
ctx := context.Background()
client := octaspace.NewClient(os.Getenv("OCTA_API_KEY"))

balance, err := client.Accounts.Balance(ctx)
if err != nil {
	return err
}
fmt.Println(balance.Balance) // e.g. $2.175268
```

An empty key is valid for public endpoints such as `Network.Info`. The client
omits the `Authorization` header in that case.

## Authentication

The API takes an account API key (`octa_ak_…`), issued in Cube under
Profile → Credentials. The client sends it as `Authorization: Bearer <key>`; a
key pasted together with its `Bearer ` prefix works too. Old account tokens of
the legacy API are rejected with `401 Invalid bearer token`: issue a new key.

## Configuration

```go
client := octaspace.NewClient(apiKey,
	octaspace.WithBaseURL(octaspace.BaseURLDev), // the default
	octaspace.WithTimeout(15*time.Second),
	octaspace.WithMaxRetries(2),
	octaspace.WithUserAgent("my-service/1.0"),
	octaspace.WithLogger(octaspace.NewStdLogger(log.Default())),
)
```

The default base URL is `octaspace.BaseURLDev` (`https://api-dev.octa.space`),
the test stand of the new API; the legacy production hosts still serve the old,
incompatible API.

`WithBaseURLs` accepts ordered fallback hosts. Failover happens only for
network failures; API responses, including 5xx, are returned from the first
reachable host. GET requests retry 429 and 5xx responses; POST and PATCH are
not retried, so session-creation calls are not duplicated.

For API-specific examples, start with the [resource reference](resources.md).
