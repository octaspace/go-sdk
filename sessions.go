package octaspace

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// Service kinds reported in [Session.Service]. Sessions of another kind
// decode as [Session] too.
const (
	ServiceCompute = "compute"
	ServiceRender  = "render"
)

// Session statuses reported in [Session.Status].
const (
	SessionStarting = "starting" // being configured (pulling the image, setting up access, ...)
	SessionReady    = "ready"
)

// Session is a running service session, as returned by GET /sessions and
// GET /services/{uuid}/info. Compute and render sessions share this type and
// are told apart by Service; the compute-only fields stay zero for render.
//
// Many fields are unknown (zero or nil) while Status is [SessionStarting].
type Session struct {
	UUID    string `json:"uuid"`
	Service string `json:"service"` // [ServiceCompute] or [ServiceRender]
	Status  string `json:"status"`  // [SessionStarting] or [SessionReady]

	// Progress is the current configuration step, e.g. "Service configured".
	Progress string `json:"progress"`
	// PullPercent is the image pull progress (an estimate until every layer
	// has started downloading); nil when no image is being pulled.
	PullPercent *int `json:"pull_percent"`

	Duration  int64 `json:"duration"`   // seconds since start
	StartedAt int64 `json:"started_at"` // Unix time, milliseconds; 0 if unknown

	NodeNumber string `json:"node_number"`
	AppName    string `json:"app_name"`
	AppLogo    string `json:"app_logo"`

	// Prices fixed at start; zero while starting. Render bills BasePrice only.
	BasePrice      MicroUSD `json:"base_price"`    // per hour
	StoragePrice   MicroUSD `json:"storage_price"` // per GB of disk per hour (compute)
	TrafficPrice   MicroUSD `json:"traffic_price"` // per GB of traffic (compute)
	MinRentMinutes int      `json:"min_rent_minutes"`
	ChargeAmount   MicroUSD `json:"charge_amount"` // charged so far

	NodeHW *SessionNodeHW `json:"node_hw"` // node hardware at start; nil while starting

	// Compute only.
	City       string            `json:"city"`
	Country    string            `json:"country"`
	CountryISO string            `json:"country_iso"`
	PublicIP   string            `json:"public_ip"`
	Ports      map[string]int    `json:"ports"` // container port => public port on the node
	URLs       map[string]string `json:"urls"`  // HTTP port => public URL
	SSH        SessionSSH        `json:"ssh"`
	FM         string            `json:"fm"` // web file manager URL; empty if not running
	Envs       map[string]string `json:"envs"`
	RX         int64             `json:"rx"` // bytes received
	TX         int64             `json:"tx"` // bytes sent
}

// IsCompute reports whether s is a compute session.
func (s *Session) IsCompute() bool { return s.Service == ServiceCompute }

// IsReady reports whether the session has finished configuring.
func (s *Session) IsReady() bool { return s.Status == SessionReady }

// StartedTime returns StartedAt as a [time.Time], or the zero time if unknown.
func (s *Session) StartedTime() time.Time {
	if s.StartedAt == 0 {
		return time.Time{}
	}
	return time.UnixMilli(s.StartedAt)
}

// SessionSSH is the access to a compute session. Direct and Proxy are nil and
// Web is empty when SSH was not requested or is not set up yet.
type SessionSSH struct {
	Direct *SSHEndpoint `json:"direct"` // straight to the node
	Proxy  *SSHEndpoint `json:"proxy"`  // through the platform proxy
	Web    string       `json:"web"`    // web terminal URL
}

// SSHEndpoint is where to connect with SSH.
type SSHEndpoint struct {
	Host string `json:"host"`
	Port int    `json:"port"`
	// User to log in as: the image's USER, root when it sets none.
	User string `json:"user"`
}

// SessionNodeHW is the hardware of the node a session runs on, as of its
// start. Render sessions report CPUCores, TotalMemory and GPUs only.
type SessionNodeHW struct {
	CPU          string           `json:"cpu"`
	CPUCores     int              `json:"cpu_cores"`
	TotalMemory  int64            `json:"total_memory"` // bytes
	MemorySpeed  int              `json:"memory_speed"` // MT/s
	MemoryType   string           `json:"memory_type"`
	Disk         int              `json:"disk"`      // rented disk, GB
	DiskType     string           `json:"disk_type"` // "nvme", "ssd", "hdd"; empty if unknown
	DiskSpeedMBS float64          `json:"disk_speed_mbs"`
	GPUVendor    string           `json:"gpu_vendor"`
	GPUs         []SessionNodeGPU `json:"gpus"`
	// Network speed in Mbit/s; 0 if unknown.
	NetDownMbps int `json:"net_down_mbps"`
	NetUpMbps   int `json:"net_up_mbps"`
}

// SessionNodeGPU is a GPU of the node a session runs on.
type SessionNodeGPU struct {
	Model      string `json:"model"`
	MemTotalMB int    `json:"mem_total_mb"`
}

// SessionLogs are the logs of a running compute session.
type SessionLogs struct {
	// System holds the platform events of the session, oldest first.
	System []SystemLogEvent `json:"system"`
	// Container is the latest container output as `docker logs` gives it, up
	// to 1 MiB from the end. Nil when the node cannot be reached, which is
	// not the same as an empty log.
	Container *string `json:"container"`
	// ContainerTruncated reports that Container was cut to its last 1 MiB,
	// starting at a whole line.
	ContainerTruncated bool `json:"container_truncated"`
}

// SystemLogEvent is a platform event of a session.
type SystemLogEvent struct {
	TS  int64  `json:"ts"` // Unix time, milliseconds; 0 if unknown
	Msg string `json:"msg"`
}

// Time returns TS as a [time.Time], or the zero time if unknown.
func (e SystemLogEvent) Time() time.Time {
	if e.TS == 0 {
		return time.Time{}
	}
	return time.UnixMilli(e.TS)
}

// LogsParams are the optional parameters of [SessionsResource.Logs].
type LogsParams struct {
	// Tail is the number of the latest container log lines, 1..1000.
	// Zero uses the API default (200).
	Tail int
}

// SessionsResource lists and manages the account's running sessions.
// Obtain via [Client.Sessions].
type SessionsResource struct{ c *Client }

// List returns the account's running sessions, oldest first. Stopped
// sessions are not listed.
//
// GET /sessions
func (r *SessionsResource) List(ctx context.Context) ([]Session, error) {
	var result []Session
	return result, r.c.get(ctx, "/sessions", nil, &result)
}

// ListRaw returns the raw JSON of GET /sessions.
func (r *SessionsResource) ListRaw(ctx context.Context) ([]byte, error) {
	return r.c.getRaw(ctx, "/sessions")
}

// Get returns a running session. A stopped or unknown session (or one of
// another account) is a [*NotFoundError]; a malformed UUID a
// [*ValidationError].
//
// GET /services/{uuid}/info
func (r *SessionsResource) Get(ctx context.Context, uuid string) (*Session, error) {
	if uuid == "" {
		return nil, errors.New("octaspace: session uuid is required")
	}
	var s Session
	if err := r.c.get(ctx, sessionPath(uuid, "info"), nil, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// GetRaw returns the raw JSON of GET /services/{uuid}/info.
func (r *SessionsResource) GetRaw(ctx context.Context, uuid string) ([]byte, error) {
	if uuid == "" {
		return nil, errors.New("octaspace: session uuid is required")
	}
	return r.c.getRaw(ctx, sessionPath(uuid, "info"))
}

// Logs returns the system log and the latest container output of a running
// compute session. A stopped session is a [*NotFoundError]: its logs are no
// longer available.
//
// GET /services/{uuid}/logs
func (r *SessionsResource) Logs(ctx context.Context, uuid string, params *LogsParams) (*SessionLogs, error) {
	if uuid == "" {
		return nil, errors.New("octaspace: session uuid is required")
	}
	var q url.Values
	if params != nil && params.Tail != 0 {
		q = url.Values{"tail": {strconv.Itoa(params.Tail)}}
	}
	var logs SessionLogs
	if err := r.c.get(ctx, sessionPath(uuid, "logs"), q, &logs); err != nil {
		return nil, err
	}
	return &logs, nil
}

// Stop stops a session and returns its UUID. Stopping is asynchronous: the
// session may still be reported for a few seconds, until Get returns a
// [*NotFoundError]. Stopping a session that is already stopping succeeds
// again. A session stopped before its MinRentMinutes is charged base and
// storage for that long.
//
// POST /services/{uuid}/stop
func (r *SessionsResource) Stop(ctx context.Context, uuid string) (string, error) {
	if uuid == "" {
		return "", errors.New("octaspace: session uuid is required")
	}
	var resp serviceUUID
	if err := r.c.post(ctx, sessionPath(uuid, "stop"), nil, &resp); err != nil {
		return "", err
	}
	return resp.UUID, nil
}

// ErrSessionStopped is returned by [SessionsResource.WaitReady] when the
// session disappears while being waited for: it stopped, for example because
// its configuration failed. It wraps the [*NotFoundError] of the API.
var ErrSessionStopped = errors.New("octaspace: session stopped before it became ready")

// defaultWaitInterval is how often WaitReady polls by default.
const defaultWaitInterval = 3 * time.Second

// WaitOptions configure [SessionsResource.WaitReady].
type WaitOptions struct {
	// Interval between polls; zero means 3 seconds.
	Interval time.Duration
	// OnProgress, if set, is called with every polled state, e.g. to show
	// Progress and PullPercent.
	OnProgress func(*Session)
}

// WaitReady polls the session until it is ready and returns it. It returns
// [ErrSessionStopped] (wrapping the [*NotFoundError]) if the session stops
// meanwhile, and the context's error when ctx is done: pass a context with a
// deadline to bound the wait. The client timeout applies to each poll, not to
// the whole wait.
func (r *SessionsResource) WaitReady(ctx context.Context, uuid string, opts *WaitOptions) (*Session, error) {
	interval := defaultWaitInterval
	var onProgress func(*Session)
	if opts != nil {
		if opts.Interval > 0 {
			interval = opts.Interval
		}
		onProgress = opts.OnProgress
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		s, err := r.Get(ctx, uuid)
		switch {
		case IsNotFound(err):
			return nil, fmt.Errorf("%w: %w", ErrSessionStopped, err)
		case err != nil:
			return nil, err
		}
		if onProgress != nil {
			onProgress(s)
		}
		if s.IsReady() {
			return s, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

// sessionPath returns /services/{uuid}/{action} with the UUID path-escaped.
func sessionPath(uuid, action string) string {
	return "/services/" + url.PathEscape(uuid) + "/" + action
}
