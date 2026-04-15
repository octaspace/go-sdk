package octaspace

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVPN_List(t *testing.T) {
	want := []VPNRelay{
		{NodeID: 10, Country: "US", City: "New York", PricePerGB: 0.01, Residential: false},
		{NodeID: 20, Country: "DE", City: "Frankfurt", PricePerGB: 0.02, Residential: true},
	}
	c, _ := newTestClient(t, jsonHandler(want))
	got, err := c.Services.VPN.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	if got[0].NodeID != 10 || got[0].Country != "US" {
		t.Errorf("got[0] = %+v", got[0])
	}
	if !got[1].Residential {
		t.Error("got[1].Residential = false, want true")
	}
}

// TestVPN_NormalizeNetMbits verifies the bytes/sec → Mbit/s conversion logic.
// Cube uses: value > 100_000 → divide by 125_000 (bytes/sec to Mbps).
func TestVPN_NormalizeNetMbits(t *testing.T) {
	cases := []struct {
		raw  int64
		want float64
	}{
		{raw: 0, want: 0},
		{raw: 100, want: 100},         // already Mbps (≤100 000)
		{raw: 100_000, want: 100_000}, // boundary: treated as Mbps
		{raw: 100_001, want: float64(100_001) / 125_000},
		{raw: 115_785_799, want: float64(115_785_799) / 125_000}, // ~926 Mbps
	}
	for _, tc := range cases {
		r := VPNRelay{NetDownMbits: tc.raw}
		if got := r.DownloadMbps(); got != tc.want {
			t.Errorf("DownloadMbps(%d) = %v, want %v", tc.raw, got, tc.want)
		}
	}
}

func TestVPN_List_Error(t *testing.T) {
	c, _ := newTestClient(t, statusHandler(401, `{"error":"unauthorized"}`))
	_, err := c.Services.VPN.List(context.Background(), nil)
	if !IsAuthError(err) {
		t.Errorf("expected AuthenticationError, got %T: %v", err, err)
	}
}

func TestVPN_Create_SendsBody(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = readBody(r)
		jsonHandler(VPNCreateResponse{UUID: "vpn-uuid"})(w, r)
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(0))

	resp, err := c.Services.VPN.Create(context.Background(), &VPNCreateParams{
		NodeID:  42,
		SubKind: "wg",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if resp.UUID != "vpn-uuid" {
		t.Errorf("UUID = %q, want vpn-uuid", resp.UUID)
	}

	var body map[string]any
	json.Unmarshal(gotBody, &body)
	if int(body["node_id"].(float64)) != 42 {
		t.Errorf("node_id = %v, want 42", body["node_id"])
	}
	if body["subkind"] != "wg" {
		t.Errorf("subkind = %v, want wg", body["subkind"])
	}
}

func TestVPN_Create_OmitsEmptySubKind(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = readBody(r)
		jsonHandler(VPNCreateResponse{UUID: "vpn-uuid"})(w, r)
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(0))

	c.Services.VPN.Create(context.Background(), &VPNCreateParams{NodeID: 1})
	var body map[string]any
	json.Unmarshal(gotBody, &body)
	if _, exists := body["subkind"]; exists {
		t.Error("subkind should be omitted when empty")
	}
}

func TestVPN_Create_Error(t *testing.T) {
	c, _ := newTestClient(t, statusHandler(422, `{"error":"invalid params"}`))
	_, err := c.Services.VPN.Create(context.Background(), &VPNCreateParams{NodeID: 99})
	var valErr *ValidationError
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	_ = valErr
}

func TestVPN_Create_V2RayWithProtocol(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = readBody(r)
		jsonHandler(VPNCreateResponse{UUID: "v2ray-uuid"})(w, r)
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(0))

	resp, err := c.Services.VPN.Create(context.Background(), &VPNCreateParams{
		NodeID:   42,
		SubKind:  "v2ray",
		Protocol: "vmess",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if resp.UUID != "v2ray-uuid" {
		t.Errorf("UUID = %q, want v2ray-uuid", resp.UUID)
	}

	var body map[string]any
	json.Unmarshal(gotBody, &body)
	if body["subkind"] != "v2ray" {
		t.Errorf("subkind = %v, want v2ray", body["subkind"])
	}
	if body["protocol"] != "vmess" {
		t.Errorf("protocol = %v, want vmess", body["protocol"])
	}
}

func TestVPN_Create_OmitsEmptyOptionalFields(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = readBody(r)
		jsonHandler(VPNCreateResponse{UUID: "uuid"})(w, r)
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(0))

	c.Services.VPN.Create(context.Background(), &VPNCreateParams{NodeID: 1, SubKind: "wg"})
	var body map[string]any
	json.Unmarshal(gotBody, &body)
	if _, exists := body["protocol"]; exists {
		t.Error("protocol should be omitted when empty")
	}
	if _, exists := body["organization_id"]; exists {
		t.Error("organization_id should be omitted when empty")
	}
	if _, exists := body["project_id"]; exists {
		t.Error("project_id should be omitted when empty")
	}
}

func TestVPN_Create_WithOrganizationAndProject(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = readBody(r)
		jsonHandler(VPNCreateResponse{UUID: "org-uuid"})(w, r)
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(0))

	c.Services.VPN.Create(context.Background(), &VPNCreateParams{
		NodeID:         42,
		SubKind:        "wg",
		OrganizationID: "org-123",
		ProjectID:      "proj-456",
	})
	var body map[string]any
	json.Unmarshal(gotBody, &body)
	if body["organization_id"] != "org-123" {
		t.Errorf("organization_id = %v, want org-123", body["organization_id"])
	}
	if body["project_id"] != "proj-456" {
		t.Errorf("project_id = %v, want proj-456", body["project_id"])
	}
}
