package octaspace

import (
	"context"
	"errors"
	"net/url"
	"time"
)

// Hosting node statuses reported in [HostingNode.Status].
const (
	NodeIdle         = "idle"         // can take a service
	NodeRented       = "rented"       // runs a service
	NodeVerification = "verification" // busy with a network check (see Activity)
	NodeBenchmark    = "benchmark"    // busy with a benchmark (see Activity)
	NodeUnknown      = "unknown"
)

// HostingResource manages the nodes the account hosts.
// Obtain via [Client.Hosting].
type HostingResource struct{ c *Client }

// HostingNode is an online node of the account.
type HostingNode struct {
	// Number is the public node number, as renters see it and SetPrices
	// takes it.
	Number string `json:"number"`
	Status string `json:"status"` // NodeIdle, NodeRented, ...
	// Activity is the check the node is busy with; nil when there is none.
	Activity *NodeActivity `json:"activity"`
	// Maintenance reports that the node is in maintenance and not offered;
	// see [HostingResource.SetMaintenance].
	Maintenance bool `json:"maintenance"`
	// Verified reports whether the node passed verification; nil when not
	// known yet.
	Verified *bool `json:"verified"`
	// Services tells, per service the owner enabled, whether the node is
	// actually offered for it.
	Services map[string]ServiceOffer `json:"services"`

	Prices   NodePriceList   `json:"prices"`
	System   NodeSystem      `json:"system"`
	Location HostingLocation `json:"location"`

	IP  string `json:"ip"`
	OSN string `json:"osn"` // node software version
	// OnlineAt is when the node last connected; zero if unknown.
	OnlineAt time.Time `json:"online_at"`
	// Uptime is the seconds since the machine booted; nil if unknown.
	Uptime *int64 `json:"uptime"`
	// UptimePercent is the share of the last 30 days the node was online
	// (since it first came online, if later); 97 and more shows as 100.
	UptimePercent *float64 `json:"uptime_percent"`
}

// NodeActivity is a check a node is busy with.
type NodeActivity struct {
	Kind      string `json:"kind"` // "verification" or "benchmark"
	Step      string `json:"step"`
	StartedAt int64  `json:"started_at"` // Unix seconds; 0 if unknown
}

// ServiceOffer tells whether a node is offered for a service.
type ServiceOffer struct {
	Offered bool `json:"offered"`
	// Reason says why it is not offered, e.g. "ports_closed" or
	// "vpn_probe_failed".
	Reason string `json:"reason"`
	// Details is what the check found, for the owner.
	Details string `json:"details,omitempty"`
	// Protocols is the offer of each VPN protocol, for the vpn service only.
	Protocols map[string]ServiceOffer `json:"protocols,omitempty"`
}

// NodePriceList are the current prices of a hosted node.
type NodePriceList struct {
	Base           MicroUSD `json:"base"`             // per hour
	Storage        MicroUSD `json:"storage"`          // per GB of disk per hour
	Traffic        MicroUSD `json:"traffic"`          // per GB of traffic
	MinRentMinutes int      `json:"min_rent_minutes"` // 0 = off
}

// NodeSystem is the hardware and load a node reports.
type NodeSystem struct {
	CPUModelName   string   `json:"cpu_model_name"`
	CPUCores       int      `json:"cpu_cores"`
	CPULoadPercent *float64 `json:"cpu_load_percent"`
	OSVersion      string   `json:"os_version"`
	Memory         *Usage   `json:"memory"` // bytes
	Disk           *Usage   `json:"disk"`   // bytes
	// GPUs are reported as the node sends them: the API does not fix their
	// fields (e.g. "model", "mem_total_mb", "gpu_temperature", "fan_speed").
	// Values the node does not know are nil.
	GPUs []map[string]any `json:"gpus"`
}

// Usage is the size and use of a resource.
type Usage struct {
	Size        int64   `json:"size"`
	Free        int64   `json:"free"`
	Used        int64   `json:"used"`
	UsedPercent float64 `json:"used_percent"`
}

// HostingLocation is where a hosted node is, by GeoIP.
type HostingLocation struct {
	City      string  `json:"city"`
	Country   string  `json:"country"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Nodes returns the account's online nodes; a node that is down is not
// listed.
//
// GET /hosting/nodes
func (r *HostingResource) Nodes(ctx context.Context) ([]HostingNode, error) {
	var nodes []HostingNode
	return nodes, r.c.getNullTolerant(ctx, "/hosting/nodes", nil, &nodes)
}

// NodesRaw returns the raw JSON of GET /hosting/nodes.
func (r *HostingResource) NodesRaw(ctx context.Context) ([]byte, error) {
	return r.c.getRaw(ctx, "/hosting/nodes")
}

// Get returns one online node of the account; case, dashes and spaces in the
// number are ignored. A node that is offline, does not exist or belongs to
// another account is a [*NotFoundError]; a malformed number a
// [*ValidationError].
//
// GET /hosting/nodes/{number}
func (r *HostingResource) Get(ctx context.Context, number string) (*HostingNode, error) {
	if number == "" {
		return nil, errors.New("octaspace: node number is required")
	}
	var node HostingNode
	if err := r.c.getNullTolerant(ctx, "/hosting/nodes/"+url.PathEscape(number), nil, &node); err != nil {
		return nil, err
	}
	return &node, nil
}

// GetRaw returns the raw JSON of GET /hosting/nodes/{number}.
func (r *HostingResource) GetRaw(ctx context.Context, number string) ([]byte, error) {
	if number == "" {
		return nil, errors.New("octaspace: node number is required")
	}
	return r.c.getRaw(ctx, "/hosting/nodes/"+url.PathEscape(number))
}

// NodePrices sets the prices of a node; nil fields stay as they are, and at
// least one must be set. The API checks the ranges and rejects others with a
// [*ValidationError].
type NodePrices struct {
	// Base price per hour, 100..1 000 000 000 micro-USD.
	Base *MicroUSD `json:"base,omitempty"`
	// Storage price per GB of disk per hour, 0..1 000 000 000 micro-USD.
	Storage *MicroUSD `json:"storage,omitempty"`
	// Traffic price per GB, 100..1 000 000 000 micro-USD.
	Traffic *MicroUSD `json:"traffic,omitempty"`
	// MinRentMinutes of a compute session, 0..60; 0 turns it off. A session
	// its renter stops sooner is charged base and storage for this long, but
	// not when the node goes down or the session cannot be configured.
	MinRentMinutes *int `json:"min_rent_minutes,omitempty"`
}

// Price returns a pointer to p, for the optional fields of [NodePrices].
func Price(p MicroUSD) *MicroUSD { return &p }

// Int returns a pointer to v, for optional integer fields such as
// [NodePrices.MinRentMinutes].
func Int(v int) *int { return &v }

// SetPrices updates the prices of a node of the account, identified by its
// number (case, dashes and spaces are ignored). An unknown node is a
// [*NotFoundError]. SetPrices is never retried automatically.
//
// PATCH /hosting/nodes/{number}/prices
func (r *HostingResource) SetPrices(ctx context.Context, number string, prices NodePrices) error {
	if number == "" {
		return errors.New("octaspace: node number is required")
	}
	if prices == (NodePrices{}) {
		return errors.New("octaspace: set at least one price")
	}
	return r.c.patch(ctx, "/hosting/nodes/"+url.PathEscape(number)+"/prices", prices, nil)
}

// SetMaintenance puts a node of the account in maintenance (on = true) or
// takes it out (on = false). In maintenance the node is not offered for new
// rentals until it is taken out; sessions already running go on and their
// renters are notified. It works for a node that is offline too, and setting
// the state the node already has changes nothing. A maintenance period or
// description set in the OctaSpace dashboard is dropped either way.
//
// The node is identified by its number (case, dashes and spaces are
// ignored). An unknown node is a [*NotFoundError]. SetMaintenance is never
// retried automatically.
//
// PUT /hosting/nodes/{number}/maintenance (on),
// DELETE /hosting/nodes/{number}/maintenance (off)
func (r *HostingResource) SetMaintenance(ctx context.Context, number string, on bool) error {
	if number == "" {
		return errors.New("octaspace: node number is required")
	}
	path := "/hosting/nodes/" + url.PathEscape(number) + "/maintenance"
	if on {
		return r.c.put(ctx, path, nil, nil)
	}
	return r.c.delete(ctx, path, nil)
}
