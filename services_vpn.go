package octaspace

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// VPNRelay represents an available VPN relay node.
type VPNRelay struct {
	NodeID      int64   `json:"node_id"`
	Country     string  `json:"country"`
	CountryISO  string  `json:"country_iso"`
	City        string  `json:"city"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Residential bool    `json:"residential"`

	// Pricing
	PricePerGB      float64 `json:"traffic_price_usd"`
	PricePerGBEther float64 `json:"traffic_price_ether"`

	// Network speed. Raw values >100 000 are in bytes/sec; divide by 125 000
	// to convert to Mbps. Use DownloadMbps() / UploadMbps() helpers.
	NetDownMbits int64 `json:"net_down_mbits"`
	NetUpMbits   int64 `json:"net_up_mbits"`
}

// DownloadMbps returns the download speed in Mbit/s.
func (r VPNRelay) DownloadMbps() float64 { return normalizeNetMbits(r.NetDownMbits) }

// UploadMbps returns the upload speed in Mbit/s.
func (r VPNRelay) UploadMbps() float64 { return normalizeNetMbits(r.NetUpMbits) }

// VPNCreateParams holds parameters for creating a VPN session.
//
// Protocol is required when SubKind == "v2ray" (e.g. "vmess", "vless"); it is
// ignored by the API for other subkinds. Values in Extra are merged into the
// request body and take precedence over the typed fields on key conflict,
// allowing callers to pass forward-compatible fields without an SDK bump.
type VPNCreateParams struct {
	NodeID         int64          `json:"node_id"`
	SubKind        string         `json:"subkind,omitempty"` // "wg", "openvpn", "ss", "v2ray"
	Protocol       string         `json:"protocol,omitempty"`
	OrganizationID string         `json:"organization_id,omitempty"`
	ProjectID      string         `json:"project_id,omitempty"`
	Extra          map[string]any `json:"-"`
}

// MarshalJSON serializes VPNCreateParams, merging Extra into the top-level
// object so callers can forward arbitrary API fields.
func (p VPNCreateParams) MarshalJSON() ([]byte, error) {
	type alias VPNCreateParams
	base, err := json.Marshal(alias(p))
	if err != nil {
		return nil, err
	}
	if len(p.Extra) == 0 {
		return base, nil
	}
	var merged map[string]any
	if err := json.Unmarshal(base, &merged); err != nil {
		return nil, err
	}
	for k, v := range p.Extra {
		merged[k] = v
	}
	return json.Marshal(merged)
}

// VPNCreateResponse is returned when a VPN session is created.
type VPNCreateResponse struct {
	UUID string `json:"uuid"`
}

// VPNResource provides access to VPN service endpoints.
// Obtain via [Client.Services.VPN].
type VPNResource struct{ c *Client }

// List returns available VPN relay nodes.
// Pass nil or an empty [url.Values] to list all relays.
//
// GET /services/vpn
func (r *VPNResource) List(ctx context.Context, params url.Values) ([]VPNRelay, error) {
	var relays []VPNRelay
	return relays, r.c.get(ctx, "/services/vpn", params, &relays)
}

// Create starts a new VPN session on the specified relay node.
//
// POST /services/vpn
func (r *VPNResource) Create(ctx context.Context, params *VPNCreateParams) (*VPNCreateResponse, error) {
	if params == nil {
		return nil, fmt.Errorf("octaspace: VPNCreateParams must not be nil")
	}
	var resp VPNCreateResponse
	return &resp, r.c.post(ctx, "/services/vpn", params, &resp)
}
