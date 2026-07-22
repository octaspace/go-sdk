# Errors and reliability

All HTTP failures are typed. Use `errors.As` or the convenience helpers:

```go
if octaspace.IsAuthError(err) { /* 401 */ }
if octaspace.IsNotFound(err) { /* 404 */ }
if octaspace.IsRateLimit(err) { /* 429 */ }
```

`*APIError` exposes the HTTP status, request ID, retry-after seconds, and the
server message. Subtypes include `AuthenticationError`, `PermissionError`,
`NotFoundError`, `ValidationError`, `RateLimitError`, and `ServerError`.
`NetworkError` wraps DNS, TLS, connection, and similar transport failures.

GET requests retry HTTP 429/500/502/503/504 with bounded exponential backoff
and jitter. POST and PATCH are never retried or failed over automatically to
avoid duplicate changes. Configure retry count with `WithMaxRetries`; configure
the per-logical-call deadline with `WithTimeout`.
