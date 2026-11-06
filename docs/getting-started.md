# Client configuration

[← Back to README](../README.md)

**On this page:** [Creating a client](#creating-a-client) ·
[Options](#options) · [Context and timeouts](#context-and-timeouts) ·
[Logging](#logging) · [Several hosts](#several-hosts)

## Creating a client

`NewClient` takes your [API key](../README.md#authentication) and optional
settings. Create one client and reuse it: it is safe for concurrent use.

```go
client := octaspace.NewClient(os.Getenv("OCTA_API_KEY"),
	octaspace.WithTimeout(15*time.Second),
	octaspace.WithMaxRetries(3),
	octaspace.WithUserAgent("my-service/1.0"),
	octaspace.WithLogger(octaspace.NewStdLogger(log.Default())),
)
```

## Options

| Option | Default | Effect |
| --- | --- | --- |
| `WithTimeout(d)` | `30s` | Time limit for one method call, retries included |
| `WithMaxRetries(n)` | `2` | Retries of failed GET requests; `0` turns them off |
| `WithUserAgent(ua)` | `octaspace-go/<version>` | Value of the `User-Agent` header |
| `WithLogger(l)` | none | Logs the method, path, status and duration of each request |
| `WithHTTPClient(hc)` | tuned `http.Client` | Your own client, for example for a proxy or tracing |
| `WithBaseURL(url)` | `DefaultBaseURL` | Another API host |
| `WithBaseURLs(urls)` | — | Several API hosts, see [below](#several-hosts) |

## Context and timeouts

Every method takes a `context.Context` as its first argument. Cancelling the
context, or reaching its deadline, stops the call and any retries:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

info, err := client.Network.Info(ctx)
```

`WithTimeout` applies to every call in addition to the context; the shorter
limit wins.

## Logging

A `Logger` has two methods, `Debugf` and `Errorf`. Wrap a standard
`*log.Logger` with `NewStdLogger`, or adapt the logger you already use:

```go
type slogAdapter struct{ l *slog.Logger }

func (a slogAdapter) Debugf(f string, v ...any) { a.l.Debug(fmt.Sprintf(f, v...)) }
func (a slogAdapter) Errorf(f string, v ...any) { a.l.Error(fmt.Sprintf(f, v...)) }

client := octaspace.NewClient(apiKey, octaspace.WithLogger(slogAdapter{slog.Default()}))
```

> [!NOTE]
> The API key and request bodies are never logged.

## Several hosts

`WithBaseURLs` takes an ordered list of API hosts. A call moves to the next
host only when the current one cannot be reached (DNS, connection or TLS
failure). An HTTP error response, including 5xx, is returned as it is.
