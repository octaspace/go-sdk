package octaspace

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIdleJobs_Find(t *testing.T) {
	want := map[string]any{
		"node_id": float64(10),
		"job_id":  float64(42),
		"status":  "running",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/idle_jobs/10/42" {
			http.NotFound(w, r)
			return
		}
		jsonHandler(want)(w, r)
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(0))

	got, err := c.IdleJobs.Find(context.Background(), 10, 42)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if got["status"] != "running" {
		t.Errorf("status = %v, want running", got["status"])
	}
}

func TestIdleJobs_Find_Error(t *testing.T) {
	c, _ := newTestClient(t, statusHandler(404, `{"error":"not found"}`))
	_, err := c.IdleJobs.Find(context.Background(), 1, 1)
	if !IsNotFound(err) {
		t.Errorf("expected NotFoundError, got %T: %v", err, err)
	}
}

func TestIdleJobs_Logs(t *testing.T) {
	wantBody := []byte("log line 1\nlog line 2\n")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/idle_jobs/5/99/logs" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(200)
		w.Write(wantBody)
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(0))

	got, err := c.IdleJobs.Logs(context.Background(), 5, 99)
	if err != nil {
		t.Fatalf("Logs: %v", err)
	}
	if string(got) != string(wantBody) {
		t.Errorf("got %q, want %q", got, wantBody)
	}
}

func TestIdleJobs_Logs_Error(t *testing.T) {
	c, _ := newTestClient(t, statusHandler(500, `{"error":"server error"}`))
	_, err := c.IdleJobs.Logs(context.Background(), 1, 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
