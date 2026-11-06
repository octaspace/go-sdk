# Errors, rate limits and retries

## Error types

An error response from the API is returned as one of these types. Each wraps
`*APIError`, which has the HTTP status, the API's message, the request ID and
rate limit details.

| Type | Status | Helper |
| --- | --- | --- |
| `*ValidationError` | 400, 422 | `IsValidation` |
| `*AuthenticationError` | 401 | `IsAuthError` |
| `*PermissionError` | 403 | — |
| `*NotFoundError` | 404 | `IsNotFound` |
| `*ConflictError` | 409 | `IsConflict` |
| `*RateLimitError` | 429 | `IsRateLimit` |
| `*ServerError` | 5xx | — |

When the API cannot be reached at all (DNS, connection or TLS failure), the
error is a `*NetworkError`.

```go
_, err := client.Compute.Start(ctx, params)

var apiErr *octaspace.APIError
if errors.As(err, &apiErr) {
	log.Printf("status %d: %s (request %s)", apiErr.StatusCode, apiErr.Message, apiErr.RequestID)
}
```

Include the request ID when you contact support about a failed call.

## Rate limits

The API limits requests per minute: 60 per IP address without an API key, 300
per account with one, and 10 per account for starting and restarting sessions.

`client.RateLimit()` returns the limit, the remaining requests and the seconds
until the window resets, as of the latest response. A failed call carries the
same in `APIError.RateLimit`.

On a `*RateLimitError`, wait `RetryAfter` seconds before trying again:

```go
var rl *octaspace.RateLimitError
if errors.As(err, &rl) {
	time.Sleep(time.Duration(rl.RetryAfter) * time.Second)
}
```

## Retries

GET requests are retried automatically on 429, 500, 502, 503 and 504, with
exponential backoff and jitter, and honour `Retry-After` up to 60 seconds. When
the API asks to wait longer, the `*RateLimitError` is returned at once.

Requests that change something (starting, restarting and stopping sessions,
setting prices) are never retried, so they cannot run twice.

Set the number of retries with `WithMaxRetries` and the time limit of a call,
retries included, with `WithTimeout`.
