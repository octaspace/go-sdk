package octaspace

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// App represents an application available for deployment.
type App struct {
	UUID         string         `json:"uuid"`
	Name         string         `json:"name"`
	Image        string         `json:"image"`
	Category     string         `json:"category"`
	Description  string         `json:"description"`
	StartCommand string         `json:"start_command"`
	Envs         map[string]any `json:"envs"`
	Ports        []int          `json:"ports"`
	HTTPPorts    []int          `json:"http_ports"`
	Personal     bool           `json:"personal"`
	Extra        map[string]any `json:"extra"`
}

// UnmarshalJSON accepts both the documented array form and the live string-encoded
// form used by /apps for ports/http_ports.
func (a *App) UnmarshalJSON(data []byte) error {
	type rawApp struct {
		UUID         string          `json:"uuid"`
		Name         string          `json:"name"`
		Image        string          `json:"image"`
		Category     string          `json:"category"`
		Description  string          `json:"description"`
		StartCommand string          `json:"start_command"`
		Envs         map[string]any  `json:"envs"`
		Ports        json.RawMessage `json:"ports"`
		HTTPPorts    json.RawMessage `json:"http_ports"`
		Personal     bool            `json:"personal"`
		Extra        map[string]any  `json:"extra"`
	}

	var raw rawApp
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	ports, err := decodeAppPorts(raw.Ports)
	if err != nil {
		return fmt.Errorf("decode ports: %w", err)
	}
	httpPorts, err := decodeAppPorts(raw.HTTPPorts)
	if err != nil {
		return fmt.Errorf("decode http_ports: %w", err)
	}

	*a = App{
		UUID:         raw.UUID,
		Name:         raw.Name,
		Image:        raw.Image,
		Category:     raw.Category,
		Description:  raw.Description,
		StartCommand: raw.StartCommand,
		Envs:         raw.Envs,
		Ports:        ports,
		HTTPPorts:    httpPorts,
		Personal:     raw.Personal,
		Extra:        raw.Extra,
	}
	return nil
}

func decodeAppPorts(raw json.RawMessage) ([]int, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	var ports []int
	if err := json.Unmarshal(raw, &ports); err == nil {
		return ports, nil
	}

	var encoded string
	if err := json.Unmarshal(raw, &encoded); err != nil {
		return nil, err
	}

	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return nil, nil
	}
	if err := json.Unmarshal([]byte(encoded), &ports); err != nil {
		return nil, err
	}
	return ports, nil
}

// AppsResource provides access to the application catalog endpoint.
// Obtain via [Client.Apps].
type AppsResource struct{ c *Client }

// List returns all available applications.
//
// GET /apps
func (r *AppsResource) List(ctx context.Context) ([]App, error) {
	var apps []App
	return apps, r.c.get(ctx, "/apps", nil, &apps)
}

// ListRaw returns the unmodified JSON body from the applications endpoint.
// Use it for lossless JSON passthrough.
//
// GET /apps
func (r *AppsResource) ListRaw(ctx context.Context) ([]byte, error) {
	return r.c.getRaw(ctx, "/apps")
}
