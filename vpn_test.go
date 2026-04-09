package sdk

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListVPNRelays(t *testing.T) {
	want := []VPNRelay{
		{NodeID: 1, Country: "DE", City: "Frankfurt", PricePerGB: 0.01, DownloadSpeed: 1000, UploadSpeed: 500, Residential: false},
		{NodeID: 2, Country: "US", City: "New York", PricePerGB: 0.02, DownloadSpeed: 800, Residential: true},
	}
	c, cleanup := withServer(t, jsonHandler(http.StatusOK, want))
	defer cleanup()

	got, err := c.ListVPNRelays()
	if err != nil {
		t.Fatalf("ListVPNRelays: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i, r := range got {
		if r.NodeID != want[i].NodeID {
			t.Errorf("[%d] NodeID = %d, want %d", i, r.NodeID, want[i].NodeID)
		}
		if r.Country != want[i].Country {
			t.Errorf("[%d] Country = %q, want %q", i, r.Country, want[i].Country)
		}
		if r.Residential != want[i].Residential {
			t.Errorf("[%d] Residential = %v, want %v", i, r.Residential, want[i].Residential)
		}
	}
}

func TestListVPNRelaysError(t *testing.T) {
	c, cleanup := withServer(t, rawHandler(http.StatusInternalServerError, `err`))
	defer cleanup()

	_, err := c.ListVPNRelays()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListVPNRelaysRaw(t *testing.T) {
	body := `[{"node_id":1}]`
	c, cleanup := withServer(t, rawHandler(http.StatusOK, body))
	defer cleanup()

	got, err := c.ListVPNRelaysRaw()
	if err != nil {
		t.Fatalf("ListVPNRelaysRaw: %v", err)
	}
	if string(got) != body {
		t.Fatalf("body = %q, want %q", got, body)
	}
}

func TestConnectVPN(t *testing.T) {
	wantUUID := "vpn-session-uuid"
	var gotBody VPNConnectRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(VPNConnectResponse{UUID: wantUUID})
	}))
	defer srv.Close()

	orig := BaseURL
	BaseURL = srv.URL
	defer func() { BaseURL = orig }()

	c := NewClient("key")
	c.http = srv.Client()

	resp, err := c.ConnectVPN(7, "wireguard")
	if err != nil {
		t.Fatalf("ConnectVPN: %v", err)
	}
	if resp.UUID != wantUUID {
		t.Errorf("UUID = %q, want %q", resp.UUID, wantUUID)
	}
	if gotBody.NodeID != 7 {
		t.Errorf("NodeID = %d, want 7", gotBody.NodeID)
	}
	if gotBody.SubKind != "wireguard" {
		t.Errorf("SubKind = %q, want wireguard", gotBody.SubKind)
	}
}

func TestConnectVPNError(t *testing.T) {
	c, cleanup := withServer(t, rawHandler(http.StatusBadRequest, `{"error":"invalid node"}`))
	defer cleanup()

	_, err := c.ConnectVPN(0, "wireguard")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
