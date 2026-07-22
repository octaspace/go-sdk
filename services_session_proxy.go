package octaspace

import (
	"context"
	"encoding/json"
	"net/url"
)

// SessionInfo holds detailed information about a running service session.
type SessionInfo struct {
	UUID      string `json:"uuid"`
	AccountID int64  `json:"account_id"`
	Service   string `json:"service"`
	Kind      string `json:"kind"`
	AppName   string `json:"app_name"`
	AppLogo   string `json:"app_logo"`
	NodeID    int64  `json:"node_id"`

	Progress    string    `json:"progress"`
	IsReady     bool      `json:"is_ready"`
	Duration    FlexInt64 `json:"duration"`
	StartedAt   FlexInt64 `json:"started_at"`
	PublicIP    string    `json:"public_ip"`
	IP          string    `json:"ip"`
	ContainerID string    `json:"container_id"`
	Country     string    `json:"country"`
	CountryISO  string    `json:"country_iso"`
	City        string    `json:"city"`

	PortsMatrix map[string]FlexInt64 `json:"ports_matrix"`
	SSHDirect   SSHEndpoint          `json:"ssh_direct"`
	SSHProxy    SSHEndpoint          `json:"ssh_proxy"`
	SSHWeb      string               `json:"ssh_web"`
	URLs        map[string]string    `json:"urls"`
	UserEnvs    map[string]any       `json:"user_envs"`

	// VPNConfig contains the WireGuard/OpenVPN configuration for this session.
	// The live API uses "config"; the Swagger "vpn_config" alias is accepted
	// when decoding.
	VPNConfig         string    `json:"config"`
	TX                int64     `json:"tx"`
	RX                int64     `json:"rx"`
	ChargeAmount      BigInt    `json:"charge_amount"`
	TerminationReason FlexInt64 `json:"termination_reason"`

	Prices SessionPrices `json:"prices"`
	NodeHW SessionNodeHW `json:"node_hw"`
}

// SessionPrices contains the pricing metadata included in session details.
type SessionPrices struct {
	BaseUSD     int     `json:"base_usd"`
	CurrencyUSD bool    `json:"currency_usd"`
	MarketPrice float64 `json:"market_price"`
}

// SessionNodeHW describes the hardware used by a session.
type SessionNodeHW struct {
	CPU         string           `json:"cpu"`
	CPUCores    int              `json:"cpu_cores"`
	GPU         []SessionNodeGPU `json:"gpu"`
	TotalMemory int64            `json:"total_memory"`
	Disk        int64            `json:"disk"`
}

// SessionNodeGPU describes a GPU included in [SessionNodeHW].
type SessionNodeGPU struct {
	Model      string `json:"model"`
	MemTotalMB int    `json:"mem_total_mb"`
}

// UnmarshalJSON supports the documented vpn_config alias and numeric strings
// returned by some session endpoints for traffic counters.
func (s *SessionInfo) UnmarshalJSON(data []byte) error {
	type sessionInfo SessionInfo
	var raw struct {
		sessionInfo
		VPNConfigAlias string    `json:"vpn_config"`
		TXFlex         FlexInt64 `json:"tx"`
		RXFlex         FlexInt64 `json:"rx"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*s = SessionInfo(raw.sessionInfo)
	if s.VPNConfig == "" {
		s.VPNConfig = raw.VPNConfigAlias
	}
	s.TX = int64(raw.TXFlex)
	s.RX = int64(raw.RXFlex)
	return nil
}

// SessionLogs holds system and container log output for a session.
type SessionLogs struct {
	System []struct {
		TS  int64  `json:"ts"`
		Msg string `json:"msg"`
	} `json:"system"`
	Container string `json:"container"`
}

// LogsParams holds optional parameters for fetching session logs.
type LogsParams struct {
	// Recent must be set to true to fetch logs for finished (recent) sessions.
	// Active (current) sessions do not require this parameter.
	Recent *bool
}

// StopParams holds optional parameters for stopping a session.
type StopParams struct {
	Score *int `json:"score,omitempty"` // optional quality score (e.g. 1–5)
}

// SessionProxy provides operations on a specific service session.
// Obtain one via [ServicesResource.Session].
//
// Example:
//
//	proxy := client.Services.Session("550e8400-e29b-41d4-a716-446655440000")
//	info, err := proxy.Info(ctx)
type SessionProxy struct {
	c    *Client
	uuid string // URL-encoded session UUID
}

// Info returns detailed information about the session including VPN configuration
// and traffic statistics.
//
// GET /services/:uuid/info
func (s *SessionProxy) Info(ctx context.Context) (*SessionInfo, error) {
	var info SessionInfo
	return &info, s.c.get(ctx, "/services/"+s.uuid+"/info", nil, &info)
}

// InfoRaw returns the unmodified JSON body from the session-info endpoint.
// Use it for lossless passthrough when the server adds fields before an SDK
// release models them.
//
// GET /services/:uuid/info
func (s *SessionProxy) InfoRaw(ctx context.Context) ([]byte, error) {
	return s.c.getRaw(ctx, "/services/"+s.uuid+"/info")
}

// Logs returns system and container log output for the session.
// Pass nil params for active (current) sessions. For finished (recent) sessions,
// set recent := true and pass &LogsParams{Recent: &recent} — the API requires the recent=true
// query parameter to return logs for sessions that are no longer running.
//
// GET /services/:uuid/logs
// GET /services/:uuid/logs?recent=true
func (s *SessionProxy) Logs(ctx context.Context, params *LogsParams) (*SessionLogs, error) {
	qp := url.Values{}
	if params != nil && params.Recent != nil && *params.Recent {
		qp.Set("recent", "true")
	}
	var logs SessionLogs
	return &logs, s.c.get(ctx, "/services/"+s.uuid+"/logs", qp, &logs)
}

// Stop terminates the session. Pass nil params to stop without sending a body.
// Pass a non-nil [*StopParams] to include optional fields such as a quality score.
//
// POST /services/:uuid/stop
func (s *SessionProxy) Stop(ctx context.Context, params *StopParams) error {
	return s.c.post(ctx, "/services/"+s.uuid+"/stop", params, nil)
}
