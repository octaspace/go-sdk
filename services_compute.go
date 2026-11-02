package octaspace

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/url"
)

// ComputeResource rents machines and manages compute sessions.
// Obtain via [Client.Compute].
type ComputeResource struct{ c *Client }

// List returns the machines available for rent.
//
// Authentication is optional: anonymous requests get a listing refreshed
// periodically (hourly by default), authenticated ones a listing at most 10
// seconds old. Create the client with an API key for an up-to-date listing.
//
// GET /services/compute
func (r *ComputeResource) List(ctx context.Context) ([]ComputeMachine, error) {
	var result []ComputeMachine
	return result, r.c.get(ctx, "/services/compute", nil, &result)
}

// ListRaw returns the raw JSON of GET /services/compute, for fields the SDK
// does not model yet.
func (r *ComputeResource) ListRaw(ctx context.Context) ([]byte, error) {
	return r.c.getRaw(ctx, "/services/compute")
}

// ComputeStartParams describes a compute session to start.
// NodeNumber and Image are required; every other field is optional and is
// left out of the request when zero.
type ComputeStartParams struct {
	// NodeNumber of a machine from [ComputeResource.List]. Case, dashes and
	// spaces are ignored; O reads as 0, I and L as 1.
	NodeNumber string `json:"node_number"`
	// Image is the container image, e.g. "ubuntu:24.04".
	Image string `json:"image"`
	// App is an application UUID. Its settings decide SSH and FM when those
	// are not set here.
	App string `json:"app,omitempty"`

	// DiskSize to allocate, GB (the API default is 1).
	DiskSize int `json:"disk_size,omitempty"`
	// Envs are user-defined variables: up to 64, values up to 4 KiB.
	Envs map[string]string `json:"envs,omitempty"`
	// Ports are TCP/UDP ports to expose; 0 and 22 (SSH is always exposed)
	// are ignored.
	Ports []int `json:"ports,omitempty"`
	// HTTPPorts are HTTP ports to expose; 7681 and 7682 are reserved.
	HTTPPorts []int `json:"http_ports,omitempty"`

	// Entrypoint replaces the image's ENTRYPOINT (one line, up to 4 KiB) and
	// drops its CMD. It is split like a shell command line, but nothing is
	// expanded: use "sh -c" for variables, globs or "&&".
	Entrypoint string `json:"entrypoint,omitempty"`
	// Args replace the image's CMD (one line, up to 4 KiB, split like
	// Entrypoint).
	Args string `json:"args,omitempty"`
	// StartupScript runs as root inside the container once it is up, in the
	// background, alongside the container's command (up to 64 KiB).
	StartupScript string `json:"startup_script,omitempty"`

	// SSH sets up a web terminal and SSH access. Nil leaves it to the app's
	// setting (off without an app); use [Bool] to set it explicitly.
	SSH *bool `json:"ssh,omitempty"`
	// FM starts a web file manager without a login. Nil leaves it to the
	// app's setting (off without an app); use [Bool] to set it explicitly.
	FM *bool `json:"fm,omitempty"`
}

// Bool returns a pointer to v, for the optional *bool fields of request
// parameters such as [ComputeStartParams.SSH].
func Bool(v bool) *bool { return &v }

// ComputeBatchItem is one request of [ComputeResource.StartBatch]. ID is
// chosen by the caller and echoed in the matching [BatchResult].
type ComputeBatchItem struct {
	ID string `json:"id"`
	ComputeStartParams
}

// BatchResult is the outcome of one request of a batch start, in request order.
type BatchResult struct {
	ID     string // the caller's request ID
	UUID   string // session UUID when started
	Reason string // why the request failed
	Status int    // 0 - started, 1 - failed
}

// OK reports whether the request started a session.
func (b BatchResult) OK() bool { return b.Status == 0 && b.UUID != "" }

// UnmarshalJSON accepts the request ID as a JSON string or number.
func (b *BatchResult) UnmarshalJSON(data []byte) error {
	var raw struct {
		ID     json.RawMessage `json:"id"`
		UUID   string          `json:"uuid"`
		Reason string          `json:"reason"`
		Status int             `json:"status"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	id := bytes.TrimSpace(raw.ID)
	switch {
	case len(id) == 0 || string(id) == "null":
	case id[0] == '"':
		if err := json.Unmarshal(id, &b.ID); err != nil {
			return err
		}
	default:
		b.ID = string(id)
	}
	b.UUID, b.Reason, b.Status = raw.UUID, raw.Reason, raw.Status
	return nil
}

// serviceUUID is the response of a service start, restart or stop.
type serviceUUID struct {
	UUID string `json:"uuid"`
}

// Start starts one compute session and returns its UUID. A request the API
// rejects (insufficient balance, node not available, invalid parameters)
// returns a [*ValidationError] with the reason.
//
// The session is billed from the start; a session stopped before the
// machine's MinRentMinutes is charged base and storage for that long.
// Starting is asynchronous: poll the session info until it is ready.
// Start is never retried automatically.
//
// POST /services/compute
func (r *ComputeResource) Start(ctx context.Context, params *ComputeStartParams) (string, error) {
	if params == nil {
		return "", errors.New("octaspace: compute start params are required")
	}
	var resp serviceUUID
	if err := r.c.post(ctx, "/services/compute", params, &resp); err != nil {
		return "", err
	}
	return resp.UUID, nil
}

// StartBatch starts several compute sessions in one request (at most 10 by
// default). Each request is validated on its own: a failed one is reported in
// its [BatchResult] and does not stop the others.
//
// POST /services/compute
func (r *ComputeResource) StartBatch(ctx context.Context, items []ComputeBatchItem) ([]BatchResult, error) {
	if len(items) == 0 {
		return nil, errors.New("octaspace: compute batch is empty")
	}
	var results []BatchResult
	return results, r.c.post(ctx, "/services/compute", items, &results)
}

// ComputeRestartParams are the parameters a restarted session runs with.
// The node, the image and the app cannot be changed.
//
// They REPLACE the session's parameters as a whole, they are not merged:
// a field left zero falls back to the default, not to the value the session
// was started with. The defaults are:
//   - DiskSize: 1 GB, NOT the session's current disk size;
//   - Entrypoint, Args: the image's own;
//   - StartupScript: none;
//   - Envs, Ports, HTTPPorts: none;
//   - SSH, FM: the app's setting, off without an app.
//
// Clearing Entrypoint while keeping Args, for example, turns Args into the
// command and can stop the session. Pass every parameter the session needs,
// or nil params to restart with the original ones. The API does not expose
// Entrypoint, Args, StartupScript or DiskSize of a running session (session
// info shows only envs, ports and SSH/FM as they are now), so keep the start
// parameters yourself if you need to restart with changes.
//
// Zero fields are left out of the request, so params with every field zero
// are sent as {}, which the API treats like nil: the original parameters are
// kept. A restart cannot reset everything to the defaults.
type ComputeRestartParams struct {
	DiskSize      int               `json:"disk_size,omitempty"`
	Envs          map[string]string `json:"envs,omitempty"`
	Ports         []int             `json:"ports,omitempty"`
	HTTPPorts     []int             `json:"http_ports,omitempty"`
	Entrypoint    string            `json:"entrypoint,omitempty"`
	Args          string            `json:"args,omitempty"`
	StartupScript string            `json:"startup_script,omitempty"`
	SSH           *bool             `json:"ssh,omitempty"`
	FM            *bool             `json:"fm,omitempty"`
}

// Restart recreates the container of a running compute session on the same
// node and returns the session UUID. Nil params (or params with every field
// zero) restart it with the parameters it was started with; other params
// replace them as a whole (see [ComputeRestartParams]).
//
// Restarting is asynchronous: poll the session info for its progress. If the
// new parameters cannot be set up, the session stops. A session can be
// restarted once in 5 minutes and 6 times an hour: an earlier attempt returns
// a [*RateLimitError] whose RetryAfter says how long to wait, and one made
// while the session is still starting or restarting a [*ConflictError].
//
// POST /services/compute/{uuid}/restart
func (r *ComputeResource) Restart(ctx context.Context, uuid string, params *ComputeRestartParams) (string, error) {
	if uuid == "" {
		return "", errors.New("octaspace: session uuid is required")
	}
	var body any
	if params != nil {
		body = params
	}
	var resp serviceUUID
	if err := r.c.post(ctx, "/services/compute/"+url.PathEscape(uuid)+"/restart", body, &resp); err != nil {
		return "", err
	}
	return resp.UUID, nil
}
