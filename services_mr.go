package octaspace

import (
	"context"
	"fmt"
	"net/url"
)

// MachineRental represents an available machine for rent.
type MachineRental struct {
	NodeID       int64  `json:"node_id"`
	Country      string `json:"country"`
	CountryISO   string `json:"country_iso"`
	City         string `json:"city"`
	Arch         string `json:"arch"`
	OSVersion    string `json:"os_version"`
	Virt         string `json:"virt"` // virtualisation type, e.g. "none", "kvm"
	MemoryType   string `json:"memory_type"`
	CPUModelName string `json:"cpu_model_name"`
	CPUVendorID  string `json:"cpu_vendor_id"`
	CPUCores     int    `json:"cpu_cores"`
	CPUSpeed     int    `json:"cpu_speed"`    // MHz
	MemorySpeed  int    `json:"memory_speed"` // MHz
	TotalMemory  int64  `json:"total_memory"` // bytes
	FreeDisk     int64  `json:"free_disk"`    // bytes
	DiskSpeedMBS int    `json:"disk_speed_mbs"`
	CUDAVersion  string `json:"cuda_version"`
	GPUVendor    string `json:"gpu_vendor"`
	IsHasGPU     bool   `json:"is_has_gpu"`
	IsHiveOS     bool   `json:"is_hive_os"`
	IsWSL        bool   `json:"is_wsl"`
	GPUPT        bool   `json:"gpu_pt"` // GPU passthrough
	GDM          bool   `json:"gdm"`

	// Scores
	AIScore      int `json:"ai_score"`
	BlenderScore int `json:"blender_score"`

	// Pricing (USD, values are scaled by 10 000; divide by 10 000 to get $/hr)
	CurrencyUSD   bool    `json:"currency_usd"`
	MarketPrice   float64 `json:"market_price"` // ETH/USD rate (used when CurrencyUSD=false)
	TotalPriceUSD float64 `json:"total_price_usd"`
	BaseUSD       int     `json:"base_usd"`
	StorageUSD    int     `json:"storage_usd"`
	TrafficUSD    int     `json:"traffic_usd"`

	// Pricing in Ether
	TotalPriceEther float64 `json:"total_price_ether"`
	BaseEther       float64 `json:"base_ether"`
	StorageEther    float64 `json:"storage_ether"`
	TrafficEther    float64 `json:"traffic_ether"`

	// Network speed. Raw values >100 000 are in bytes/sec; divide by 125 000
	// to convert to Mbps. Use NetDownMbps() / NetUpMbps() helpers.
	NetDownMbits int64 `json:"net_down_mbits"`
	NetUpMbits   int64 `json:"net_up_mbits"`

	// Reliability stats
	Reliability struct {
		Uptime    float64 `json:"uptime"`
		Stability float64 `json:"stability"`
		Rating    struct {
			Count int     `json:"count"`
			Score float64 `json:"score"`
		} `json:"rating"`
	} `json:"reliability"`

	// GPU list (flat array; same shape as Node.System.GPUs).
	GPUs []GPU `json:"gpus"`

	AvailablePorts int      `json:"available_ports"`
	Features       []string `json:"features"`
}

const bandwidthBPSPerMbit = 125_000.0

// NetDownMbps returns the download speed in Mbit/s.
// The raw API field net_down_mbits is in bytes/sec when > 100 000.
func (m MachineRental) NetDownMbps() float64 {
	return normalizeNetMbits(m.NetDownMbits)
}

// NetUpMbps returns the upload speed in Mbit/s.
func (m MachineRental) NetUpMbps() float64 {
	return normalizeNetMbits(m.NetUpMbits)
}

// normalizeNetMbits converts the raw net_*_mbits API value to Mbit/s.
// Values >100 000 are bytes/sec (divide by 125 000); smaller values are
// already in Mbit/s. This matches Cube's normalize_network_mbits_value logic.
func normalizeNetMbits(raw int64) float64 {
	if raw > 100_000 {
		return float64(raw) / bandwidthBPSPerMbit
	}
	return float64(raw)
}

// MachineRentalCreateParams holds the parameters for creating a machine rental session.
type MachineRentalCreateParams struct {
	NodeID       int64
	DiskSize     int
	Image        string
	App          string            // application UUID; empty string for custom images
	Envs         map[string]string // environment variables injected into the container
	Ports        []int             // TCP ports to expose
	HTTPPorts    []int             // HTTP ports to expose via reverse proxy
	StartCommand string
	Entrypoint   string
}

// MachineRentalCreateResponse is returned when a machine rental session is created.
type MachineRentalCreateResponse struct {
	UUID string `json:"uuid"`
}

// mrCreateItem is the wire format for POST /services/mr.
// The API expects a single-item array with id=0.
type mrCreateItem struct {
	ID           int               `json:"id"`
	NodeID       int64             `json:"node_id"`
	DiskSize     int               `json:"disk_size"`
	Image        string            `json:"image"`
	App          string            `json:"app"`
	Envs         map[string]string `json:"envs"`
	Ports        []int             `json:"ports"`
	HTTPPorts    []int             `json:"http_ports"`
	StartCommand string            `json:"start_command"`
	Entrypoint   string            `json:"entrypoint"`
}

// MachineRentalResource provides access to machine rental (MR) service endpoints.
// Obtain via [Client.Services.MR].
type MachineRentalResource struct{ c *Client }

// List returns available machines for rent.
// Pass nil or an empty [url.Values] to list all available machines.
//
// GET /services/mr
func (r *MachineRentalResource) List(ctx context.Context, params url.Values) ([]MachineRental, error) {
	var machines []MachineRental
	return machines, r.c.get(ctx, "/services/mr", params, &machines)
}

// Create starts a new machine rental session.
//
// The API expects the payload as a single-item array with id=0; this method
// handles that encoding automatically so callers do not need to know the wire format.
//
// POST /services/mr
func (r *MachineRentalResource) Create(ctx context.Context, params *MachineRentalCreateParams) (*MachineRentalCreateResponse, error) {
	if params == nil {
		return nil, fmt.Errorf("octaspace: MachineRentalCreateParams must not be nil")
	}
	envs := params.Envs
	if envs == nil {
		envs = map[string]string{}
	}
	ports := params.Ports
	if ports == nil {
		ports = []int{}
	}
	httpPorts := params.HTTPPorts
	if httpPorts == nil {
		httpPorts = []int{}
	}

	item := mrCreateItem{
		ID:           0,
		NodeID:       params.NodeID,
		DiskSize:     params.DiskSize,
		Image:        params.Image,
		App:          params.App,
		Envs:         envs,
		Ports:        ports,
		HTTPPorts:    httpPorts,
		StartCommand: params.StartCommand,
		Entrypoint:   params.Entrypoint,
	}

	var resp MachineRentalCreateResponse
	return &resp, r.c.post(ctx, "/services/mr", []mrCreateItem{item}, &resp)
}
