package octaspace

import (
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSessions_List(t *testing.T) {
	want := []Session{
		{UUID: "sess-1", Service: "mr", IsReady: true, NodeID: 10},
		{UUID: "sess-2", Service: "vpn", IsReady: false},
	}
	c, _ := newTestClient(t, jsonHandler(want))
	got, err := c.Sessions.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i, s := range got {
		if s.UUID != want[i].UUID {
			t.Errorf("[%d] UUID = %q, want %q", i, s.UUID, want[i].UUID)
		}
		if s.IsReady != want[i].IsReady {
			t.Errorf("[%d] IsReady = %v, want %v", i, s.IsReady, want[i].IsReady)
		}
	}
}

func TestSessions_List_RecentParam(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		jsonHandler([]Session{})(w, r)
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(0))

	recent := true
	c.Sessions.List(context.Background(), &ListSessionsParams{Recent: &recent})
	if gotQuery != "recent=true" {
		t.Errorf("query = %q, want recent=true", gotQuery)
	}
}

func TestSessions_List_Error(t *testing.T) {
	c, _ := newTestClient(t, statusHandler(401, `{"error":"unauthorized"}`))
	_, err := c.Sessions.List(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSessionProxy_Info(t *testing.T) {
	want := SessionInfo{
		VPNConfig:    "wg-config-data",
		TX:           1024,
		RX:           2048,
		ChargeAmount: BigInt{Int: big.NewInt(100)},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/services/sess-abc/info" {
			http.NotFound(w, r)
			return
		}
		jsonHandler(want)(w, r)
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(0))

	got, err := c.Services.Session("sess-abc").Info(context.Background())
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if got.VPNConfig != want.VPNConfig {
		t.Errorf("VPNConfig = %q, want %q", got.VPNConfig, want.VPNConfig)
	}
	if got.TX != want.TX {
		t.Errorf("TX = %d, want %d", got.TX, want.TX)
	}
	if got.ChargeAmount.Int.Cmp(want.ChargeAmount.Int) != 0 {
		t.Errorf("ChargeAmount = %s, want %s", got.ChargeAmount.Int, want.ChargeAmount.Int)
	}
}

func TestSessionProxy_Info_Error(t *testing.T) {
	c, _ := newTestClient(t, statusHandler(404, `{"error":"not found"}`))
	_, err := c.Services.Session("missing").Info(context.Background())
	if !IsNotFound(err) {
		t.Errorf("expected NotFoundError, got %T: %v", err, err)
	}
}

func TestSessionProxy_Logs(t *testing.T) {
	want := SessionLogs{
		Container: "container log line",
		System: []struct {
			TS  int64  `json:"ts"`
			Msg string `json:"msg"`
		}{{TS: 1700000000, Msg: "started"}},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/services/sess-logs/logs" {
			http.NotFound(w, r)
			return
		}
		jsonHandler(want)(w, r)
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(0))

	got, err := c.Services.Session("sess-logs").Logs(context.Background(), nil)
	if err != nil {
		t.Fatalf("Logs: %v", err)
	}
	if got.Container != want.Container {
		t.Errorf("Container = %q, want %q", got.Container, want.Container)
	}
	if len(got.System) != 1 || got.System[0].Msg != "started" {
		t.Errorf("System logs = %+v", got.System)
	}
}

func TestSessionProxy_Stop_UsesPost(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(0))

	if err := c.Services.Session("sess-stop").Stop(context.Background(), nil); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/services/sess-stop/stop" {
		t.Errorf("path = %q, want /services/sess-stop/stop", gotPath)
	}
}

func TestSessionProxy_Stop_WithScore(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = readBody(r)
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(0))

	score := 5
	c.Services.Session("uuid").Stop(context.Background(), &StopParams{Score: &score})
	if !containsAt(string(gotBody), `"score":5`) {
		t.Errorf("body %q does not contain score", gotBody)
	}
}

func TestSessionProxy_Stop_Error(t *testing.T) {
	c, _ := newTestClient(t, statusHandler(500, `{"error":"server error"}`))
	if err := c.Services.Session("any").Stop(context.Background(), nil); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSessionProxy_UUIDEncoding(t *testing.T) {
	var gotRawPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// RawPath is set when percent-encoded characters differ from the decoded path.
		if r.URL.RawPath != "" {
			gotRawPath = r.URL.RawPath
		} else {
			gotRawPath = r.URL.Path
		}
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(0))

	// Slashes inside the UUID should be percent-encoded (%2F), not treated as
	// path separators that would split the route.
	c.Services.Session("uuid/with/slashes").Stop(context.Background(), nil)
	if containsAt(gotRawPath, "/uuid/with/slashes/") {
		t.Errorf("RawPath %q has unencoded slashes inside UUID segment", gotRawPath)
	}
}

func TestSessionProxy_Logs_RecentParam(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		jsonHandler(SessionLogs{})(w, r)
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(0))

	recent := true
	_, err := c.Services.Session("sess-abc").Logs(context.Background(), &LogsParams{Recent: &recent})
	if err != nil {
		t.Fatalf("Logs: %v", err)
	}
	if gotQuery != "recent=true" {
		t.Errorf("query = %q, want recent=true", gotQuery)
	}
}

func TestSessionProxy_Logs_NilParams(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		jsonHandler(SessionLogs{})(w, r)
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(0))

	_, err := c.Services.Session("sess-abc").Logs(context.Background(), nil)
	if err != nil {
		t.Fatalf("Logs: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty (active session)", gotQuery)
	}
}
