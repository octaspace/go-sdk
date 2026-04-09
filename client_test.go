package sdk

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestClient returns a Client pointed at the given test server.
func newTestClient(server *httptest.Server) *Client {
	c := NewClient("test-api-key")
	c.http = server.Client()
	// Override BaseURL by patching the client's transport — instead we patch
	// the helper methods via a thin wrapper. Since BaseURL is a package-level
	// constant we intercept at the transport level by replacing c.http with the
	// test server's client and temporarily swapping BaseURL.
	return c
}

// withServer starts a handler as a test server and temporarily points the
// package BaseURL at it. The caller receives a ready *Client.
func withServer(t *testing.T, handler http.Handler) (*Client, func()) {
	t.Helper()
	srv := httptest.NewServer(handler)
	orig := BaseURL
	BaseURL = srv.URL
	c := NewClient("test-api-key")
	c.http = srv.Client()
	return c, func() {
		srv.Close()
		BaseURL = orig
	}
}

// jsonHandler responds with statusCode and marshals body as JSON.
func jsonHandler(statusCode int, body any) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(body)
	})
}

// rawHandler responds with statusCode and the raw string body.
func rawHandler(statusCode int, body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(body))
	})
}

// TestNewClient verifies basic client construction.
func TestNewClient(t *testing.T) {
	c := NewClient("my-key")
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
	if c.apiKey != "my-key" {
		t.Fatalf("apiKey = %q, want %q", c.apiKey, "my-key")
	}
	if c.http == nil {
		t.Fatal("http client is nil")
	}
}

// TestGetAuthHeader verifies that the authorization header is sent.
func TestGetAuthHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()

	orig := BaseURL
	BaseURL = srv.URL
	defer func() { BaseURL = orig }()

	c := NewClient("secret-key")
	c.http = srv.Client()

	var dst map[string]any
	_ = c.get("/test", &dst)

	if gotAuth != "secret-key" {
		t.Fatalf("authorization header = %q, want %q", gotAuth, "secret-key")
	}
}

// TestGetAPIError verifies that non-200 responses return an error.
func TestGetAPIError(t *testing.T) {
	c, cleanup := withServer(t, rawHandler(http.StatusUnauthorized, `{"error":"unauthorized"}`))
	defer cleanup()

	var dst map[string]any
	err := c.get("/anything", &dst)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Fatalf("error %q does not contain status code", err.Error())
	}
}

// TestGetRawAPIError verifies getRaw also surfaces non-200 errors.
func TestGetRawAPIError(t *testing.T) {
	c, cleanup := withServer(t, rawHandler(http.StatusForbidden, `forbidden`))
	defer cleanup()

	_, err := c.getRaw("/anything")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Fatalf("error %q does not contain 403", err.Error())
	}
}

// TestPostAuthHeader verifies the post method sends authorization and Content-Type.
func TestPostAuthHeader(t *testing.T) {
	var gotAuth, gotCT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("authorization")
		gotCT = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()

	orig := BaseURL
	BaseURL = srv.URL
	defer func() { BaseURL = orig }()

	c := NewClient("secret-key")
	c.http = srv.Client()

	var dst map[string]any
	_ = c.post("/test", map[string]string{"k": "v"}, &dst)

	if gotAuth != "secret-key" {
		t.Fatalf("authorization = %q, want secret-key", gotAuth)
	}
	if gotCT != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", gotCT)
	}
}

// TestPostAPIError verifies that 4xx responses return an error.
func TestPostAPIError(t *testing.T) {
	c, cleanup := withServer(t, rawHandler(http.StatusBadRequest, `{"error":"bad request"}`))
	defer cleanup()

	var dst map[string]any
	err := c.post("/anything", nil, &dst)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Fatalf("error %q does not contain 400", err.Error())
	}
}
