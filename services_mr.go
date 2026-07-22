package octaspace

import (
	"bytes"
	"context"
	"encoding/json"
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
	AIScore            int `json:"ai_score"`
	BlenderScore       int `json:"blender_score"`
	BlenderCPUScore    int `json:"blender_cpu_score"`
	BlenderCUDAScore   int `json:"blender_cuda_score"`
	BlenderOptixScore  int `json:"blender_optix_score"`
	BlenderHIPScore    int `json:"blender_hip_score"`
	BlenderOneAPIScore int `json:"blender_oneapi_score"`

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
	// Raw Wei prices are retained separately from Ether and USD values.
	Base          BigInt `json:"base"`
	Storage       BigInt `json:"storage"`
	Traffic       BigInt `json:"traffic"`
	TotalPriceWei BigInt `json:"total_price_wei"`
	// Legacy Swagger price names are distinct from the live base/storage/
	// traffic fields and are therefore retained separately.
	BasePrice    BigInt `json:"base_price"`
	StoragePrice BigInt `json:"storage_price"`
	TrafficPrice BigInt `json:"traffic_price"`
	TotalPrice   BigInt `json:"total_price"`

	// Network speed. Raw values >100 000 are in bytes/sec; divide by 125 000
	// to convert to Mbps. Use NetDownMbps() / NetUpMbps() helpers.
	NetDownMbits int64 `json:"net_down_mbits"`
	NetUpMbits   int64 `json:"net_up_mbits"`

	// Reliability stats
	Reliability MachineReliability `json:"reliability"`

	// GPU list (flat array; same shape as Node.System.GPUs).
	GPUs []GPU `json:"gpus"`
	// GPU is the legacy vendor-grouped representation returned by some API
	// deployments. Prefer GPUs when it is present.
	GPU GPUByVendor `json:"gpu"`

	AvailablePorts int      `json:"available_ports"`
	Features       []string `json:"features"`
}

// GPUByVendor groups the legacy machine-rental GPU payload by vendor.
type GPUByVendor struct {
	NVIDIA    []GPU `json:"nvidia"`
	AMD       []GPU `json:"amd"`
	Intel     []GPU `json:"intel"`
	Undefined []GPU `json:"undefined"`
}

// MachineReliability accepts both the live object and the older Swagger
// numeric score. LegacyScore is populated only for the numeric form.
type MachineReliability struct {
	Uptime      float64  `json:"uptime"`
	Stability   float64  `json:"stability"`
	SAvgD       float64  `json:"s_avg_d"`
	SMaxD       float64  `json:"s_max_d"`
	SNormal     int      `json:"s_normal"`
	SFailed     int      `json:"s_failed"`
	LegacyScore *float64 `json:"-"`
	Rating      struct {
		Count int     `json:"count"`
		Score float64 `json:"score"`
	} `json:"rating"`
}

// UnmarshalJSON implements tolerant decoding for reliability schema drift.
func (r *MachineReliability) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) > 0 && data[0] != '{' && string(data) != "null" {
		var score float64
		if err := json.Unmarshal(data, &score); err != nil {
			return err
		}
		*r = MachineReliability{LegacyScore: &score}
		return nil
	}
	type alias MachineReliability
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*r = MachineReliability(decoded)
	return nil
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
	NodeID         int64
	DiskSize       int
	Image          string
	App            string            // application UUID; empty string for custom images
	Envs           map[string]string // environment variables injected into the container
	Ports          []int             // TCP ports to expose
	HTTPPorts      []int             // HTTP ports to expose via reverse proxy
	StartCommand   string
	Entrypoint     string
	OrganizationID string
	ProjectID      string
}

// MachineRentalCreateResponse is returned when a machine rental session is created.
// It preserves UUID as the first successful result for single-node requests.
type MachineRentalCreateResponse = ServiceCreateResponse

// mrCreateItem is the wire format for POST /services/mr.
// The API expects a single-item array with id=0.
type mrCreateFields struct {
	NodeID         int64             `json:"node_id"`
	DiskSize       int               `json:"disk_size"`
	Image          string            `json:"image"`
	App            string            `json:"app"`
	Envs           map[string]string `json:"envs"`
	Ports          []int             `json:"ports"`
	HTTPPorts      []int             `json:"http_ports"`
	StartCommand   string            `json:"start_command"`
	Entrypoint     string            `json:"entrypoint"`
	OrganizationID string            `json:"organization_id,omitempty"`
	ProjectID      string            `json:"project_id,omitempty"`
}

type mrCreateItem struct {
	ID int `json:"id"`
	mrCreateFields
}

// mrCreateSingleRequest is the documented object form of POST /services/mr.
// It intentionally has no batch item ID and includes only fields advertised by
// the current Swagger schema for this object endpoint.
type mrCreateSingleRequest struct {
	NodeID       int64             `json:"node_id"`
	DiskSize     int               `json:"disk_size"`
	Image        string            `json:"image"`
	App          string            `json:"app"`
	Envs         map[string]string `json:"envs"`
	Ports        []int             `json:"ports"`
	HTTPPorts    []int             `json:"http_ports"`
	StartCommand string            `json:"start_command"`
}

// MachineRentalCreateSingleResponse is returned by the documented object form
// of POST /services/mr.
type MachineRentalCreateSingleResponse struct {
	UUID string `json:"uuid"`
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

// ListRaw returns the unmodified JSON body from the machine-rental endpoint.
//
// GET /services/mr
func (r *MachineRentalResource) ListRaw(ctx context.Context, params url.Values) ([]byte, error) {
	return r.c.getRawParams(ctx, "/services/mr", params)
}

// Create starts a new machine rental session.
//
// The API expects the payload as a single-item array with id=0; this method
// handles that encoding automatically so callers do not need to know the wire format.
//
// POST /services/mr
func (r *MachineRentalResource) Create(ctx context.Context, params *MachineRentalCreateParams) (*MachineRentalCreateResponse, error) {
	fields, err := normalizeMRCreateParams(params)
	if err != nil {
		return nil, err
	}

	var resp MachineRentalCreateResponse
	return &resp, r.c.post(ctx, "/services/mr", []mrCreateItem{{ID: 0, mrCreateFields: fields}}, &resp)
}

// CreateSingle starts one machine-rental session using the documented object
// request and object response form. Create retains the production batch wire
// format (a single-item array with id=0) for compatibility.
//
// Entrypoint, OrganizationID, and ProjectID are not sent because the current
// object schema does not advertise them. Use Create when any is needed.
//
// POST /services/mr
func (r *MachineRentalResource) CreateSingle(ctx context.Context, params *MachineRentalCreateParams) (*MachineRentalCreateSingleResponse, error) {
	fields, err := normalizeMRCreateParams(params)
	if err != nil {
		return nil, err
	}
	if fields.Entrypoint != "" || fields.OrganizationID != "" || fields.ProjectID != "" {
		return nil, fmt.Errorf("octaspace: CreateSingle does not support Entrypoint, OrganizationID, or ProjectID; use Create")
	}
	var resp MachineRentalCreateSingleResponse
	request := mrCreateSingleRequest{
		NodeID: fields.NodeID, DiskSize: fields.DiskSize, Image: fields.Image,
		App: fields.App, Envs: fields.Envs, Ports: fields.Ports,
		HTTPPorts: fields.HTTPPorts, StartCommand: fields.StartCommand,
	}
	return &resp, r.c.post(ctx, "/services/mr", request, &resp)
}

func normalizeMRCreateParams(params *MachineRentalCreateParams) (mrCreateFields, error) {
	if params == nil {
		return mrCreateFields{}, fmt.Errorf("octaspace: MachineRentalCreateParams must not be nil")
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
	return mrCreateFields{
		NodeID: params.NodeID, DiskSize: params.DiskSize, Image: params.Image,
		App: params.App, Envs: envs, Ports: ports, HTTPPorts: httpPorts,
		StartCommand: params.StartCommand, Entrypoint: params.Entrypoint,
		OrganizationID: params.OrganizationID, ProjectID: params.ProjectID,
	}, nil
}
