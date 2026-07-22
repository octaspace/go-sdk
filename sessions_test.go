package octaspace

import (
	"context"
	"encoding/json"
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

func TestSessions_List_AcceptsNumericAndQuotedCounters(t *testing.T) {
	c, _ := newTestClient(t, jsonHandler([]map[string]any{
		{"uuid": "numeric", "rx": 1024, "tx": 2048},
		{"uuid": "quoted", "rx": "4096", "tx": "8192"},
	}))
	got, err := c.Sessions.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if int64(got[0].RX) != 1024 || int64(got[0].TX) != 2048 {
		t.Errorf("numeric counters = %d/%d, want 1024/2048", got[0].RX, got[0].TX)
	}
	if int64(got[1].RX) != 4096 || int64(got[1].TX) != 8192 {
		t.Errorf("quoted counters = %d/%d, want 4096/8192", got[1].RX, got[1].TX)
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
	if got.ChargeAmount.Cmp(want.ChargeAmount.Int) != 0 {
		t.Errorf("ChargeAmount = %s, want %s", got.ChargeAmount.Int, want.ChargeAmount.Int)
	}
}

func TestSessionInfo_UnmarshalFullPayloadAndConfigAlias(t *testing.T) {
	var info SessionInfo
	payload := []byte(`{
		"uuid":"session-1","service":"mr","kind":"base","app_name":"Jupyter","app_logo":"logo",
		"node_id":7,"progress":"ready","is_ready":true,"duration":"3600","started_at":1712700000,
		"public_ip":"1.2.3.4","ip":"10.0.0.2","container_id":"container","country":"Germany","country_iso":"DE","city":"Berlin",
		"ports_matrix":{"22":"10022"},"ssh_direct":{"host":"1.2.3.4","port":10022},"ssh_proxy":{"host":"proxy","port":22},"ssh_web":"https://ssh.example",
		"urls":{"8080":"https://example"},"user_envs":{"AUTH":false},"vpn_config":"vpn-data","tx":"1024","rx":512,
		"charge_amount":"5000000000000000","termination_reason":"3",
		"prices":{"base_usd":3,"currency_usd":true,"market_price":1234.5},
		"node_hw":{"cpu":"Ryzen","cpu_cores":8,"gpu":[{"model":"RTX","mem_total_mb":24576}],"total_memory":34359738368,"disk":100000000000}
	}`)
	if err := json.Unmarshal(payload, &info); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if info.VPNConfig != "vpn-data" || info.TX != 1024 || info.RX != 512 || info.Duration.Int64() != 3600 {
		t.Errorf("core info = %+v", info)
	}
	if info.PortsMatrix["22"].Int64() != 10022 || info.SSHWeb == "" || info.UserEnvs["AUTH"] != false {
		t.Errorf("connection fields = %+v", info)
	}
	if info.NodeHW.GPU[0].Model != "RTX" || info.Prices.MarketPrice != 1234.5 || info.TerminationReason.Int64() != 3 {
		t.Errorf("metadata = %+v", info)
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
