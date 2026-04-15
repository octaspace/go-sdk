// Package octaspace provides a Go client library for the OctaSpace API.
//
// Basic usage:
//
//	client := octaspace.NewClient(os.Getenv("OCTA_API_KEY"))
//	account, err := client.Accounts.Profile(ctx)
//
// The client is organized into resource sub-clients that mirror the API structure:
//
//	client.Accounts         – account profile, balance, wallet generation
//	client.Nodes            – compute node listing and management
//	client.Sessions         – active session listing
//	client.Apps             – available application catalog
//	client.Network          – network information
//	client.Services.MR      – machine rental (compute)
//	client.Services.VPN     – VPN service
//	client.Services.Render  – render service
//	client.Services.Session – per-session operations (info, logs, stop)
//	client.IdleJobs         – idle job inspection
//
// All methods accept a [context.Context] as the first argument and return a
// typed value and an error. Errors are one of the types defined in errors.go
// and can be inspected with [errors.As] or the helper functions [IsNotFound],
// [IsRateLimit], and [IsAuthError].
package octaspace

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"
)

// Version is the current SDK version, embedded in the default User-Agent header.
const Version = "0.1.0"

const (
	defaultBaseURL    = "https://api.octa.space"
	defaultTimeout    = 30 * time.Second
	defaultMaxRetries = 2
)

// Retry tuning constants.
const (
	// maxBackoffShift caps the exponential backoff shift to avoid int overflow
	// on high attempt counts. Base delay is capped at 500ms * 2^6 = 32s.
	maxBackoffShift = 6
	// maxRetryAfter caps a server-supplied Retry-After value so a hostile or
	// misconfigured server cannot stall the client indefinitely.
	maxRetryAfter = 60 * time.Second
)

// Logger is the interface for SDK diagnostic logging. It requires Debugf and
// Errorf methods. The standard library [*log.Logger] does NOT satisfy this
// interface directly — use [NewStdLogger] to wrap it.
//
// Example with the standard library:
//
//	l := log.New(os.Stderr, "[octaspace] ", log.LstdFlags)
//	client := octaspace.NewClient(apiKey, octaspace.WithLogger(octaspace.NewStdLogger(l)))
type Logger interface {
	Debugf(format string, v ...any)
	Errorf(format string, v ...any)
}

// stdLogAdapter wraps a Printf-compatible logger to satisfy [Logger].
type stdLogAdapter struct {
	printf func(string, ...any)
}

func (a *stdLogAdapter) Debugf(format string, v ...any) { a.printf(format, v...) }
func (a *stdLogAdapter) Errorf(format string, v ...any) { a.printf(format, v...) }

// NewStdLogger wraps a standard library [*log.Logger] (or any type with a
// Printf method) into a [Logger] suitable for [WithLogger].
// Both Debugf and Errorf are mapped to Printf.
func NewStdLogger(l interface{ Printf(string, ...any) }) Logger {
	return &stdLogAdapter{printf: l.Printf}
}

// config holds the resolved Client configuration.
type config struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration // per-request SDK timeout applied via context
	maxRetries int
	logger     Logger
	userAgent  string
}

// Option is a functional option for configuring a [Client].
type Option func(*config)

// WithBaseURL overrides the API base URL (default: https://api.octa.space).
func WithBaseURL(baseURL string) Option {
	return func(c *config) {
		c.baseURL = strings.TrimRight(baseURL, "/")
	}
}

// WithHTTPClient replaces the default [http.Client]. The provided client is
// used as-is and never mutated by the SDK; its Timeout (if any) still applies
// in addition to the SDK's own context timeout from [WithTimeout].
func WithHTTPClient(hc *http.Client) Option {
	return func(c *config) {
		if hc != nil {
			c.httpClient = hc
		}
	}
}

// WithTimeout sets a per-request SDK timeout applied via [context.WithTimeout]
// (default: 30s). Pass 0 to disable.
//
// The timeout is independent of any [http.Client.Timeout] set via [WithHTTPClient]
// and the two compose — the shorter bound wins. Unlike a client-level Timeout,
// WithTimeout is applied per logical SDK call, so it bounds the *total* time
// including retries.
func WithTimeout(d time.Duration) Option {
	return func(c *config) { c.timeout = d }
}

// WithMaxRetries sets the maximum number of retry attempts for transient errors
// (HTTP 429, 500, 502, 503, 504) on idempotent requests (default: 2).
// Set to 0 to disable retries entirely.
func WithMaxRetries(n int) Option {
	return func(c *config) {
		if n < 0 {
			n = 0
		}
		c.maxRetries = n
	}
}

// WithLogger sets a [Logger] for request/response diagnostics.
// Requests and responses are logged at Debug level; transport errors at Error level.
func WithLogger(l Logger) Option {
	return func(c *config) { c.logger = l }
}

// WithUserAgent overrides the User-Agent header sent with every request.
func WithUserAgent(ua string) Option {
	return func(c *config) { c.userAgent = ua }
}

// Client is an authenticated HTTP client for the OctaSpace API.
// It is safe for concurrent use. Create one with [NewClient].
type Client struct {
	cfg      *config
	Accounts *AccountsResource
	Nodes    *NodesResource
	Sessions *SessionsResource
	Apps     *AppsResource
	Network  *NetworkResource
	Services *ServicesResource
	IdleJobs *IdleJobsResource
}

// NewClient creates a new OctaSpace API client authenticated with apiKey.
// Pass an empty string to use only unauthenticated public endpoints (e.g. Network.Info).
func NewClient(apiKey string, opts ...Option) *Client {
	cfg := &config{
		apiKey:     apiKey,
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{}, // no transport-level Timeout — we use context
		timeout:    defaultTimeout,
		maxRetries: defaultMaxRetries,
		userAgent:  "octaspace-go/" + Version,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	c := &Client{cfg: cfg}
	c.Accounts = &AccountsResource{c}
	c.Nodes = &NodesResource{c}
	c.Sessions = &SessionsResource{c}
	c.Apps = &AppsResource{c}
	c.Network = &NetworkResource{c}
	c.Services = &ServicesResource{
		c:      c,
		MR:     &MachineRentalResource{c},
		VPN:    &VPNResource{c},
		Render: &RenderResource{c},
	}
	c.IdleJobs = &IdleJobsResource{c}
	return c
}

// ---- internal transport ----

// timeoutCtx wraps ctx with the SDK-configured per-request timeout.
// Returns ctx unchanged (with a no-op cancel) when timeout is 0.
func (c *Client) timeoutCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.cfg.timeout > 0 {
		return context.WithTimeout(ctx, c.cfg.timeout)
	}
	return ctx, func() {}
}

// get performs a GET request with optional query parameters and JSON-decodes
// the response into dst. Pass nil params or dst to omit them.
// Retries transient failures (429, 5xx) up to maxRetries times.
func (c *Client) get(ctx context.Context, path string, params url.Values, dst any) error {
	ctx, cancel := c.timeoutCtx(ctx)
	defer cancel()
	return c.withRetry(ctx, http.MethodGet, func() error {
		return c.do(ctx, http.MethodGet, path, params, nil, dst)
	})
}

// post performs a POST request, JSON-encodes body (if non-nil), and decodes the
// response into dst (if non-nil). POST is never retried automatically.
func (c *Client) post(ctx context.Context, path string, body, dst any) error {
	ctx, cancel := c.timeoutCtx(ctx)
	defer cancel()
	return c.do(ctx, http.MethodPost, path, nil, body, dst)
}

// patch performs a PATCH request. Arguments work the same as post.
// PATCH is never retried automatically.
func (c *Client) patch(ctx context.Context, path string, body, dst any) error {
	ctx, cancel := c.timeoutCtx(ctx)
	defer cancel()
	return c.do(ctx, http.MethodPatch, path, nil, body, dst)
}

// getRaw performs a GET request and returns the raw response body bytes without
// JSON decoding. Used for binary responses (logs, ident files).
// Retries transient failures the same way [get] does.
func (c *Client) getRaw(ctx context.Context, path string) ([]byte, error) {
	ctx, cancel := c.timeoutCtx(ctx)
	defer cancel()
	var body []byte
	err := c.withRetry(ctx, http.MethodGet, func() error {
		var ierr error
		body, ierr = c.doRawOnce(ctx, path)
		return ierr
	})
	return body, err
}

// withRetry runs fn and retries it on retryable transport/API errors.
// Only idempotent methods (GET, HEAD) are retried, and only on HTTP 429/5xx.
func (c *Client) withRetry(ctx context.Context, method string, fn func() error) error {
	for attempt := 0; ; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}
		if attempt >= c.cfg.maxRetries {
			return err
		}
		var apiErr *APIError
		if !errors.As(err, &apiErr) || !isRetryable(method, apiErr.StatusCode) {
			return err
		}
		delay := retryDelay(attempt, apiErr.RetryAfter)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
}

// doRawOnce executes a single raw GET request without retry.
func (c *Client) doRawOnce(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("octaspace: build request: %w", err)
	}
	req.Header.Set("User-Agent", c.cfg.userAgent)
	req.Header.Set("Accept", "*/*")
	if c.cfg.apiKey != "" {
		req.Header.Set("Authorization", c.cfg.apiKey)
	}

	start := time.Now()
	resp, err := c.cfg.httpClient.Do(req)
	elapsed := time.Since(start)
	if err != nil {
		if c.cfg.logger != nil {
			c.cfg.logger.Errorf("octaspace: GET %s failed after %dms: %v",
				path, elapsed.Milliseconds(), err)
		}
		return nil, &NetworkError{Err: err}
	}
	defer resp.Body.Close()
	if c.cfg.logger != nil {
		c.cfg.logger.Debugf("octaspace: GET %s -> %d (%dms)",
			path, resp.StatusCode, elapsed.Milliseconds())
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("octaspace: read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, errorFromResponse(resp, body)
	}
	return body, nil
}

// do executes a single HTTP request, logging diagnostics and parsing errors.
func (c *Client) do(ctx context.Context, method, path string, params url.Values, body, dst any) error {
	var bodyReader io.Reader
	if !isNilValue(body) {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("octaspace: marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	rawURL := c.cfg.baseURL + path
	if len(params) > 0 {
		rawURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, rawURL, bodyReader)
	if err != nil {
		return fmt.Errorf("octaspace: build request: %w", err)
	}
	req.Header.Set("User-Agent", c.cfg.userAgent)
	req.Header.Set("Accept", "application/json")
	if c.cfg.apiKey != "" {
		req.Header.Set("Authorization", c.cfg.apiKey)
	}
	if bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	start := time.Now()
	resp, err := c.cfg.httpClient.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		if c.cfg.logger != nil {
			c.cfg.logger.Errorf("octaspace: %s %s failed after %dms: %v",
				method, path, elapsed.Milliseconds(), err)
		}
		return &NetworkError{Err: err}
	}
	defer resp.Body.Close()

	if c.cfg.logger != nil {
		c.cfg.logger.Debugf("octaspace: %s %s -> %d (%dms)",
			method, path, resp.StatusCode, elapsed.Milliseconds())
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("octaspace: read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return errorFromResponse(resp, respBody)
	}
	if dst == nil {
		return nil
	}
	if err := json.Unmarshal(respBody, dst); err != nil {
		return fmt.Errorf("octaspace: decode response: %w", err)
	}
	return nil
}

// isNilValue reports whether v is nil, including typed-nil pointers, maps,
// slices, channels, and function values. This prevents the "typed-nil body"
// footgun where a nil *T passed as any would marshal to "null".
func isNilValue(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Interface, reflect.Chan, reflect.Func:
		return rv.IsNil()
	}
	return false
}

// isRetryable reports whether a failed request should be retried.
// Only idempotent methods (GET, HEAD) on transient status codes are retried.
func isRetryable(method string, statusCode int) bool {
	if method != http.MethodGet && method != http.MethodHead {
		return false
	}
	switch statusCode {
	case 429, 500, 502, 503, 504:
		return true
	}
	return false
}

// retryDelay returns how long to wait before the next retry attempt.
// It honours the Retry-After header value when present (capped at maxRetryAfter),
// otherwise uses exponential backoff with jitter: 500ms, 1s, 2s, … (±25% jitter).
// Backoff is capped at attempt=maxBackoffShift to avoid int overflow.
func retryDelay(attempt, retryAfterSeconds int) time.Duration {
	if retryAfterSeconds > 0 {
		d := time.Duration(retryAfterSeconds) * time.Second
		if d > maxRetryAfter {
			d = maxRetryAfter
		}
		return d
	}
	if attempt < 0 {
		attempt = 0
	}
	if attempt > maxBackoffShift {
		attempt = maxBackoffShift
	}
	base := 500 * time.Millisecond * time.Duration(1<<attempt)
	jitter := time.Duration(rand.Int63n(int64(base / 4)))
	return base + jitter
}
