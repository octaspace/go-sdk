package octaspace

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func fixtureHandler(t *testing.T, path string) http.HandlerFunc {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(data) }
}

func TestVPN_ListFixture(t *testing.T) {
	var strict []VPNMachine
	decodeStrict(t, "testdata/next/services_vpn.json", &strict)
	var gotPath string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		fixtureHandler(t, "testdata/next/services_vpn.json")(w, r)
	})
	nodes, err := c.VPN.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/services/vpn" || len(nodes) != 1 || nodes[0].NodeNumber != "7KQ4-M2XD-4" ||
		nodes[0].CountryISO != "DE" || nodes[0].TrafficPrice != 100 || nodes[0].Latitude == nil {
		t.Errorf("List = %+v (path %s)", nodes, gotPath)
	}
	found := false
	for _, p := range nodes[0].Protocols {
		found = found || p == VPNWireGuard
	}
	if !found {
		t.Errorf("Protocols = %v", nodes[0].Protocols)
	}
}

func TestVPN_StartSendsNodeAndSubkind(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody []byte
	c, _ := newTestClient(t, captureBody(t, &gotMethod, &gotPath, &gotBody, 201, `{"uuid":"v-1"}`))
	uuid, err := c.VPN.Start(context.Background(), VPNStartParams{NodeNumber: "7KQ4-M2XD-4", Subkind: VPNHTTPProxy})
	if err != nil || uuid != "v-1" {
		t.Fatalf("Start = %q, %v", uuid, err)
	}
	if gotMethod != http.MethodPost || gotPath != "/services/vpn" ||
		string(gotBody) != `{"node_number":"7KQ4-M2XD-4","subkind":"http_proxy"}` {
		t.Errorf("sent %s %s %s", gotMethod, gotPath, gotBody)
	}
	if _, err := c.VPN.Start(context.Background(), VPNStartParams{NodeNumber: "N"}); err == nil {
		t.Error("Start without subkind: want an error")
	}
}

func TestVPN_StartIsNotRetried(t *testing.T) {
	attempts := 0
	url := newCountingServer(t, &attempts, 503, `{"message":"Service is temporarily unavailable"}`)
	c := NewClient("octa_ak_test", WithBaseURL(url), WithMaxRetries(3))
	if _, err := c.VPN.Start(context.Background(), VPNStartParams{NodeNumber: "N", Subkind: VPNWireGuard}); err == nil {
		t.Fatal("want an error")
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
}

func TestSession_VPNFixtures(t *testing.T) {
	var wg Session
	decodeStrict(t, "testdata/next/session_vpn_ready.json", &wg)
	if !wg.IsVPN() || wg.IsCompute() || wg.Subkind != VPNWireGuard || wg.Config == nil ||
		!strings.HasPrefix(wg.Config.Text, "[Interface]") || wg.Config.HTTPProxy != nil {
		t.Errorf("wg session = %+v, config %+v", wg, wg.Config)
	}
	var proxy Session
	decodeStrict(t, "testdata/next/session_vpn_http_proxy.json", &proxy)
	want := HTTPProxyConfig{IP: "203.0.113.7", Port: 40000, Username: "a1b2c3d4", Password: "tt1f70D8qUUknMdlHoIUyp3Q"}
	if proxy.Config == nil || proxy.Config.HTTPProxy == nil || *proxy.Config.HTTPProxy != want || proxy.Config.Text != "" {
		t.Errorf("proxy config = %+v", proxy.Config)
	}
	out, err := json.Marshal(proxy.Config)
	if err != nil || !strings.Contains(string(out), `"port":40000`) {
		t.Errorf("MarshalJSON = %s, %v", out, err)
	}
	var starting Session
	if err := json.Unmarshal([]byte(`{"service":"vpn","config":null}`), &starting); err != nil || starting.Config != nil {
		t.Errorf("null config = %+v, %v", starting.Config, err)
	}
}

func TestCompute_Apps(t *testing.T) {
	var strict []ComputeApp
	decodeStrict(t, "testdata/next/compute_apps.json", &strict)
	c, _ := newTestClient(t, fixtureHandler(t, "testdata/next/compute_apps.json"))
	apps, err := c.Compute.Apps(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) < 2 || apps[0].Image == "" || apps[0].Extra.MinDiskSize == 0 || apps[0].Personal || !apps[1].Personal {
		t.Errorf("Apps = %+v", apps)
	}
	if apps[1].Category != "" {
		t.Errorf(`"null" category = %q, want empty`, apps[1].Category)
	}
	raw, err := c.Compute.AppsRaw(context.Background())
	if err != nil || !strings.Contains(string(raw), `"category": "null"`) {
		t.Errorf("AppsRaw: %v", err)
	}
}

func TestCompute_AppsKeepNullStringsInUserData(t *testing.T) {
	body := `[{"uuid":"u","name":"n","image":"i","personal":true,"category":"null","entrypoint":"null","envs":{"MODE":"null"}}]`
	c, _ := newTestClient(t, jsonHandler(json.RawMessage(body)))
	apps, err := c.Compute.Apps(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if apps[0].Category != "" || apps[0].Entrypoint != "null" || apps[0].Envs["MODE"] != "null" {
		t.Errorf("Apps = %+v", apps[0])
	}
}

func TestComputeMachine_NewFields(t *testing.T) {
	c, _ := newTestClient(t, fixtureHandler(t, "testdata/next/services_compute.json"))
	machines, err := c.Compute.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	m := machines[1]
	if m.CountryISO != "DE" || m.OSVersion != "WSL - Ubuntu 24.04" || !m.IsWSL || m.Virt != "wsl" {
		t.Errorf("machine = %+v", m)
	}
}

func TestHosting_Get(t *testing.T) {
	var gotPath string
	nodes, err := os.ReadFile("testdata/next/hosting_nodes.json")
	if err != nil {
		t.Fatal(err)
	}
	var list []json.RawMessage
	if err := json.Unmarshal(nodes, &list); err != nil {
		t.Fatal(err)
	}
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		_, _ = w.Write(list[0])
	})
	node, err := c.Hosting.Get(context.Background(), "7kq4 m2xd 4")
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/hosting/nodes/7kq4%20m2xd%204" || node.Number == "" {
		t.Errorf("Get = %+v (path %s)", node, gotPath)
	}
	if _, err := c.Hosting.GetRaw(context.Background(), "7KQ4-M2XD-4"); err != nil {
		t.Errorf("GetRaw: %v", err)
	}
	nf, _ := newTestClient(t, statusHandler(404, `{"message":"Node not found"}`))
	if _, err := nf.Hosting.Get(context.Background(), "7KQ4-M2XD-4"); !IsNotFound(err) {
		t.Errorf("Get unknown = %v, want NotFoundError", err)
	}
}

func TestNetwork_History(t *testing.T) {
	var strict NetworkHistory
	decodeStrict(t, "testdata/next/network_history.json", &strict)
	var query string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		fixtureHandler(t, "testdata/next/network_history.json")(w, r)
	})
	start := time.Unix(1793491200, 0)
	h, err := c.Network.History(context.Background(), &NetworkHistoryParams{Start: start, Step: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if query != "start=1793491200&step=3600" {
		t.Errorf("query = %s", query)
	}
	points := h.Points()
	if len(points) != len(h.Timestamps) || len(points) == 0 || points[0].NodesOnline == nil ||
		*points[0].NodesOnline != *h.NodesOnline[0] || !points[0].Time.Equal(time.Unix(h.Timestamps[0], 0)) {
		t.Errorf("Points = %+v", points)
	}
	raw, err := c.Network.HistoryRaw(context.Background(), nil)
	if err != nil || query != "" || len(raw) == 0 {
		t.Errorf("HistoryRaw: query %q, %v", query, err)
	}
}

func TestNetwork_GPUsRented(t *testing.T) {
	c, _ := newTestClient(t, jsonHandler(json.RawMessage(`{"at":1,"power":{"gpus":4,"gpus_rented":"null"}}`)))
	info, err := c.Network.Info(context.Background())
	if err != nil || info.Power.GPUsRented != nil {
		t.Errorf("Info = %+v, %v", info, err)
	}
}

func TestStats_Mining(t *testing.T) {
	var query string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		fixtureHandler(t, "testdata/next/stats_mining.json")(w, r)
	})
	m, err := c.Stats.Mining(context.Background(), &MiningParams{Role: MiningCPU, Days: 7})
	if err != nil {
		t.Fatal(err)
	}
	if query != "days=7&role=cpu" {
		t.Errorf("query = %s", query)
	}
	if m.From == "" || m.Measure == "" || len(m.Miners) == 0 || len(m.Daily.Dates) == 0 {
		t.Errorf("Mining = %+v", m)
	}
	for _, row := range m.Miners {
		if row.Label == "null" {
			t.Errorf(`label "null" kept: %+v`, row)
		}
	}
	if _, err := c.Stats.MiningRaw(context.Background(), nil); err != nil {
		t.Errorf("MiningRaw: %v", err)
	}
}

func TestGPUModelStats_MemoryMB(t *testing.T) {
	c, _ := newTestClient(t, fixtureHandler(t, "testdata/next/stats_gpu_current.json"))
	m, err := c.Stats.GPUCurrent(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if m.GPUs[0].MemoryMB == nil || *m.GPUs[0].MemoryMB != 24576 || m.GPUs[1].MemoryMB != nil {
		t.Errorf("MemoryMB = %v, %v", m.GPUs[0].MemoryMB, m.GPUs[1].MemoryMB)
	}
}

func TestSessions_WaitStopped(t *testing.T) {
	calls := 0
	ready, err := os.ReadFile("testdata/next/session_compute_ready.json")
	if err != nil {
		t.Fatal(err)
	}
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		calls++
		if calls < 3 {
			_, _ = w.Write(ready)
			return
		}
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"message":"Session not found"}`))
	})
	if err := c.Sessions.WaitStopped(context.Background(), "u", &WaitOptions{Interval: time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	always, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(ready) })
	if err := always.Sessions.WaitStopped(ctx, "u", nil); err == nil {
		t.Error("WaitStopped with a done context: want an error")
	}
}
