# Errors and reliability

All HTTP failures are typed. Use `errors.As` or the convenience helpers:

```go
if octaspace.IsValidation(err) { /* 400, 422 */ }
if octaspace.IsAuthError(err) { /* 401 */ }
if octaspace.IsNotFound(err) { /* 404 */ }
if octaspace.IsConflict(err) { /* 409 */ }
if octaspace.IsRateLimit(err) { /* 429 */ }
```

`*APIError` exposes the HTTP status, request ID, retry-after seconds, the
server message, and the rate limit state (`RateLimit`, see below). Subtypes
include `ValidationError` (400/422, e.g. `Invalid session uuid`),
`AuthenticationError`, `PermissionError`, `NotFoundError`, `ConflictError`
(409, e.g. a session that is still starting or restarting), `RateLimitError`,
and `ServerError`.
`NetworkError` wraps DNS, TLS, connection, and similar transport failures.

## Rate limits

Every response carries `RateLimit-Limit`, `RateLimit-Remaining` and
`RateLimit-Reset` (seconds). `client.RateLimit()` returns the state of the
latest response; a failed call carries its own in `APIError.RateLimit`. The
API keeps separate windows: 60 requests/min per IP without a key, 300 per
account with one, and 10 per account for starting or restarting services.

When a call is refused with 429, wait for `APIError.RetryAfter` seconds rather
than `RateLimit.Reset`: a per-session cooldown (one restart per 5 minutes) can
demand a longer wait than the window reset.

## Retries

GET requests retry HTTP 429/500/502/503/504 with bounded exponential backoff
and jitter, honouring `Retry-After` up to 60 seconds; a longer `Retry-After`
returns the `RateLimitError` at once instead of retrying a call that would be
refused again. POST and PATCH are never retried or failed over automatically to
avoid duplicate changes. Configure retry count with `WithMaxRetries`; configure
the per-logical-call deadline with `WithTimeout`.
