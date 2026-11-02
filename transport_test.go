package octaspace

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewClient_DefaultBaseURLIsDevStand(t *testing.T) {
	c := NewClient("")
	if got := c.cfg.baseURLs; len(got) != 1 || got[0] != BaseURLDev {
		t.Fatalf("baseURLs = %v, want [%s]", got, BaseURLDev)
	}
}

func TestNormalizeAPIKey(t *testing.T) {
	tests := map[string]string{
		"octa_ak_abc":          "octa_ak_abc",
		"  octa_ak_abc \n":     "octa_ak_abc",
		"Bearer octa_ak_abc":   "octa_ak_abc",
		"bearer   octa_ak_abc": "octa_ak_abc",
		"BEARER octa_ak_abc":   "octa_ak_abc",
		"Bearer":               "Bearer", // no space after the scheme: not a prefix
		"":                     "",
	}
	for in, want := range tests {
		if got := normalizeAPIKey(in); got != want {
			t.Errorf("normalizeAPIKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestClient_BearerPrefixNotDoubled(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Authorization")
		jsonHandler(map[string]any{})(w, r)
	}))
	t.Cleanup(srv.Close)

	c := NewClient("Bearer octa_ak_abc", WithBaseURL(srv.URL), WithMaxRetries(0))
	if _, err := c.Network.Info(context.Background()); err != nil {
		t.Fatalf("Network.Info: %v", err)
	}
	if got != "Bearer octa_ak_abc" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer octa_ak_abc")
	}
}

func TestClient_RawRequestUsesBearer(t *testing.T) {
	var got string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Authorization")
		_, _ = w.Write([]byte("log line"))
	})
	if _, err := c.getRaw(context.Background(), "/raw"); err != nil {
		t.Fatalf("getRaw: %v", err)
	}
	if got != "Bearer test-key" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer test-key")
	}
}

func TestClient_EmptyBodyWithDestination(t *testing.T) {
	for _, code := range []int{http.StatusAccepted, http.StatusNoContent} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			c, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(code)
			})
			dst := map[string]any{"kept": true}
			if err := c.patch(context.Background(), "/hosting/nodes/7KQ4-M2XD-4/prices", map[string]int{"base": 100}, &dst); err != nil {
				t.Fatalf("patch: %v", err)
			}
			if dst["kept"] != true {
				t.Errorf("dst was modified by an empty body: %v", dst)
			}
		})
	}
}

func TestClient_ValidationAndConflictErrors(t *testing.T) {
	tests := []struct {
		code  int
		body  string
		check func(error) bool
		name  string
	}{
		{400, `{"message":"tail must be 1..1000"}`, IsValidation, "IsValidation"},
		{409, `{"message":"Session is still starting or restarting"}`, IsConflict, "IsConflict"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := newTestClient(t, statusHandler(tc.code, tc.body))
			_, err := c.Network.Info(context.Background())
			if !tc.check(err) {
				t.Fatalf("%s(%v) = false", tc.name, err)
			}
			var apiErr *APIError
			if !errors.As(err, &apiErr) || apiErr.Message == "" {
				t.Errorf("message not extracted: %v", err)
			}
		})
	}
}

func TestParseRateLimit(t *testing.T) {
	tests := []struct {
		name   string
		header http.Header
		want   RateLimit
		ok     bool
	}{
		{"all", http.Header{"Ratelimit-Limit": {"300"}, "Ratelimit-Remaining": {"299"}, "Ratelimit-Reset": {"54"}}, RateLimit{300, 299, 54}, true},
		{"partial", http.Header{"Ratelimit-Remaining": {"6"}}, RateLimit{Remaining: 6}, true},
		{"none", http.Header{}, RateLimit{}, false},
		{"malformed", http.Header{"Ratelimit-Limit": {"lots"}}, RateLimit{}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseRateLimit(tc.header)
			if ok != tc.ok || got != tc.want {
				t.Errorf("parseRateLimit = %+v, %v; want %+v, %v", got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestClient_RateLimitFromSuccessResponse(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("RateLimit-Limit", "300")
		w.Header().Set("RateLimit-Remaining", "297")
		w.Header().Set("RateLimit-Reset", "53")
		jsonHandler(map[string]any{})(w, r)
	})
	if _, ok := c.RateLimit(); ok {
		t.Fatal("RateLimit reported before any request")
	}
	if _, err := c.Network.Info(context.Background()); err != nil {
		t.Fatalf("Network.Info: %v", err)
	}
	got, ok := c.RateLimit()
	if !ok || got != (RateLimit{Limit: 300, Remaining: 297, Reset: 53}) {
		t.Errorf("RateLimit() = %+v, %v", got, ok)
	}
}

func TestClient_RateLimitOnError(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("RateLimit-Limit", "10")
		w.Header().Set("RateLimit-Remaining", "6")
		w.Header().Set("Retry-After", "288")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"message":"Session was restarted too recently, retry in 288 seconds"}`))
	})
	_, err := c.Network.Info(context.Background())
	var rlErr *RateLimitError
	if !errors.As(err, &rlErr) {
		t.Fatalf("err = %v, want *RateLimitError", err)
	}
	if rlErr.RetryAfter != 288 {
		t.Errorf("RetryAfter = %d, want 288", rlErr.RetryAfter)
	}
	if rlErr.RateLimit == nil || rlErr.RateLimit.Remaining != 6 {
		t.Errorf("RateLimit = %+v, want Remaining 6", rlErr.RateLimit)
	}
	if got, ok := c.RateLimit(); !ok || got.Limit != 10 {
		t.Errorf("client RateLimit() = %+v, %v; want Limit 10", got, ok)
	}
}

func TestClient_NoRetryWhenRetryAfterExceedsCap(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.Header().Set("Retry-After", "288")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"message":"Too many requests"}`))
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(2))
	start := time.Now()
	_, err := c.Network.Info(context.Background())
	if !IsRateLimit(err) {
		t.Fatalf("err = %v, want rate limit error", err)
	}
	if n := attempts.Load(); n != 1 {
		t.Errorf("attempts = %d, want 1 (Retry-After above the cap must not be retried)", n)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("returned after %v, want an immediate error", elapsed)
	}
}
