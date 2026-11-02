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
	// Maintenance reports that the node is in maintenance and not offered.
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
	// Reason says why it is not offered, e.g. "ports_closed".
	Reason string `json:"reason"`
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
