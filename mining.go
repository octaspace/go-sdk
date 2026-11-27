package octaspace

import (
	"context"
	"net/url"
	"strconv"
)

// Mining roles: the values of [MiningParams.Role].
const (
	MiningGPU = "gpu" // GPU miners (the API default)
	MiningCPU = "cpu" // the CPU miner
)

// MiningParams select [StatsResource.Mining]. Zero values use the API
// defaults: the GPU role, 30 days.
type MiningParams struct {
	Role string // [MiningGPU] or [MiningCPU]
	Days int    // days back, today included: 1 to 365
}

// MiningUsage is what the platform's mining sessions of a role ran over the
// last days. Lists are ordered by Measure, most used first, "other" last.
type MiningUsage struct {
	Role    string `json:"role"`
	Days    int    `json:"days"`
	From    string `json:"from"`    // first day, YYYY-MM-DD
	To      string `json:"to"`      // last day, YYYY-MM-DD
	At      int64  `json:"at"`      // as of, Unix seconds; 0 if unknown
	Measure string `json:"measure"` // "gpu_hours" or "hours"

	Total      MiningTotal  `json:"total"`
	Miners     []MiningRow  `json:"miners"`
	Algorithms []MiningRow  `json:"algorithms"`
	Pools      []MiningPool `json:"pools"`
	Daily      MiningDaily  `json:"daily"`
}

// MiningTotal sums all mining sessions of the role.
type MiningTotal struct {
	Started  int     `json:"started"`
	Hours    float64 `json:"hours"`
	GPUHours float64 `json:"gpu_hours"`
}

// MiningRow is the use of a miner or an algorithm.
type MiningRow struct {
	Name  string `json:"name"`  // catalogue name, or "other"
	Label string `json:"label"` // display name; empty if unknown
	// Started counts the sessions that started with it, or restarted into it.
	Started  int     `json:"started"`
	Hours    float64 `json:"hours"`     // session hours
	GPUHours float64 `json:"gpu_hours"` // hours times GPUs; 0 for the CPU role
}

// MiningPool is the use of a mining pool and the coins mined there.
type MiningPool struct {
	MiningRow
	Coins []string `json:"coins"`
}

// MiningDaily is the use of each miner per day.
type MiningDaily struct {
	Dates []string `json:"dates"` // YYYY-MM-DD, one per day
	// Miners holds a value of Measure per date, by miner name.
	Miners map[string][]float64 `json:"miners"`
}

// Mining returns what the platform's mining sessions run: miners, algorithms
// and pools, and the miners per day. Days outside 1 to 365 is a
// [*ValidationError].
//
// GET /stats/mining
func (r *StatsResource) Mining(ctx context.Context, params *MiningParams) (*MiningUsage, error) {
	var m MiningUsage
	if err := r.c.getNullTolerant(ctx, "/stats/mining", miningQuery(params), &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// MiningRaw returns the raw JSON of GET /stats/mining.
func (r *StatsResource) MiningRaw(ctx context.Context, params *MiningParams) ([]byte, error) {
	return r.c.getRawParams(ctx, "/stats/mining", miningQuery(params))
}

func miningQuery(p *MiningParams) url.Values {
	q := url.Values{}
	if p == nil {
		return q
	}
	if p.Role != "" {
		q.Set("role", p.Role)
	}
	if p.Days > 0 {
		q.Set("days", strconv.Itoa(p.Days))
	}
	return q
}
