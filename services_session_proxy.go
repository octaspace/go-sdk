package octaspace

import (
	"context"
	"net/url"
)

// SessionInfo holds detailed information about a running service session.
type SessionInfo struct {
	// VPNConfig contains the WireGuard/OpenVPN configuration for this session.
	// The JSON field name is "config" (verified against ruby-sdk and python-sdk).
	VPNConfig    string `json:"config"`
	TX           int64  `json:"tx"`
	RX           int64  `json:"rx"`
	ChargeAmount BigInt `json:"charge_amount"`
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
