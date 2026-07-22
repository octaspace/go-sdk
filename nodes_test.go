package octaspace

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNodes_List(t *testing.T) {
	want := []Node{
		{ID: 1, IP: "1.2.3.4", State: "online", OSN: "linux"},
		{ID: 2, IP: "5.6.7.8", State: "offline"},
	}
	c, _ := newTestClient(t, jsonHandler(want))
	got, err := c.Nodes.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i, n := range got {
		if n.ID != want[i].ID {
			t.Errorf("[%d] ID = %d, want %d", i, n.ID, want[i].ID)
		}
		if n.IP != want[i].IP {
			t.Errorf("[%d] IP = %q, want %q", i, n.IP, want[i].IP)
		}
	}
}

func TestNodes_List_WithParams(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		jsonHandler([]Node{})(w, r)
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(0))

	country := "DE"
	appUUID := "app-uuid-1"
	c.Nodes.List(context.Background(), &ListNodesParams{
		Country: &country,
		AppUUID: &appUUID,
	})

	if gotQuery == "" {
		t.Error("expected query parameters, got empty string")
	}
	for _, want := range []string{"country=DE", "app_uuid=app-uuid-1"} {
		if !contains(gotQuery, want) {
			t.Errorf("query %q does not contain %q", gotQuery, want)
		}
	}
}

func TestNodes_List_NestedFields(t *testing.T) {
	want := []Node{{ID: 3, State: "online"}}
	want[0].Location.City = "Berlin"
	want[0].Location.Country = "DE"
	want[0].Prices.Base = 10
	want[0].System.CPUCores = 16
	want[0].System.GPUs = []GPU{{Model: "A100", MemTotalMB: 40960, MemFreeMB: 30000}}

	c, _ := newTestClient(t, jsonHandler(want))
	got, err := c.Nodes.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	n := got[0]
	if n.Location.City != "Berlin" {
		t.Errorf("Location.City = %q, want Berlin", n.Location.City)
	}
	if n.Prices.Base != 10 {
		t.Errorf("Prices.Base = %d, want 10", n.Prices.Base)
	}
	if len(n.System.GPUs) != 1 || n.System.GPUs[0].Model != "A100" {
		t.Errorf("System.GPUs = %+v", n.System.GPUs)
	}
}

func TestNodes_List_Empty(t *testing.T) {
	c, _ := newTestClient(t, jsonHandler([]Node{}))
	got, err := c.Nodes.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty, got %d", len(got))
	}
}

func TestNodes_List_Error(t *testing.T) {
	c, _ := newTestClient(t, statusHandler(500, `{"error":"server error"}`))
	_, err := c.Nodes.List(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestNodes_Find(t *testing.T) {
	// GET /nodes/:id returns a different envelope than GET /nodes: the system
	// block is nested under "data", memory uses total_memory/free_memory, and
	// the agent version is data.version. Decode a real captured response and
	// assert those fields land where callers expect them.
	body, err := os.ReadFile(filepath.Join("testdata", "node_detail.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/nodes/65" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(0))

	got, err := c.Nodes.Find(context.Background(), 65)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}

	if got.ID != 65 || got.IP != "109.108.71.177" || got.State != "busy" {
		t.Errorf("top-level: got id=%d ip=%q state=%q", got.ID, got.IP, got.State)
	}
	if got.AccountID != 3 {
		t.Errorf("AccountID = %d, want 3", got.AccountID)
	}
	if got.Data.Uptime != 1367738 {
		t.Errorf("Data.Uptime = %d, want 1367738", got.Data.Uptime)
	}
	if got.Data.Version != "0.0.74" {
		t.Errorf("Data.Version = %q, want 0.0.74", got.Data.Version)
	}
	if got.Data.CPUCores != 12 || got.Data.CPUModelName == "" || got.Data.Arch != "x86_64" {
		t.Errorf("Data cpu/arch: cores=%d model=%q arch=%q", got.Data.CPUCores, got.Data.CPUModelName, got.Data.Arch)
	}
	if got.Data.OSVersion != "Ubuntu 24.04.4 LTS" {
		t.Errorf("Data.OSVersion = %q", got.Data.OSVersion)
	}
	if got.Data.Memory.TotalMemory != 67319312384 || got.Data.Memory.FreeMemory != 31259217920 {
		t.Errorf("Data.Memory total=%d free=%d", got.Data.Memory.TotalMemory, got.Data.Memory.FreeMemory)
	}
	if got.Data.Disk.Size != 858574028800 || got.Data.Disk.Free != 207427837952 {
		t.Errorf("Data.Disk size=%d free=%d", got.Data.Disk.Size, got.Data.Disk.Free)
	}
	if len(got.Data.GPUs) != 1 || got.Data.GPUs[0].Model != "NVIDIA GeForce RTX 3090" {
		t.Errorf("Data.GPUs = %+v", got.Data.GPUs)
	}
	if got.Reliability.Stability != 98.41 {
		t.Errorf("Reliability.Stability = %v, want 98.41", got.Reliability.Stability)
	}
	if got.Location.City != "Bilychi" || got.Location.Country != "Ukraine" {
		t.Errorf("Location = %+v", got.Location)
	}
}

func TestNodes_Find_NotFound(t *testing.T) {
	c, _ := newTestClient(t, statusHandler(404, `{"error":"not found"}`))
	_, err := c.Nodes.Find(context.Background(), 999)
	if !IsNotFound(err) {
		t.Errorf("expected NotFoundError, got %T: %v", err, err)
	}
}

func TestNodes_Reboot(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(0))

	if err := c.Nodes.Reboot(context.Background(), 7); err != nil {
		t.Fatalf("Reboot: %v", err)
	}
	if gotPath != "/nodes/7/reboot" {
		t.Errorf("path = %q, want /nodes/7/reboot", gotPath)
	}
}

func TestNodes_UpdatePrices(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(0))

	prices := map[string]any{"gpu_hour": 0.5, "cpu_hour": 0.1}
	if err := c.Nodes.UpdatePrices(context.Background(), 5, prices); err != nil {
		t.Fatalf("UpdatePrices: %v", err)
	}
	if gotMethod != "PATCH" {
		t.Errorf("method = %q, want PATCH", gotMethod)
	}
	if gotPath != "/nodes/5/prices" {
		t.Errorf("path = %q, want /nodes/5/prices", gotPath)
	}
}

// contains is a simple substring helper for query string assertions.
func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsAt(s, sub))
}

func containsAt(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
