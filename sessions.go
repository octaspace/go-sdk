package octaspace

import (
	"context"
	"net/url"
)

// SSHEndpoint describes an SSH connection endpoint attached to a session.
type SSHEndpoint struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// Session represents an active or recent service session.
//
// Note: the /sessions and /sessions?recent=true endpoints return overlapping
// but not identical field sets. Fields absent in a response are zero-valued.
type Session struct {
	UUID      string `json:"uuid"`
	AccountID int64  `json:"account_id"`
	Service   string `json:"service"`
	Kind      string `json:"kind"`
	AppName   string `json:"app_name"`
	AppLogo   string `json:"app_logo"`
	NodeID    int64  `json:"node_id"`

	// Runtime state (present in active sessions)
	Progress   string `json:"progress"`
	IsReady    bool   `json:"is_ready"`
	PublicIP   string `json:"public_ip"`
	IP         string `json:"ip"`
	Country    string `json:"country"`
	CountryISO string `json:"country_iso"`
	City       string `json:"city"`

	// ContainerID is absent for VPN sessions, and returned as a JSON array of
	// container IDs (joined here with ", ") for compute (MR) sessions.
	ContainerID FlexString `json:"container_id"`

	// Timing — returned as a bare integer or a quoted string depending on endpoint;
	// FlexInt64 handles both transparently.
	Duration   FlexInt64 `json:"duration"`
	StartedAt  FlexInt64 `json:"started_at"`
	FinishedAt FlexInt64 `json:"finished_at"`

	// Network counters may be returned as JSON numbers or quoted strings.
	RX FlexInt64 `json:"rx"`
	TX FlexInt64 `json:"tx"`

	// Billing
	ChargeAmount      BigInt `json:"charge_amount"`
	TerminationReason int    `json:"termination_reason"`

	SSHDirect SSHEndpoint `json:"ssh_direct"`
	SSHProxy  SSHEndpoint `json:"ssh_proxy"`
	SSHWeb    string      `json:"ssh_web"`

	PortsMatrix map[string]FlexInt64 `json:"ports_matrix"`
	URLs        map[string]string    `json:"urls"`
	UserEnvs    map[string]any       `json:"user_envs"`

	Prices struct {
		BaseUSD     int     `json:"base_usd"`
		CurrencyUSD bool    `json:"currency_usd"`
		MarketPrice float64 `json:"market_price"`
	} `json:"prices"`

	NodeHW struct {
		CPU      string `json:"cpu"`
		CPUCores int    `json:"cpu_cores"`
		GPU      []struct {
			Model      string `json:"model"`
			MemTotalMB int    `json:"mem_total_mb"`
		} `json:"gpu"`
		TotalMemory int64 `json:"total_memory"`
		Disk        int64 `json:"disk"`
	} `json:"node_hw"`
}

// ListSessionsParams contains optional filter parameters for [SessionsResource.List].
type ListSessionsParams struct {
	Recent *bool // set to true to include only recent sessions
}

// SessionsResource provides access to the sessions listing endpoint.
// Obtain via [Client.Sessions].
type SessionsResource struct{ c *Client }

// List returns active and recent sessions. Pass nil params to list all sessions.
//
// GET /sessions
func (r *SessionsResource) List(ctx context.Context, params *ListSessionsParams) ([]Session, error) {
	qp := sessionListValues(params)
	var sessions []Session
	return sessions, r.c.get(ctx, "/sessions", qp, &sessions)
}

// ListRaw returns the unmodified JSON body from the sessions endpoint.
//
// GET /sessions
func (r *SessionsResource) ListRaw(ctx context.Context, params *ListSessionsParams) ([]byte, error) {
	return r.c.getRawParams(ctx, "/sessions", sessionListValues(params))
}

func sessionListValues(params *ListSessionsParams) url.Values {
	qp := url.Values{}
	if params != nil && params.Recent != nil && *params.Recent {
		qp.Set("recent", "true")
	}
	return qp
}
