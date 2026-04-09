package sdk

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListMachinesForRent(t *testing.T) {
	want := []MachineRental{
		{
			NodeID:        5,
			Country:       "NL",
			City:          "Amsterdam",
			CPUModelName:  "AMD EPYC 7R13",
			CPUCores:      32,
			TotalMemory:   128 * 1024 * 1024 * 1024,
			TotalPriceUSD: 1.5,
			GPUs: []struct {
				Model      string `json:"model"`
				MemTotalMB int    `json:"mem_total_mb"`
			}{
				{Model: "RTX 4090", MemTotalMB: 24576},
			},
		},
	}
	c, cleanup := withServer(t, jsonHandler(http.StatusOK, want))
	defer cleanup()

	got, err := c.ListMachinesForRent()
	if err != nil {
		t.Fatalf("ListMachinesForRent: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	m := got[0]
	if m.NodeID != want[0].NodeID {
		t.Errorf("NodeID = %d, want %d", m.NodeID, want[0].NodeID)
	}
	if m.TotalPriceUSD != want[0].TotalPriceUSD {
		t.Errorf("TotalPriceUSD = %f, want %f", m.TotalPriceUSD, want[0].TotalPriceUSD)
	}
	if len(m.GPUs) != 1 || m.GPUs[0].Model != "RTX 4090" {
		t.Errorf("GPUs unexpected: %+v", m.GPUs)
	}
}

func TestListMachinesForRentError(t *testing.T) {
	c, cleanup := withServer(t, rawHandler(http.StatusUnauthorized, `{}`))
	defer cleanup()

	_, err := c.ListMachinesForRent()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListMachinesForRentRaw(t *testing.T) {
	body := `[{"node_id":5}]`
	c, cleanup := withServer(t, rawHandler(http.StatusOK, body))
	defer cleanup()

	got, err := c.ListMachinesForRentRaw()
	if err != nil {
		t.Fatalf("ListMachinesForRentRaw: %v", err)
	}
	if string(got) != body {
		t.Fatalf("body = %q, want %q", got, body)
	}
}

func TestDeployMachine(t *testing.T) {
	wantUUID := "deploy-uuid-123"
	var gotReq DeployRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotReq)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(DeployResponse{UUID: wantUUID})
	}))
	defer srv.Close()

	orig := BaseURL
	BaseURL = srv.URL
	defer func() { BaseURL = orig }()

	c := NewClient("key")
	c.http = srv.Client()

	req := DeployRequest{
		App:      "app-uuid-1",
		NodeID:   5,
		Image:    "jupyter:latest",
		DiskSize: 50,
		Envs:     map[string]string{"FOO": "bar"},
	}
	resp, err := c.DeployMachine(req)
	if err != nil {
		t.Fatalf("DeployMachine: %v", err)
	}
	if resp.UUID != wantUUID {
		t.Errorf("UUID = %q, want %q", resp.UUID, wantUUID)
	}
	if gotReq.App != req.App {
		t.Errorf("req.App = %q, want %q", gotReq.App, req.App)
	}
	if gotReq.NodeID != req.NodeID {
		t.Errorf("req.NodeID = %d, want %d", gotReq.NodeID, req.NodeID)
	}
	if gotReq.DiskSize != req.DiskSize {
		t.Errorf("req.DiskSize = %d, want %d", gotReq.DiskSize, req.DiskSize)
	}
	if gotReq.Envs["FOO"] != "bar" {
		t.Errorf("req.Envs[FOO] = %q, want %q", gotReq.Envs["FOO"], "bar")
	}
}

func TestDeployMachineMinimal(t *testing.T) {
	// Omitempty fields should not appear in the JSON body.
	var gotReq map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotReq)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(DeployResponse{UUID: "x"})
	}))
	defer srv.Close()

	orig := BaseURL
	BaseURL = srv.URL
	defer func() { BaseURL = orig }()

	c := NewClient("key")
	c.http = srv.Client()

	_, err := c.DeployMachine(DeployRequest{App: "app-1", NodeID: 1})
	if err != nil {
		t.Fatalf("DeployMachine: %v", err)
	}
	if _, ok := gotReq["image"]; ok {
		t.Error("image field should be omitted when empty")
	}
	if _, ok := gotReq["envs"]; ok {
		t.Error("envs field should be omitted when nil")
	}
}

func TestDeployMachineError(t *testing.T) {
	c, cleanup := withServer(t, rawHandler(http.StatusBadRequest, `{"error":"invalid"}`))
	defer cleanup()

	_, err := c.DeployMachine(DeployRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
