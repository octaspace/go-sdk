package octaspace

import (
	"context"
	"errors"
)

// VPN protocols: the values of [VPNMachine.Protocols] and of
// [VPNStartParams.Subkind].
const (
	VPNWireGuard = "wg"         // WireGuard; Config.Text is a config file
	VPNAmneziaWG = "awg"        // AmneziaWG; Config.Text is a config file
	VPNReality   = "reality"    // VLESS Reality; Config.Text is a link
	VPNHTTPProxy = "http_proxy" // an HTTP proxy with a login; Config.HTTPProxy
)

// VPNResource rents VPN on nodes. Obtain via [Client.VPN].
type VPNResource struct{ c *Client }

// VPNMachine is a node offered for VPN, as listed by GET /services/vpn.
type VPNMachine struct {
	NodeNumber string `json:"node_number"`
	// TrafficPrice is per GB of traffic: VPN is billed by traffic only.
	TrafficPrice MicroUSD `json:"traffic_price"`
	// Protocols passed the end-to-end check on this node, at least one.
	// Start a session with one of them.
	Protocols []string `json:"protocols"`

	City       string   `json:"city"`
	Country    string   `json:"country"`
	CountryISO string   `json:"country_iso"` // ISO 3166-1 alpha-2
	Latitude   *float64 `json:"latitude"`
	Longitude  *float64 `json:"longitude"`
	// Residential reports that the node's address is believed to be
	// assigned to a residence.
	Residential bool `json:"residential"`

	NetDownMbps int `json:"net_down_mbps"` // Mbit/s; 0 if not measured
	NetUpMbps   int `json:"net_up_mbps"`   // Mbit/s; 0 if not measured
}

// List returns the nodes offered for VPN. Authentication is optional, as
// for [ComputeResource.List].
//
// GET /services/vpn
func (r *VPNResource) List(ctx context.Context) ([]VPNMachine, error) {
	var result []VPNMachine
	return result, r.c.getNullTolerant(ctx, "/services/vpn", nil, &result)
}

// ListRaw returns the raw JSON of GET /services/vpn.
func (r *VPNResource) ListRaw(ctx context.Context) ([]byte, error) {
	return r.c.getRaw(ctx, "/services/vpn")
}

// VPNStartParams describe a VPN session to start.
type VPNStartParams struct {
	// NodeNumber of a node from [VPNResource.List].
	NodeNumber string `json:"node_number"`
	// Subkind is the protocol, one of the node's Protocols: [VPNWireGuard],
	// [VPNAmneziaWG], [VPNReality] or [VPNHTTPProxy]. The HTTP proxy lets in
	// only the IP address the start request came from.
	Subkind string `json:"subkind"`
}

// Start starts a VPN session and returns its UUID. It is billed by traffic
// until stopped with [SessionsResource.Stop]. Once
// [SessionsResource.WaitReady] returns the session, its Config is what the
// client connects with.
//
// An account can have one VPN session per node. A rejected request is a
// [*ValidationError] with the reason, for example "Insufficient balance",
// "You already have a VPN session on this node", or "Node's VPN did not pass
// the connection check" for a protocol the node did not pass. Start is never
// retried automatically.
//
// POST /services/vpn
func (r *VPNResource) Start(ctx context.Context, params VPNStartParams) (string, error) {
	if params.NodeNumber == "" {
		return "", errors.New("octaspace: node number is required")
	}
	if params.Subkind == "" {
		return "", errors.New("octaspace: subkind is required")
	}
	var resp serviceUUID
	if err := r.c.post(ctx, "/services/vpn", params, &resp); err != nil {
		return "", err
	}
	return resp.UUID, nil
}
