package octaspace

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// readBody reads and returns r.Body bytes (helper used across test files).
func readBody(r *http.Request) ([]byte, error) {
	return io.ReadAll(r.Body)
}

func TestMachineRental_List(t *testing.T) {
	want := []MachineRental{
		{NodeID: 1, Country: "US", CPUCores: 8, TotalPriceUSD: 0.5},
		{NodeID: 2, Country: "DE", CPUCores: 16},
	}
	c, _ := newTestClient(t, jsonHandler(want))
	got, err := c.Services.MR.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	if got[0].NodeID != 1 || got[0].Country != "US" {
		t.Errorf("got[0] = %+v", got[0])
	}
}

func TestMachineRental_List_Error(t *testing.T) {
	c, _ := newTestClient(t, statusHandler(401, `{"error":"unauthorized"}`))
	_, err := c.Services.MR.List(context.Background(), nil)
	if !IsAuthError(err) {
		t.Errorf("expected AuthenticationError, got %T: %v", err, err)
	}
}

func TestMachineRental_Create_WireFormat(t *testing.T) {
	// Verify the API wire format: single-item array with id=0
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = readBody(r)
		jsonHandler(MachineRentalCreateResponse{UUID: "new-uuid"})(w, r)
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(0))

	_, err := c.Services.MR.Create(context.Background(), &MachineRentalCreateParams{
		NodeID:   123,
		DiskSize: 50,
		Image:    "ubuntu:24.04",
		App:      "app-uuid",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Payload must be an array
	var payload []map[string]any
	if err := json.Unmarshal(gotBody, &payload); err != nil {
		t.Fatalf("body is not a JSON array: %v — body: %s", err, gotBody)
	}
	if len(payload) != 1 {
		t.Fatalf("array length = %d, want 1", len(payload))
	}
	item := payload[0]
	if int(item["id"].(float64)) != 0 {
		t.Errorf("id = %v, want 0", item["id"])
	}
	if int(item["node_id"].(float64)) != 123 {
		t.Errorf("node_id = %v, want 123", item["node_id"])
	}
	if int(item["disk_size"].(float64)) != 50 {
		t.Errorf("disk_size = %v, want 50", item["disk_size"])
	}
	if item["image"] != "ubuntu:24.04" {
		t.Errorf("image = %v, want ubuntu:24.04", item["image"])
	}
}

func TestMachineRental_Create_DefaultSlices(t *testing.T) {
	// Envs, Ports, HTTPPorts must be empty objects/arrays (not null) in the wire format
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = readBody(r)
		jsonHandler(MachineRentalCreateResponse{UUID: "uuid"})(w, r)
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(0))

	c.Services.MR.Create(context.Background(), &MachineRentalCreateParams{
		NodeID: 1, DiskSize: 10, Image: "alpine",
		// Envs, Ports, HTTPPorts intentionally nil
	})

	var payload []map[string]any
	json.Unmarshal(gotBody, &payload)
	if len(payload) == 0 {
		t.Fatal("empty payload")
	}
	item := payload[0]

	// envs must be {} not null
	envs, ok := item["envs"].(map[string]any)
	if !ok {
		t.Errorf("envs is %T, want map ({})", item["envs"])
	}
	if len(envs) != 0 {
		t.Errorf("envs = %v, want empty", envs)
	}

	// ports must be [] not null
	ports, ok := item["ports"].([]any)
	if !ok {
		t.Errorf("ports is %T, want array ([])", item["ports"])
	}
	if len(ports) != 0 {
		t.Errorf("ports = %v, want empty", ports)
	}
}

func TestMachineRental_Create_WithEnvAndPorts(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = readBody(r)
		jsonHandler(MachineRentalCreateResponse{UUID: "uuid"})(w, r)
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(0))

	c.Services.MR.Create(context.Background(), &MachineRentalCreateParams{
		NodeID:    1,
		DiskSize:  50,
		Image:     "ubuntu:24.04",
		Envs:      map[string]string{"KEY": "VALUE"},
		Ports:     []int{8080, 9090},
		HTTPPorts: []int{80},
	})

	var payload []map[string]any
	json.Unmarshal(gotBody, &payload)
	item := payload[0]

	envs, _ := item["envs"].(map[string]any)
	if envs["KEY"] != "VALUE" {
		t.Errorf("envs[KEY] = %v, want VALUE", envs["KEY"])
	}

	ports, _ := item["ports"].([]any)
	if len(ports) != 2 {
		t.Errorf("ports len = %d, want 2", len(ports))
	}
}

func TestMachineRental_Create_ReturnsUUID(t *testing.T) {
	c, _ := newTestClient(t, jsonHandler(MachineRentalCreateResponse{UUID: "created-uuid"}))
	resp, err := c.Services.MR.Create(context.Background(), &MachineRentalCreateParams{
		NodeID: 1, DiskSize: 10, Image: "alpine",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if resp.UUID != "created-uuid" {
		t.Errorf("UUID = %q, want created-uuid", resp.UUID)
	}
}
