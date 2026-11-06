# Client configuration

`NewClient` takes the API key and optional settings:

```go
client := octaspace.NewClient(os.Getenv("OCTA_API_KEY"),
	octaspace.WithTimeout(15*time.Second),
	octaspace.WithMaxRetries(3),
	octaspace.WithUserAgent("my-service/1.0"),
	octaspace.WithLogger(octaspace.NewStdLogger(log.Default())),
)
```

| Option | Default | Effect |
| --- | --- | --- |
| `WithTimeout(d)` | 30s | Limit for one method call, retries included |
| `WithMaxRetries(n)` | 2 | Retries of failed GET requests; 0 turns them off |
| `WithUserAgent(ua)` | `octaspace-go/<version>` | `User-Agent` header |
| `WithLogger(l)` | none | Logs the method, path, status and duration of each request |
| `WithHTTPClient(hc)` | tuned `http.Client` | Your own client, e.g. for a proxy or tracing |
| `WithBaseURL(url)` | OctaSpace API | Another API host |
| `WithBaseURLs(urls)` | — | Several hosts, tried in order when one is unreachable |

## Context

Every method takes a `context.Context` as its first argument. Cancelling it, or
reaching its deadline, stops the call and any retries:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

info, err := client.Network.Info(ctx)
```

## Logging

`Logger` has `Debugf` and `Errorf` methods. Wrap a standard `*log.Logger` with
`NewStdLogger`, or adapt the logger you already use:

```go
type slogAdapter struct{ l *slog.Logger }

func (a slogAdapter) Debugf(f string, v ...any) { a.l.Debug(fmt.Sprintf(f, v...)) }
func (a slogAdapter) Errorf(f string, v ...any) { a.l.Error(fmt.Sprintf(f, v...)) }

client := octaspace.NewClient(apiKey, octaspace.WithLogger(slogAdapter{slog.Default()}))
```

The API key is never logged.

## Several hosts

With `WithBaseURLs`, a call moves to the next host only when the current one
cannot be reached (DNS, connection or TLS failure). An HTTP error response,
including 5xx, is returned as it is.
