package octaspace

import (
	"context"
	"encoding/json"
	"errors"
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

func TestMachineRental_List_PreservesWeiAndVendorGPUFields(t *testing.T) {
	payload := `[{"node_id":1,"base":"100000000000000000","storage":2,"traffic":"3","total_price_wei":"100000000000000005","gpu":{"nvidia":[{"model":"RTX","mem_total_mb":24576}]},"reliability":{"s_avg_d":0.2,"s_max_d":1.5,"s_normal":10,"s_failed":2}}]`
	c, _ := newTestClient(t, statusHandler(http.StatusOK, payload))
	machines, err := c.Services.MR.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	m := machines[0]
	if m.Base.String() != "100000000000000000" || m.TotalPriceWei.String() != "100000000000000005" {
		t.Errorf("Wei prices were not retained: %+v", m)
	}
	if len(m.GPU.NVIDIA) != 1 || m.GPU.NVIDIA[0].Model != "RTX" || m.Reliability.SFailed != 2 {
		t.Errorf("extended fields = %+v", m)
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
	c, _ := newTestClient(t, jsonHandler([]ServiceCreateResult{
		{ID: 0, UUID: "created-uuid"},
		{ID: 1, Reason: "node busy"},
	}))
	resp, err := c.Services.MR.Create(context.Background(), &MachineRentalCreateParams{
		NodeID: 1, DiskSize: 10, Image: "alpine",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if resp.UUID != "created-uuid" {
		t.Errorf("UUID = %q, want created-uuid", resp.UUID)
	}
	if len(resp.Results) != 2 || resp.Results[1].Reason != "node busy" {
		t.Errorf("Results = %+v, want per-item response", resp.Results)
	}
}

func TestMachineRental_CreateSingle_WireFormatAndResponse(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/services/mr" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q", got)
		}
		gotBody, _ = readBody(r)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"uuid":"single-uuid"}`))
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(0))

	resp, err := c.Services.MR.CreateSingle(context.Background(), &MachineRentalCreateParams{
		NodeID: 5, DiskSize: 20, Image: "alpine", App: "app-uuid",
		Envs: map[string]string{"A": "B"}, Ports: []int{22}, HTTPPorts: []int{80},
		StartCommand: "sleep infinity",
	})
	if err != nil {
		t.Fatalf("CreateSingle: %v", err)
	}
	if resp.UUID != "single-uuid" {
		t.Errorf("UUID = %q", resp.UUID)
	}
	var body map[string]any
	if err := json.Unmarshal(gotBody, &body); err != nil {
		t.Fatalf("body is not an object: %v; body=%s", err, gotBody)
	}
	if _, found := body["id"]; found {
		t.Errorf("single request unexpectedly contains id: %s", gotBody)
	}
	if body["start_command"] != "sleep infinity" {
		t.Errorf("single request lost command fields: %s", gotBody)
	}
	if _, found := body["entrypoint"]; found {
		t.Errorf("single request unexpectedly contains entrypoint: %s", gotBody)
	}
}

func TestMachineRental_CreateSingle_RejectsUnconfirmedScopeWithoutRequest(t *testing.T) {
	for _, params := range []*MachineRentalCreateParams{
		{Entrypoint: "/bin/sh"},
		{OrganizationID: "org"},
		{ProjectID: "project"},
	} {
		calls := 0
		srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
		c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(0))
		_, err := c.Services.MR.CreateSingle(context.Background(), params)
		srv.Close()
		if err == nil {
			t.Fatal("CreateSingle with unsupported object field error = nil")
		}
		if calls != 0 {
			t.Errorf("requests = %d, want 0", calls)
		}
	}
}

func TestMachineRental_CreateSingle_NilParamsDoesNotRequest(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(0))
	_, err := c.Services.MR.CreateSingle(context.Background(), nil)
	if err == nil {
		t.Fatal("CreateSingle(nil) error = nil")
	}
	if calls != 0 {
		t.Errorf("requests = %d, want 0", calls)
	}
}

func TestMachineRental_CreateSingle_DefaultCollections(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := readBody(r)
		if err := json.Unmarshal(data, &body); err != nil {
			t.Fatalf("Unmarshal body: %v", err)
		}
		_, _ = w.Write([]byte(`{"uuid":"single-uuid"}`))
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(0))
	_, err := c.Services.MR.CreateSingle(context.Background(), &MachineRentalCreateParams{
		NodeID: 1, DiskSize: 10, Image: "alpine",
	})
	if err != nil {
		t.Fatalf("CreateSingle: %v", err)
	}
	if envs, ok := body["envs"].(map[string]any); !ok || len(envs) != 0 {
		t.Errorf("envs = %#v, want {}", body["envs"])
	}
	for _, key := range []string{"ports", "http_ports"} {
		if values, ok := body[key].([]any); !ok || len(values) != 0 {
			t.Errorf("%s = %#v, want []", key, body[key])
		}
	}
}

func TestMachineRental_CreateSingle_MapsAPIErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		assert     func(*testing.T, error)
	}{
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			body:       `{"message":"invalid request"}`,
			assert: func(t *testing.T, err error) {
				var apiErr *APIError
				if !errors.As(err, &apiErr) || apiErr.Message != "invalid request" {
					t.Errorf("error = %v, want APIError with message", err)
				}
			},
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"error":"unauthorized"}`,
			assert: func(t *testing.T, err error) {
				if !IsAuthError(err) {
					t.Errorf("error = %T, want AuthenticationError", err)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := newTestClient(t, statusHandler(tt.statusCode, tt.body))
			_, err := c.Services.MR.CreateSingle(context.Background(), &MachineRentalCreateParams{})
			tt.assert(t, err)
		})
	}
}

func TestMachineRental_CreateSingle_CanceledContextDoesNotRequest(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(0))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.Services.MR.CreateSingle(ctx, &MachineRentalCreateParams{})
	if err == nil {
		t.Fatal("CreateSingle canceled context error = nil")
	}
	if calls != 0 {
		t.Errorf("requests = %d, want 0", calls)
	}
}
