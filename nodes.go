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
			Free        int64   `json:"free"`
			Size        int64   `json:"size"`
			Used        int64   `json:"used"`
			UsedPercent float64 `json:"used_percent"`
		} `json:"memory"`
	} `json:"system"`
}

// NodeDetail is the response shape of GET /nodes/:id (see [NodesResource.Find]).
//
// IMPORTANT: this endpoint uses a DIFFERENT JSON envelope than GET /nodes,
// which is why it needs its own type instead of reusing [Node]:
//
//   - the hardware/system block is nested under "data" (not "system");
//   - memory fields use total_memory / free_memory (not size / free);
//   - the agent version is data.version (not the top-level "osn");
//   - it carries detail-only blocks (reliability, account_id, features);
//   - node prices are NOT returned here — read them from [NodesResource.List].
type NodeDetail struct {
	ID    int64  `json:"id"`
	IP    string `json:"ip"`
	State string `json:"state"`
	VrfDt string `json:"vrf_dt"`

	AccountID int64    `json:"account_id"`
	Features  []string `json:"features"`

	Data        NodeData        `json:"data"`
	Reliability NodeReliability `json:"reliability"`
	Location    NodeLocation    `json:"location"`
}

// NodeData is the "data" block of GET /nodes/:id: the node's hardware and
// agent state. Uptime and the agent Version live here (unlike [Node], where
// uptime/osn sit at the top level).
type NodeData struct {
	Arch             string  `json:"arch"`
	CPUCores         int     `json:"cpu_cores"`
	CPULoadPercent   float64 `json:"cpu_load_percent"`
	CPUModelName     string  `json:"cpu_model_name"`
	CPUVendorID      string  `json:"cpu_vendor_id"`
	OSVersion        string  `json:"os_version"`
	KernelVersion    string  `json:"kernel_version"`
	LinuxDistro      string  `json:"linux_distro"`
	LinuxRelease     string  `json:"linux_release"`
	CUDAVersion      string  `json:"cuda_version"`
	GPUDriverVersion string  `json:"gpu_driver_version"`
	GPUVendor        string  `json:"gpu_vendor"`
	Virt             string  `json:"virt"`
	// GDM reports whether a desktop display manager is available on the node.
	GDM      bool   `json:"gdm"`
	IsHiveOS bool   `json:"is_hive_os"`
	IsWSL    bool   `json:"is_wsl"`
	GPUPT    bool   `json:"gpu_pt"`
	Uptime   int64  `json:"uptime"`  // seconds since agent start
	Version  string `json:"version"` // node agent version (a.k.a. OSN)

	Disk       NodeDisk       `json:"disk"`
	Memory     NodeMemory     `json:"memory"`
	MemorySpec NodeMemorySpec `json:"memory_spec"`
	GPUs       []GPU          `json:"gpus"`
	// GPU is the legacy vendor-grouped GPU representation, also returned by
	// the node-detail endpoint alongside the flat GPUs list.
	GPU GPUByVendor `json:"gpu"`
}

// NodeDisk is the disk block from the node detail endpoint.
type NodeDisk struct {
	Free        int64   `json:"free"`
	Size        int64   `json:"size"`
	Used        int64   `json:"used"`
	UsedPercent float64 `json:"used_percent"`
}

// NodeMemory is the memory block from GET /nodes/:id. It uses
// total_memory / free_memory naming and exposes swap and cache detail that the
// list endpoint's memory block does not.
type NodeMemory struct {
	TotalMemory     int64 `json:"total_memory"`
	FreeMemory      int64 `json:"free_memory"`
	AvailableMemory int64 `json:"available_memory"`
	CachedMemory    int64 `json:"cached_memory"`
	BufferedMemory  int64 `json:"buffered_memory"`
	TotalSwap       int64 `json:"total_swap"`
	FreeSwap        int64 `json:"free_swap"`
}

// NodeMemorySpec describes installed memory reported by GET /nodes/:id.
type NodeMemorySpec struct {
	Speed int    `json:"speed"`
	Type  string `json:"type"`
}

// NodeReliability is the "reliability" block of the node detail endpoint.
type NodeReliability struct {
	Uptime    int     `json:"uptime"`    // percentage
	Stability float64 `json:"stability"` // percentage
	SAvgD     int64   `json:"s_avg_d"`
	SMaxD     int64   `json:"s_max_d"`
	SFailed   int     `json:"s_failed"`
	SNormal   int     `json:"s_normal"`
	Rating    struct {
		Count int     `json:"count"`
		Score float64 `json:"score"`
	} `json:"rating"`
}

// NodeLocation is the richer location block returned by the node detail
// endpoint (a superset of [Node]'s inline location).
type NodeLocation struct {
	City          string              `json:"city"`
	Country       string              `json:"country"`
	Continent     string              `json:"continent"`
	ContinentCode string              `json:"continentcode"`
	ISOCode       string              `json:"isocode"`
	Latitude      float64             `json:"latitude"`
	Longitude     float64             `json:"longitude"`
	Organisation  string              `json:"organisation"`
	Provider      string              `json:"provider"`
	Hostname      string              `json:"hostname"`
	IP            string              `json:"ip"`
	Proxy         string              `json:"proxy"`
	Risk          int                 `json:"risk"`
	Type          string              `json:"type"`
	Devices       NodeLocationDevices `json:"devices"`
}

// NodeLocationDevices contains the address and subnet counts attributed to a
// node location by GET /nodes/:id.
type NodeLocationDevices struct {
	Address int `json:"address"`
	Subnet  int `json:"subnet"`
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
	qp := nodeListValues(params)
	var nodes []Node
	return nodes, r.c.get(ctx, "/nodes", qp, &nodes)
}

// ListRaw returns the unmodified JSON body from the nodes endpoint.
//
// GET /nodes
func (r *NodesResource) ListRaw(ctx context.Context, params *ListNodesParams) ([]byte, error) {
	return r.c.getRawParams(ctx, "/nodes", nodeListValues(params))
}

func nodeListValues(params *ListNodesParams) url.Values {
	qp := url.Values{}
	if params == nil {
		return qp
	}
	if params.Country != nil {
		qp.Set("country", *params.Country)
	}
	if params.AppUUID != nil {
		qp.Set("app_uuid", *params.AppUUID)
	}
	return qp
}

// Find returns a single compute node by its numeric ID.
//
// The detail endpoint returns a different JSON envelope than [NodesResource.List],
// so it decodes into [NodeDetail] (not [Node]). Notably, node prices are not
// available here — use [NodesResource.List] if you need pricing.
//
// GET /nodes/:id
func (r *NodesResource) Find(ctx context.Context, nodeID int64) (*NodeDetail, error) {
	var node NodeDetail
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
