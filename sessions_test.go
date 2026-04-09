package sdk

import (
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListSessions(t *testing.T) {
	want := []Session{
		{UUID: "sess-1", Service: "mr", IsReady: true, NodeID: 10},
		{UUID: "sess-2", Service: "vpn", IsReady: false},
	}
	c, cleanup := withServer(t, jsonHandler(http.StatusOK, want))
	defer cleanup()

	got, err := c.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
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

func TestListSessionsError(t *testing.T) {
	c, cleanup := withServer(t, rawHandler(http.StatusUnauthorized, `{}`))
	defer cleanup()

	_, err := c.ListSessions()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListSessionsRaw(t *testing.T) {
	body := `[{"uuid":"sess-1"}]`
	c, cleanup := withServer(t, rawHandler(http.StatusOK, body))
	defer cleanup()

	got, err := c.ListSessionsRaw()
	if err != nil {
		t.Fatalf("ListSessionsRaw: %v", err)
	}
	if string(got) != body {
		t.Fatalf("body = %q, want %q", got, body)
	}
}

func TestGetSessionInfo(t *testing.T) {
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
		jsonHandler(http.StatusOK, want).ServeHTTP(w, r)
	}))
	defer srv.Close()

	orig := BaseURL
	BaseURL = srv.URL
	defer func() { BaseURL = orig }()

	c := NewClient("key")
	c.http = srv.Client()

	got, err := c.GetSessionInfo("sess-abc")
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if got.VPNConfig != want.VPNConfig {
		t.Errorf("VPNConfig = %q, want %q", got.VPNConfig, want.VPNConfig)
	}
	if got.TX != want.TX {
		t.Errorf("TX = %d, want %d", got.TX, want.TX)
	}
}

func TestGetSessionInfoError(t *testing.T) {
	c, cleanup := withServer(t, rawHandler(http.StatusNotFound, `not found`))
	defer cleanup()

	_, err := c.GetSessionInfo("missing")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetSessionInfoRaw(t *testing.T) {
	body := `{"config":"wg0"}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	orig := BaseURL
	BaseURL = srv.URL
	defer func() { BaseURL = orig }()

	c := NewClient("key")
	c.http = srv.Client()

	got, err := c.GetSessionInfoRaw("some-uuid")
	if err != nil {
		t.Fatalf("GetSessionInfoRaw: %v", err)
	}
	if string(got) != body {
		t.Fatalf("body = %q, want %q", got, body)
	}
}

func TestGetSessionLogs(t *testing.T) {
	want := SessionLogs{
		Container: "container log line",
		System: []struct {
			TS  int64  `json:"ts"`
			Msg string `json:"msg"`
		}{
			{TS: 1700000000, Msg: "started"},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/services/sess-logs/logs" {
			http.NotFound(w, r)
			return
		}
		jsonHandler(http.StatusOK, want).ServeHTTP(w, r)
	}))
	defer srv.Close()

	orig := BaseURL
	BaseURL = srv.URL
	defer func() { BaseURL = orig }()

	c := NewClient("key")
	c.http = srv.Client()

	got, err := c.GetSessionLogs("sess-logs")
	if err != nil {
		t.Fatalf("GetSessionLogs: %v", err)
	}
	if got.Container != want.Container {
		t.Errorf("Container = %q, want %q", got.Container, want.Container)
	}
	if len(got.System) != 1 || got.System[0].Msg != "started" {
		t.Errorf("System logs unexpected: %+v", got.System)
	}
}

func TestGetSessionLogsError(t *testing.T) {
	c, cleanup := withServer(t, rawHandler(http.StatusNotFound, `not found`))
	defer cleanup()

	_, err := c.GetSessionLogs("missing")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestStopSession(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/services/sess-stop/stop" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	orig := BaseURL
	BaseURL = srv.URL
	defer func() { BaseURL = orig }()

	c := NewClient("key")
	c.http = srv.Client()

	if err := c.StopSession("sess-stop"); err != nil {
		t.Fatalf("StopSession: %v", err)
	}
}

func TestStopSessionError(t *testing.T) {
	c, cleanup := withServer(t, rawHandler(http.StatusInternalServerError, `err`))
	defer cleanup()

	if err := c.StopSession("any"); err == nil {
		t.Fatal("expected error, got nil")
	}
}
