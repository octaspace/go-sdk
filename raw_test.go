package octaspace

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRawMethods_ReturnExactBodyAndEncodeQuery(t *testing.T) {
	const appsPayload = `[{"uuid":"app-1","new_server_field":{"enabled":true}}]`
	const sessionsPayload = `[{"uuid":"session-1","new_server_field":true}]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization = %q", got)
		}
		switch r.URL.Path {
		case "/apps":
			_, _ = w.Write([]byte(appsPayload))
		case "/sessions":
			_, _ = w.Write([]byte(sessionsPayload))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	c := NewClient("test-key", WithBaseURL(srv.URL), WithMaxRetries(0))

	apps, err := c.Apps.ListRaw(context.Background())
	if err != nil {
		t.Fatalf("Apps.ListRaw: %v", err)
	}
	if string(apps) != appsPayload {
		t.Errorf("Apps.ListRaw = %s, want exact server body", apps)
	}
	sessions, err := c.Sessions.ListRaw(context.Background())
	if err != nil {
		t.Fatalf("Sessions.ListRaw: %v", err)
	}
	if string(sessions) != sessionsPayload {
		t.Errorf("Sessions.ListRaw = %s, want exact server body", sessions)
	}
}

func TestRawMethods_RetryAndFailover(t *testing.T) {
	attempts := 0
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts == 1 {
			http.Error(w, `{"error":"temporary"}`, http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`[{"unknown":true}]`))
	}))
	t.Cleanup(good.Close)
	dead := closedURL(t)
	c := NewClient("key", WithBaseURLs([]string{dead, good.URL}), WithMaxRetries(1))
	body, err := c.Apps.ListRaw(context.Background())
	if err != nil {
		t.Fatalf("ListRaw: %v", err)
	}
	if string(body) != `[{"unknown":true}]` || attempts != 2 {
		t.Errorf("body=%s attempts=%d", body, attempts)
	}
}

func TestRawMethods_TimeoutAndCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithTimeout(20*time.Millisecond), WithMaxRetries(0))
	if _, err := c.Apps.ListRaw(context.Background()); err == nil {
		t.Fatal("timeout error = nil")
	} else {
		var networkErr *NetworkError
		if !errors.As(err, &networkErr) {
			t.Errorf("timeout error = %T, want NetworkError", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.Apps.ListRaw(ctx); err == nil {
		t.Fatal("canceled context error = nil")
	}
}

func TestSessionGetRaw_EscapesUUID(t *testing.T) {
	var escapedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawPath != "" {
			escapedPath = r.URL.RawPath
		} else {
			escapedPath = r.RequestURI
		}
		_, _ = w.Write([]byte(`{"unknown":true}`))
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(0))
	body, err := c.Sessions.GetRaw(context.Background(), "uuid/with space")
	if err != nil {
		t.Fatalf("GetRaw: %v", err)
	}
	if string(body) != `{"unknown":true}` {
		t.Errorf("body = %s", body)
	}
	if escapedPath != "/services/uuid%2Fwith%20space/info" {
		t.Errorf("escaped path = %q", escapedPath)
	}
}
