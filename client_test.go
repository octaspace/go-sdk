package octaspace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/http/httptest"
	"testing"
	"time"
)

// newTestClient creates a Client pointed at a test server and returns both.
// Retries are disabled by default; override with WithMaxRetries if needed.
func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient("test-key", WithBaseURL(srv.URL), WithMaxRetries(0)), srv
}

// jsonHandler returns a handler that writes v as JSON with HTTP 200.
func jsonHandler(v any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(v)
	}
}

// statusHandler returns a handler that writes the given status code and body.
func statusHandler(code int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		w.Write([]byte(body))
	}
}

func TestNewClient_ResourcesInitialized(t *testing.T) {
	c := NewClient("key")
	checks := map[string]bool{
		"Accounts":        c.Accounts != nil,
		"Nodes":           c.Nodes != nil,
		"Sessions":        c.Sessions != nil,
		"Apps":            c.Apps != nil,
		"Network":         c.Network != nil,
		"Services":        c.Services != nil,
		"Services.MR":     c.Services.MR != nil,
		"Services.VPN":    c.Services.VPN != nil,
		"Services.Render": c.Services.Render != nil,
		"IdleJobs":        c.IdleJobs != nil,
	}
	for name, ok := range checks {
		if !ok {
			t.Errorf("%s is nil", name)
		}
	}
}

func TestClient_AuthorizationHeader(t *testing.T) {
	var got string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Authorization")
		jsonHandler(map[string]any{})(w, r)
	})
	c.Network.Info(context.Background())
	if got != "test-key" {
		t.Errorf("Authorization = %q, want %q", got, "test-key")
	}
}

func TestClient_NoAuthorizationHeaderWhenAPIKeyEmpty(t *testing.T) {
	var hasAuth bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, hasAuth = r.Header["Authorization"]
		jsonHandler(map[string]any{})(w, r)
	}))
	t.Cleanup(srv.Close)

	c := NewClient("", WithBaseURL(srv.URL), WithMaxRetries(0))
	if _, err := c.Network.Info(context.Background()); err != nil {
		t.Fatalf("Network.Info: %v", err)
	}
	if hasAuth {
		t.Fatal("Authorization header is present for empty API key")
	}
}

func TestClient_UserAgentHeader(t *testing.T) {
	var got string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("User-Agent")
		jsonHandler(map[string]any{})(w, r)
	})
	c.Network.Info(context.Background())
	if got == "" {
		t.Error("User-Agent header is empty")
	}
}

func TestClient_WithUserAgent(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("User-Agent")
		jsonHandler(map[string]any{})(w, r)
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithUserAgent("my-app/1.0"))
	c.Network.Info(context.Background())
	if got != "my-app/1.0" {
		t.Errorf("User-Agent = %q, want %q", got, "my-app/1.0")
	}
}

func TestClient_WithTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		jsonHandler(map[string]any{})(w, r)
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithTimeout(30*time.Millisecond))
	_, err := c.Network.Info(context.Background())
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	var netErr *NetworkError
	if !errors.As(err, &netErr) {
		t.Errorf("expected *NetworkError, got %T: %v", err, err)
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		jsonHandler(map[string]any{})(w, r)
	}))
	t.Cleanup(srv.Close)
	ctx, cancel := context.WithCancel(context.Background())
	c := NewClient("key", WithBaseURL(srv.URL))
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	_, err := c.Network.Info(ctx)
	if err == nil {
		t.Fatal("expected cancellation error, got nil")
	}
}

func TestClient_APIError401(t *testing.T) {
	c, _ := newTestClient(t, statusHandler(401, `{"error":"unauthorized"}`))
	_, err := c.Accounts.Profile(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsAuthError(err) {
		t.Errorf("expected AuthenticationError, got %T: %v", err, err)
	}
	var authErr *AuthenticationError
	errors.As(err, &authErr)
	if authErr.StatusCode != 401 {
		t.Errorf("StatusCode = %d, want 401", authErr.StatusCode)
	}
}

func TestClient_APIError404(t *testing.T) {
	c, _ := newTestClient(t, statusHandler(404, `{"error":"not found"}`))
	_, err := c.Nodes.Find(context.Background(), 999)
	if !IsNotFound(err) {
		t.Errorf("expected NotFoundError, got %T: %v", err, err)
	}
}

func TestClient_APIError429(t *testing.T) {
	c, _ := newTestClient(t, statusHandler(429, `{"error":"rate limit"}`))
	_, err := c.Apps.List(context.Background())
	if !IsRateLimit(err) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}
}

func TestClient_APIError500(t *testing.T) {
	c, _ := newTestClient(t, statusHandler(500, `{"error":"internal server error"}`))
	_, err := c.Apps.List(context.Background())
	var srvErr *ServerError
	if !errors.As(err, &srvErr) {
		t.Errorf("expected *ServerError, got %T: %v", err, err)
	}
}

func TestClient_RequestIDInError(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "req-abc-123")
		w.WriteHeader(500)
		w.Write([]byte(`{}`))
	})
	_, err := c.Apps.List(context.Background())
	var srvErr *ServerError
	if !errors.As(err, &srvErr) {
		t.Fatalf("expected *ServerError, got %T", err)
	}
	if srvErr.RequestID != "req-abc-123" {
		t.Errorf("RequestID = %q, want %q", srvErr.RequestID, "req-abc-123")
	}
}

func TestClient_RetryOnTransientGET(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(503)
			w.Write([]byte(`{"error":"unavailable"}`))
			return
		}
		jsonHandler([]App{})(w, r)
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(2))
	_, err := c.Apps.List(context.Background())
	if err != nil {
		t.Errorf("unexpected error after retries: %v", err)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestClient_NoRetryOnPOST(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(503)
		w.Write([]byte(`{"error":"unavailable"}`))
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(3))
	c.Accounts.GenerateWallet(context.Background())
	if attempts != 1 {
		t.Errorf("POST should not be retried; attempts = %d, want 1", attempts)
	}
}

func TestClient_RetryAfterHeader(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(429)
			w.Write([]byte(`{"error":"rate limit"}`))
			return
		}
		jsonHandler([]App{})(w, r)
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(1))
	_, err := c.Apps.List(context.Background())
	if err != nil {
		t.Errorf("unexpected error after retry: %v", err)
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
}

func TestClient_WithHTTPClient(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		jsonHandler(map[string]any{})(w, r)
	}))
	t.Cleanup(srv.Close)
	custom := &http.Client{Timeout: 10 * time.Second}
	c := NewClient("key", WithBaseURL(srv.URL), WithHTTPClient(custom))
	c.Network.Info(context.Background())
	if gotUA == "" {
		t.Error("User-Agent should still be set when using custom http.Client")
	}
}

func TestClient_WithLogger(t *testing.T) {
	var debugCalled bool
	logger := &testLogger{onDebug: func() { debugCalled = true }}
	c, _ := newTestClient(t, jsonHandler(map[string]any{}))
	// Re-create with logger
	srv := httptest.NewServer(jsonHandler(map[string]any{}))
	t.Cleanup(srv.Close)
	c2 := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(0), WithLogger(logger))
	c2.Network.Info(context.Background())
	_ = c
	if !debugCalled {
		t.Error("logger.Debugf should have been called")
	}
}

// testLogger implements Logger for testing.
type testLogger struct {
	onDebug func()
}

func (l *testLogger) Debugf(format string, v ...any) {
	if l.onDebug != nil {
		l.onDebug()
	}
}
func (l *testLogger) Errorf(format string, v ...any) {}

func TestClient_RetryNonRetryableStatus(t *testing.T) {
	// A 404 on GET must NOT be retried even when maxRetries > 0.
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(404)
		w.Write([]byte(`{"error":"not found"}`))
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(3))
	_, err := c.Nodes.Find(context.Background(), 1)
	if !IsNotFound(err) {
		t.Errorf("expected NotFoundError, got %T", err)
	}
	if attempts != 1 {
		t.Errorf("404 should not be retried; attempts = %d, want 1", attempts)
	}
}

func TestClient_RetryAfterPositive(t *testing.T) {
	// Retry-After: 1 means honour the header value (≥1 second).
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(429)
			w.Write([]byte(`{"error":"rate limit"}`))
			return
		}
		jsonHandler([]App{})(w, r)
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(1))
	start := time.Now()
	_, err := c.Apps.List(context.Background())
	elapsed := time.Since(start)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
	// Should have waited at least ~1 second.
	if elapsed < 900*time.Millisecond {
		t.Errorf("Retry-After=1 should delay ~1s, got %v", elapsed)
	}
}

func TestClient_DecodeError(t *testing.T) {
	// If the server returns 200 but not valid JSON, do() should return a decode error.
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("not json at all"))
	})
	_, err := c.Apps.List(context.Background())
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
}

func TestNetworkError_ErrorString(t *testing.T) {
	inner := &net.OpError{Op: "dial", Err: errors.New("refused")}
	netErr := &NetworkError{Err: inner}
	s := netErr.Error()
	if !containsAt(s, "network error") {
		t.Errorf("NetworkError.Error() = %q, want to contain 'network error'", s)
	}
}

func TestClient_ContentTypeOnPOST(t *testing.T) {
	var gotCT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		jsonHandler(MachineRentalCreateResponse{UUID: "u"})(w, r)
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(0))
	// Create sends a non-nil body, so Content-Type must be set.
	c.Services.MR.Create(context.Background(), &MachineRentalCreateParams{
		NodeID: 1, DiskSize: 10, Image: "alpine",
	})
	if gotCT != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotCT)
	}
}

func TestNewStdLogger(t *testing.T) {
	var logged string
	// testPrintfer is a minimal Printf-compatible type.
	tp := &testPrintfer{fn: func(format string, v ...any) {
		logged = fmt.Sprintf(format, v...)
	}}
	logger := NewStdLogger(tp)
	logger.Debugf("hello %s", "world")
	if logged != "hello world" {
		t.Errorf("Debugf: logged = %q, want %q", logged, "hello world")
	}
	logger.Errorf("err %d", 42)
	if logged != "err 42" {
		t.Errorf("Errorf: logged = %q, want %q", logged, "err 42")
	}
}

type testPrintfer struct {
	fn func(string, ...any)
}

func (p *testPrintfer) Printf(format string, v ...any) { p.fn(format, v...) }

func TestNewStdLogger_WithRealLogLogger(t *testing.T) {
	// Verify that *log.Logger satisfies the interface accepted by NewStdLogger.
	l := log.New(io.Discard, "", 0)
	logger := NewStdLogger(l)
	// Should not panic.
	logger.Debugf("test %s", "debug")
	logger.Errorf("test %s", "error")
}

// closedURL returns the URL of a just-closed server — guaranteed connection refused.
func closedURL(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	u := srv.URL
	srv.Close()
	return u
}

func TestClient_Failover_SkipsUnreachableHost(t *testing.T) {
	// First URL is closed (connection refused); second is live and returns data.
	good := httptest.NewServer(jsonHandler([]App{{UUID: "app-1"}}))
	t.Cleanup(good.Close)
	dead := closedURL(t)

	c := NewClient("key",
		WithBaseURLs([]string{dead, good.URL}),
		WithMaxRetries(0),
		WithTimeout(5*time.Second),
	)

	apps, err := c.Apps.List(context.Background())
	if err != nil {
		t.Fatalf("expected failover success, got: %v", err)
	}
	if len(apps) != 1 || apps[0].UUID != "app-1" {
		t.Errorf("unexpected apps: %v", apps)
	}
}

func TestClient_Failover_DoesNotFailoverOn500(t *testing.T) {
	// HTTP 500 is a server error, not a network failure — must NOT failover.
	calls := 0
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
	}))
	t.Cleanup(bad.Close)
	good := httptest.NewServer(jsonHandler([]App{{UUID: "should-not-reach"}}))
	t.Cleanup(good.Close)

	c := NewClient("key",
		WithBaseURLs([]string{bad.URL, good.URL}),
		WithMaxRetries(0),
	)

	_, err := c.Apps.List(context.Background())
	if err == nil {
		t.Fatal("expected error on 500, got nil")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 500 {
		t.Errorf("expected APIError 500, got: %v", err)
	}
	if calls != 1 {
		t.Errorf("bad server called %d times, want 1 (no failover on 500)", calls)
	}
}

func TestClient_Failover_AllHostsDown(t *testing.T) {
	dead1 := closedURL(t)
	dead2 := closedURL(t)

	c := NewClient("key",
		WithBaseURLs([]string{dead1, dead2}),
		WithMaxRetries(0),
		WithTimeout(5*time.Second),
	)

	_, err := c.Apps.List(context.Background())
	if err == nil {
		t.Fatal("expected error when all hosts are down, got nil")
	}
	// Error must mention unreachable hosts.
	if !containsAt(err.Error(), "unreachable") {
		t.Errorf("error %q does not mention 'unreachable'", err.Error())
	}
}

func TestWithBaseURLs_FiltersEmpty(t *testing.T) {
	good := httptest.NewServer(jsonHandler([]App{{UUID: "ok"}}))
	t.Cleanup(good.Close)

	// Mix of empty strings and the real URL.
	c := NewClient("key",
		WithBaseURLs([]string{"", good.URL, "   ", "  "}),
		WithMaxRetries(0),
	)
	// WithBaseURLs trims trailing slash but empty after trim are skipped.
	// "   " trims to "   " (no slash trimming), but is not empty — however
	// the test verifies the real URL still works when mixed with empties.
	apps, err := c.Apps.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(apps) == 0 {
		t.Error("expected at least one app")
	}
}

func TestClient_ConnectionReuse(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(map[string]any{"nodes": map[string]any{"count": 1}}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(0))
	ctx := context.Background()

	// First request — establishes connection.
	if _, err := c.Network.Info(ctx); err != nil {
		t.Fatalf("first request: %v", err)
	}

	// Second request — should reuse the existing connection.
	reused := false
	trace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			reused = info.Reused
		},
	}
	ctx2 := httptrace.WithClientTrace(ctx, trace)
	// Issue a raw request through the same transport used by the client.
	req, _ := http.NewRequestWithContext(ctx2, http.MethodGet, srv.URL+"/network", nil)
	resp, err := c.cfg.httpClient.Do(req)
	if err != nil {
		t.Fatalf("second request: %v", err)
	}
	resp.Body.Close()

	if !reused {
		t.Error("expected connection to be reused on second request")
	}
}
