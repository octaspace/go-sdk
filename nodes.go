package octaspace

import (
	"context"
	"fmt"
	"net/url"
)

// GPU represents a single GPU device reported by a node.
// The same shape is used inside [Node.System] and [MachineRental].
type GPU struct {
	Idx                   int     `json:"idx"`
	Model                 string  `json:"model"`
	BusID                 string  `json:"bus_id"`
	MemTotalMB            int     `json:"mem_total_mb"`
	MemFreeMB             int     `json:"mem_free_mb"`
	GPUTemperature        int     `json:"gpu_temperature"`
	GPUUtilization        int     `json:"gpu_utilization"`
	FanSpeed              int     `json:"fan_speed"`
	PcieLinkGen           int     `json:"pcie_link_gen"`
	PcieLinkWidth         int     `json:"pcie_link_width"`
	PowerLimitWatt        float64 `json:"power_limit_watt"`
	PowerLimitDefaultWatt float64 `json:"power_limit_default_watt"`
	PState                string  `json:"pstate"`
}

// Node represents a compute node in the OctaSpace network.
type Node struct {
	ID     int64  `json:"id"`
	IP     string `json:"ip"`
	State  string `json:"state"`  // e.g. "idle", "busy", "offline"
	OSN    string `json:"osn"`    // OctaSpace Node agent version, e.g. "0.0.73"
	Uptime int64  `json:"uptime"` // seconds since agent start
	VrfDt  string `json:"vrf_dt"` // timestamp of the last VRF (verifiable random function) verification

	Location struct {
		City      string  `json:"city"`
		Country   string  `json:"country"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"location"`

	// Prices are node-owner-configured values denominated in Wei.
	// Use base_usd / storage_usd / traffic_usd on MachineRental for USD prices.
	Prices struct {
		Base    int64 `json:"base"`
		Storage int64 `json:"storage"`
		Traffic int64 `json:"traffic"`
	} `json:"prices"`

	System struct {
		Arch           string  `json:"arch"`
		CPUCores       int     `json:"cpu_cores"`
		CPULoadPercent float64 `json:"cpu_load_percent"`
		CPUModelName   string  `json:"cpu_model_name"`
		OSVersion      string  `json:"os_version"`

		Disk struct {
			Free        int64   `json:"free"`
			Size        int64   `json:"size"`
			Used        int64   `json:"used"`
			UsedPercent float64 `json:"used_percent"`
		} `json:"disk"`

		GPUs []GPU `json:"gpus"`

		Memory struct {
			Free int64 `json:"free"`
			Size int64 `json:"size"`
		} `json:"memory"`
	} `json:"system"`
}

// ListNodesParams contains optional filter parameters for [NodesResource.List].
type ListNodesParams struct {
	Country *string // filter by ISO country code, e.g. "US" or "DE"
	AppUUID *string // filter by compatible application UUID
}

// NodesResource provides access to compute node endpoints.
// Obtain via [Client.Nodes].
type NodesResource struct{ c *Client }

// List returns all compute nodes. Pass nil params to list all nodes without filters.
//
// GET /nodes
func (r *NodesResource) List(ctx context.Context, params *ListNodesParams) ([]Node, error) {
	qp := url.Values{}
	if params != nil {
		if params.Country != nil {
			qp.Set("country", *params.Country)
		}
		if params.AppUUID != nil {
			qp.Set("app_uuid", *params.AppUUID)
		}
	}
	var nodes []Node
	return nodes, r.c.get(ctx, "/nodes", qp, &nodes)
}

// Find returns a single compute node by its numeric ID.
//
// GET /nodes/:id
func (r *NodesResource) Find(ctx context.Context, nodeID int64) (*Node, error) {
	var node Node
	return &node, r.c.get(ctx, fmt.Sprintf("/nodes/%d", nodeID), nil, &node)
}

// DownloadIdent returns the node's identity file as raw bytes.
//
// GET /nodes/:id/ident
func (r *NodesResource) DownloadIdent(ctx context.Context, nodeID int64) ([]byte, error) {
	return r.c.getRaw(ctx, fmt.Sprintf("/nodes/%d/ident", nodeID))
}

// DownloadLogs returns the node's log archive as raw bytes.
//
// GET /nodes/:id/logs
func (r *NodesResource) DownloadLogs(ctx context.Context, nodeID int64) ([]byte, error) {
	return r.c.getRaw(ctx, fmt.Sprintf("/nodes/%d/logs", nodeID))
}

// UpdatePrices updates the pricing configuration for a node.
//
// prices is forwarded as-is to the API; the exact set of accepted fields is
// defined by the OctaSpace API and is not enforced by the SDK. Pass nil to
// send an empty PATCH (no body).
//
// PATCH /nodes/:id/prices
func (r *NodesResource) UpdatePrices(ctx context.Context, nodeID int64, prices map[string]any) error {
	path := fmt.Sprintf("/nodes/%d/prices", nodeID)
	if prices == nil {
		return r.c.patch(ctx, path, nil, nil)
	}
	return r.c.patch(ctx, path, prices, nil)
}

// Reboot triggers a reboot of the node.
//
// GET /nodes/:id/reboot
func (r *NodesResource) Reboot(ctx context.Context, nodeID int64) error {
	return r.c.get(ctx, fmt.Sprintf("/nodes/%d/reboot", nodeID), nil, nil)
}
