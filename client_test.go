package octaspace

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
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
