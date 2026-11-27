package octaspace

import (
	"context"
	"encoding/json"
	"fmt"
)

// ComputeApp is an application to start a compute session from: an official
// one, or one of the account's own (Personal). Pass its UUID as
// [ComputeStartParams.App] and its Image as [ComputeStartParams.Image].
type ComputeApp struct {
	UUID     string `json:"uuid"`
	Name     string `json:"name"`
	Image    string `json:"image"` // e.g. "ubuntu:24.04"
	Personal bool   `json:"personal"`

	Category    string `json:"category"` // e.g. "Machine Learning"; empty if none
	Description string `json:"description"`
	Entrypoint  string `json:"entrypoint"`
	Ports       []int  `json:"ports"`      // TCP/UDP ports the app exposes
	HTTPPorts   []int  `json:"http_ports"` // HTTP ports the app exposes

	Extra AppExtra `json:"extra"`

	// StartupScript and Envs are set on the account's own apps only.
	StartupScript string            `json:"startup_script"`
	Envs          map[string]string `json:"envs"`
}

// AppExtra are the extra settings of an app.
type AppExtra struct {
	Logo string `json:"logo"`
	// MinDiskSize is the disk the app needs, GB. It is advice only: a
	// session with a smaller DiskSize starts, and the app may run out of
	// space.
	MinDiskSize int `json:"min_disk_size"`
}

// Apps returns the applications to start a session from: the official ones
// and, with an API key, the account's own. Authentication is optional.
//
// GET /compute/apps
func (r *ComputeResource) Apps(ctx context.Context) ([]ComputeApp, error) {
	body, err := r.c.getRaw(ctx, "/compute/apps")
	if err != nil {
		return nil, err
	}
	var apps []ComputeApp
	if err := json.Unmarshal(body, &apps); err != nil {
		return nil, fmt.Errorf("octaspace: decode response: %w", err)
	}
	for i := range apps {
		if apps[i].Category == "null" {
			apps[i].Category = ""
		}
		if apps[i].Description == "null" {
			apps[i].Description = ""
		}
	}
	return apps, nil
}

// AppsRaw returns the raw JSON of GET /compute/apps.
func (r *ComputeResource) AppsRaw(ctx context.Context) ([]byte, error) {
	return r.c.getRaw(ctx, "/compute/apps")
}
