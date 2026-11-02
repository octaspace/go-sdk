package octaspace

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ComputeMachine is a machine available for rent, as listed by
// GET /services/compute.
type ComputeMachine struct {
	// NodeNumber identifies the machine: 8 Crockford base32 symbols and a
	// check symbol, e.g. "7KQ4-M2XD-4". Pass it to [ComputeResource.Start].
	NodeNumber string `json:"node_number"`

	City    string `json:"city"`
	Country string `json:"country"` // empty if unknown

	Arch         string `json:"arch"` // "x86_64" or "aarch64"
	CPUModelName string `json:"cpu_model_name"`
	CPUCores     int    `json:"cpu_cores"`

	Memory      int64  `json:"memory"`       // free memory, bytes
	MemorySpeed int    `json:"memory_speed"` // MT/s
	MemoryType  string `json:"memory_type"`

	Disk         int64   `json:"disk"`      // free disk space, bytes
	DiskType     string  `json:"disk_type"` // "nvme", "ssd", "hdd"; empty if unknown
	DiskSpeedMBS float64 `json:"disk_speed_mbs"`

	// Network speed in Mbit/s; 0 if not measured.
	NetDownMbps int `json:"net_down_mbps"`
	NetUpMbps   int `json:"net_up_mbps"`

	FreePorts int `json:"free_ports"` // service ports still free on the machine

	GPUVendor    string       `json:"gpu_vendor"`
	CUDAVersion  string       `json:"cuda_version"`
	GPUs         []MachineGPU `json:"gpus"`
	AIScore      int64        `json:"ai_score"`      // NVIDIA GPU performance counter
	BlenderScore int64        `json:"blender_score"` // Blender benchmark for the GPU vendor

	BasePrice      MicroUSD `json:"base_price"`       // per hour of rent
	StoragePrice   MicroUSD `json:"storage_price"`    // per GB of disk per hour
	TrafficPrice   MicroUSD `json:"traffic_price"`    // per GB of traffic
	MinRentMinutes int      `json:"min_rent_minutes"` // 0 = none; see [ComputeResource.Start]

	Reliability Reliability `json:"reliability"`
}

// RenderMachine is a node available for render (an NVIDIA GPU with a Blender
// score), as listed by GET /services/render. Render is billed by time only.
type RenderMachine struct {
	NodeNumber string `json:"node_number"`

	City    string `json:"city"`
	Country string `json:"country"`

	CPUModelName string `json:"cpu_model_name"`
	CPUCores     int    `json:"cpu_cores"`
	Memory       int64  `json:"memory"` // free memory, bytes
	Disk         int64  `json:"disk"`   // free disk space, bytes

	NetDownMbps int `json:"net_down_mbps"` // Mbit/s; 0 if not measured
	NetUpMbps   int `json:"net_up_mbps"`   // Mbit/s; 0 if not measured

	GPUVendor   string       `json:"gpu_vendor"`
	CUDAVersion string       `json:"cuda_version"`
	GPUs        []MachineGPU `json:"gpus"`
	// BlenderScore is the OptiX score, or the CUDA one when there is none.
	BlenderScore int64 `json:"blender_score"`

	BasePrice      MicroUSD `json:"base_price"` // per hour of rent
	MinRentMinutes int      `json:"min_rent_minutes"`

	Reliability Reliability `json:"reliability"`
}

// MachineGPU is a GPU of a listed machine.
type MachineGPU struct {
	Model         string `json:"model"`
	MemTotalMB    int    `json:"mem_total_mb"`
	PCIeLinkGen   int    `json:"pcie_link_gen"`
	PCIeLinkWidth int    `json:"pcie_link_width"`
	// Clock ceilings in MHz: the one the node owner locked, otherwise the
	// card's maximum. Nil if unknown.
	CoreClock *int `json:"core_clock"`
	MemClock  *int `json:"mem_clock"`
	// PowerLimit in watts; nil if the card does not report one.
	PowerLimit *int `json:"power_limit"`
	// Tuned reports that the owner changed the power limit or locked clocks.
	Tuned bool `json:"tuned"`
}

// UnmarshalJSON decodes a GPU, tolerating the optional integers sent as the
// string "null" (or as numeric strings): some nodes, e.g. with Intel Arc
// cards, report missing values that way, and one such GPU must not break the
// whole listing.
func (g *MachineGPU) UnmarshalJSON(data []byte) error {
	type plain MachineGPU
	var raw struct {
		plain
		CoreClock  json.RawMessage `json:"core_clock"`
		MemClock   json.RawMessage `json:"mem_clock"`
		PowerLimit json.RawMessage `json:"power_limit"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*g = MachineGPU(raw.plain)
	var err error
	if g.CoreClock, err = optionalInt("core_clock", raw.CoreClock); err != nil {
		return err
	}
	if g.MemClock, err = optionalInt("mem_clock", raw.MemClock); err != nil {
		return err
	}
	g.PowerLimit, err = optionalInt("power_limit", raw.PowerLimit)
	return err
}

// optionalInt decodes a nullable integer that may also arrive as a string:
// absent, null, "null" and "" give nil; 42 and "42" give 42.
func optionalInt(field string, data json.RawMessage) (*int, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		return nil, nil
	}
	text := string(data)
	if data[0] == '"' {
		if err := json.Unmarshal(data, &text); err != nil {
			return nil, err
		}
		text = strings.TrimSpace(text)
		if text == "" || text == "null" {
			return nil, nil
		}
	}
	n, err := strconv.Atoi(text)
	if err != nil {
		return nil, fmt.Errorf("octaspace: %s: cannot parse %s as an integer", field, data)
	}
	return &n, nil
}

// Reliability describes how dependable a node is. The API documents it as an
// open object: Uptime is always present, the session statistics only when
// known. Fetch the listing with ListRaw for fields not modelled here.
type Reliability struct {
	// Uptime is the percent of time online over the last 30 days (since the
	// node first came online, for a newer node).
	Uptime float64 `json:"uptime"`

	// Session stability, present when known. The API does not document these
	// fields; they are passed through as reported.
	Stability float64 `json:"stability"`
	SNormal   int     `json:"s_normal"`
	SFailed   int     `json:"s_failed"`
	SAvgD     float64 `json:"s_avg_d"`
	SMaxD     float64 `json:"s_max_d"`
}
