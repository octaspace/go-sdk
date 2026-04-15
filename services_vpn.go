package octaspace

import (
	"context"
	"fmt"
	"net/url"
)

// VPNRelay represents an available VPN relay node.
type VPNRelay struct {
	NodeID     int64   `json:"node_id"`
	Country    string  `json:"country"`
	CountryISO string  `json:"country_iso"`
	City       string  `json:"city"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Residential bool   `json:"residential"`

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
type VPNCreateParams struct {
	NodeID  int64  `json:"node_id"`
	SubKind string `json:"subkind,omitempty"` // e.g. "wg", "openvpn", "ss", "v2ray"
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
