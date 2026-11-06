# Errors, rate limits and retries

[← Back to README](../README.md)

**On this page:** [Error types](#error-types) ·
[Rate limits](#rate-limits) · [Retries](#retries)

## Error types

An error response from the API is returned as one of these types. Each wraps
`*APIError`, which holds the HTTP status, the API's message, the request ID and
the rate limit state.

| Type | Status | Helper | Typical cause |
| --- | --- | --- | --- |
| `*ValidationError` | 400, 422 | `IsValidation` | A parameter is missing or invalid |
| `*AuthenticationError` | 401 | `IsAuthError` | The API key is missing, wrong or deleted |
| `*PermissionError` | 403 | — | Access denied |
| `*NotFoundError` | 404 | `IsNotFound` | Unknown or stopped session, unknown node |
| `*ConflictError` | 409 | `IsConflict` | The session is already starting or restarting |
| `*RateLimitError` | 429 | `IsRateLimit` | Too many requests |
| `*ServerError` | 5xx | — | A temporary problem on the API side |

When the API cannot be reached at all (DNS, connection or TLS failure), the
error is a `*NetworkError`.

```go
_, err := client.Compute.Start(ctx, params)

var apiErr *octaspace.APIError
if errors.As(err, &apiErr) {
	log.Printf("status %d: %s (request %s)", apiErr.StatusCode, apiErr.Message, apiErr.RequestID)
}
```

> [!TIP]
> Include the request ID when you contact support about a failed call.

## Rate limits

| Requests | Limit per minute |
| --- | --- |
| Without an API key | 60 per IP address |
| With an API key | 300 per account |
| Starting and restarting sessions | 10 per account |

`client.RateLimit()` returns the limit, the remaining requests and the seconds
until the window resets, as of the latest response. A failed call carries the
same in `APIError.RateLimit`.

On a `*RateLimitError`, wait `RetryAfter` seconds before you try again:

```go
var rl *octaspace.RateLimitError
if errors.As(err, &rl) {
	time.Sleep(time.Duration(rl.RetryAfter) * time.Second)
}
```

## Retries

GET requests are retried automatically on 429, 500, 502, 503 and 504, with
exponential backoff and jitter. They honour `Retry-After` up to 60 seconds;
when the API asks to wait longer, the `*RateLimitError` is returned at once.

> [!IMPORTANT]
> Requests that change something — starting, restarting and stopping
> sessions, setting prices — are never retried, so that they cannot run twice.
> Decide yourself whether to repeat them after an error.

Set the number of retries with `WithMaxRetries` and the time limit of a call,
retries included, with `WithTimeout`.
