package octaspace

import (
	"context"
	"net/url"
	"strconv"
	"time"
)

// NetworkHistoryParams select the range of [NetworkResource.History]. Zero
// values use the API defaults: the last day, a point per hour.
type NetworkHistoryParams struct {
	Start time.Time
	End   time.Time
	// Step between points, at least 5 minutes; a range has at most 2000
	// points.
	Step time.Duration
}

// NetworkHistory is the platform over time as parallel series: value i of
// every series belongs to Timestamps[i]. Values are averages over the step,
// so they can be fractional; nil where there is no value. Use
// [NetworkHistory.Points] for one value per point.
type NetworkHistory struct {
	// Timestamps are the starts of the steps, Unix seconds. Steps without
	// data are left out.
	Timestamps      []int64    `json:"timestamps"`
	NodesOnline     []*float64 `json:"nodes_online"`
	NodesAvailable  []*float64 `json:"nodes_available"`
	NodesRented     []*float64 `json:"nodes_rented"`
	Countries       []*float64 `json:"countries"` // of the online nodes
	GPUs            []*float64 `json:"gpus"`
	GPUsRented      []*float64 `json:"gpus_rented"`
	TFLOPS          []*float64 `json:"tflops"`    // FP32 TFLOPS of the online GPUs
	VPNNodes        []*float64 `json:"vpn_nodes"` // nodes VPN is offered on
	Users           []*float64 `json:"users"`     // confirmed users
	SessionsRunning []*float64 `json:"sessions_running"`
	Sessions24h     []*float64 `json:"sessions_24h"` // sessions started in the 24 hours before
}

// NetworkHistoryPoint is the platform in one step of [NetworkHistory].
type NetworkHistoryPoint struct {
	Time            time.Time
	NodesOnline     *float64
	NodesAvailable  *float64
	NodesRented     *float64
	Countries       *float64
	GPUs            *float64
	GPUsRented      *float64
	TFLOPS          *float64
	VPNNodes        *float64
	Users           *float64
	SessionsRunning *float64
	Sessions24h     *float64
}

// Points returns the history as one value per timestamp.
func (h NetworkHistory) Points() []NetworkHistoryPoint {
	at := func(series []*float64, i int) *float64 {
		if i < len(series) {
			return series[i]
		}
		return nil
	}
	points := make([]NetworkHistoryPoint, len(h.Timestamps))
	for i, ts := range h.Timestamps {
		points[i] = NetworkHistoryPoint{
			Time:            time.Unix(ts, 0),
			NodesOnline:     at(h.NodesOnline, i),
			NodesAvailable:  at(h.NodesAvailable, i),
			NodesRented:     at(h.NodesRented, i),
			Countries:       at(h.Countries, i),
			GPUs:            at(h.GPUs, i),
			GPUsRented:      at(h.GPUsRented, i),
			TFLOPS:          at(h.TFLOPS, i),
			VPNNodes:        at(h.VPNNodes, i),
			Users:           at(h.Users, i),
			SessionsRunning: at(h.SessionsRunning, i),
			Sessions24h:     at(h.Sessions24h, i),
		}
	}
	return points
}

// History returns the platform statistics over time; no API key is needed.
// An invalid range (Step under 5 minutes, too many points, Start after End)
// is a [*ValidationError].
//
// GET /network/history
func (r *NetworkResource) History(ctx context.Context, params *NetworkHistoryParams) (*NetworkHistory, error) {
	var h NetworkHistory
	if err := r.c.getNullTolerant(ctx, "/network/history", networkHistoryQuery(params), &h); err != nil {
		return nil, err
	}
	return &h, nil
}

// HistoryRaw returns the raw JSON of GET /network/history.
func (r *NetworkResource) HistoryRaw(ctx context.Context, params *NetworkHistoryParams) ([]byte, error) {
	return r.c.getRawParams(ctx, "/network/history", networkHistoryQuery(params))
}

func networkHistoryQuery(p *NetworkHistoryParams) url.Values {
	q := url.Values{}
	if p == nil {
		return q
	}
	if !p.Start.IsZero() {
		q.Set("start", strconv.FormatInt(p.Start.Unix(), 10))
	}
	if !p.End.IsZero() {
		q.Set("end", strconv.FormatInt(p.End.Unix(), 10))
	}
	if p.Step > 0 {
		q.Set("step", strconv.FormatInt(int64(p.Step/time.Second), 10))
	}
	return q
}
