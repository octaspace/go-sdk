package octaspace

import (
	"context"
	"time"
)

// NetworkResource provides the statistics of the platform; no API key is
// needed.
// Obtain via [Client.Network].
type NetworkResource struct{ c *Client }

// NetworkInfo describes the platform as of its latest snapshot, taken every
// few minutes.
type NetworkInfo struct {
	At       int64           `json:"at"` // when the snapshot was taken, Unix seconds
	Nodes    NetworkNodes    `json:"nodes"`
	Power    NetworkPower    `json:"power"`
	Sessions NetworkSessions `json:"sessions"`
	Users    NetworkUsers    `json:"users"`
	Render   NetworkRender   `json:"render"`
	VPN      NetworkVPN      `json:"vpn"`
}

// Time returns At as a [time.Time].
func (n *NetworkInfo) Time() time.Time { return time.Unix(n.At, 0) }

// NetworkNodes counts the nodes of the platform.
type NetworkNodes struct {
	Online    int `json:"online"`
	Available int `json:"available"`
	Rented    int `json:"rented"`
	Countries int `json:"countries"`
}

// NetworkPower is the capacity of the online nodes.
type NetworkPower struct {
	CPUCores int     `json:"cpu_cores"`
	Memory   int64   `json:"memory"` // bytes
	Disk     int64   `json:"disk"`   // bytes
	GPUs     int     `json:"gpus"`
	TFLOPS   float64 `json:"tflops"`
}

// NetworkSessions counts service sessions.
type NetworkSessions struct {
	Running int `json:"running"`
	Total   int `json:"total"`
	Last24h int `json:"last_24h"`
}

// NetworkUsers counts the users of the platform.
type NetworkUsers struct {
	Total   int `json:"total"`
	Last24h int `json:"last_24h"`
}

// NetworkRender counts rendered frames.
type NetworkRender struct {
	FramesTotal  int64          `json:"frames_total"`
	Frames24h    int64          `json:"frames_24h"`
	FramesByWeek []WeeklyFrames `json:"frames_by_week"`
}

// WeeklyFrames is the number of frames rendered in a week.
type WeeklyFrames struct {
	Week   string `json:"week"` // first day of the week, YYYY-MM-DD
	Frames int64  `json:"frames"`
}

// NetworkVPN counts the VPN relays.
type NetworkVPN struct {
	Nodes     int `json:"nodes"`
	Countries int `json:"countries"`
}

// Info returns the platform statistics. It does not require authentication.
//
// GET /network
func (r *NetworkResource) Info(ctx context.Context) (*NetworkInfo, error) {
	var info NetworkInfo
	if err := r.c.get(ctx, "/network", nil, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// InfoRaw returns the raw JSON of GET /network.
func (r *NetworkResource) InfoRaw(ctx context.Context) ([]byte, error) {
	return r.c.getRaw(ctx, "/network")
}
