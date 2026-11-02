package octaspace

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Verified filters GPU statistics by node verification.
type Verified string

// Values of [Verified].
const (
	VerifiedAll Verified = "all" // all GPUs (the API default)
	VerifiedYes Verified = "yes" // GPUs of verified nodes
	VerifiedNo  Verified = "no"  // GPUs of the other nodes
)

// StatsResource provides the public GPU market statistics. The endpoints do
// not require authentication. Obtain via [Client.Stats].
type StatsResource struct{ c *Client }

// Percentiles are price percentiles per GPU-hour. A percentile is nil when
// there is no price to compute it from.
type Percentiles struct {
	P10 *MicroUSD `json:"p10"`
	P25 *MicroUSD `json:"p25"`
	P50 *MicroUSD `json:"p50"`
	P75 *MicroUSD `json:"p75"`
	P90 *MicroUSD `json:"p90"`
}

// GPUMarket is the GPU market as of the latest snapshot.
type GPUMarket struct {
	At   int64           `json:"at"`   // when the snapshot was taken, Unix seconds
	GPUs []GPUModelStats `json:"gpus"` // most GPUs first
}

// GPUModelStats is the market of one GPU model. Counts are of GPUs, not
// nodes: a node of four cards counts four, rented together.
type GPUModelStats struct {
	Model     string   `json:"model"`
	Total     int      `json:"total"`
	Available int      `json:"available"`
	Rented    int      `json:"rented"`
	Usage     float64  `json:"usage"`     // percent rented now
	Usage30d  *float64 `json:"usage_30d"` // percent rented over 30 days; nil if unknown
	AIScore   int64    `json:"ai_score"`
	TFLOPS    *float64 `json:"tflops"`
	// TFLOPSPerDollar is nil when there is no price to divide by.
	TFLOPSPerDollar *float64 `json:"tflops_per_dollar"`
	// Prices are only of nodes whose GPUs are all of this model.
	AvailablePrice Percentiles `json:"available_price"` // listing prices
	RentedPrice    Percentiles `json:"rented_price"`    // what renters paid
}

// GPUStatsParams filter [StatsResource.GPUCurrent].
type GPUStatsParams struct {
	Verified Verified // empty means all
}

// GPUCurrent returns the GPU market per model, as of the latest snapshot
// (taken every few minutes).
//
// GET /stats/gpu/current
func (r *StatsResource) GPUCurrent(ctx context.Context, params *GPUStatsParams) (*GPUMarket, error) {
	var q url.Values
	if params != nil && params.Verified != "" {
		q = url.Values{"verified": {string(params.Verified)}}
	}
	var m GPUMarket
	if err := r.c.getNullTolerant(ctx, "/stats/gpu/current", q, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// GPUHistoryParams select the range of [StatsResource.GPUHistory]. Zero
// values use the API defaults.
type GPUHistoryParams struct {
	GPUNames []string // GPU models; empty means all
	Verified Verified // empty means all
	Start    time.Time
	End      time.Time     // zero means now; Start defaults to a day before End
	Step     time.Duration // at least 5 minutes, at most 2000 points; zero means an hour
}

// GPUHistory is the market of one GPU model over time, as parallel series:
// value i of every series belongs to Timestamps[i]. Use [GPUHistory.Points]
// for one value per point.
type GPUHistory struct {
	Timestamps     []int64          `json:"timestamps"` // start of each step, Unix seconds
	SupplyDemand   GPUSupplyDemand  `json:"supply_demand"`
	AvailablePrice PercentileSeries `json:"available_price"`
	RentedPrice    PercentileSeries `json:"rented_price"`
	TFLOPS         *float64         `json:"tflops"`
}

// GPUSupplyDemand is the number of GPUs per step, averaged over its snapshots.
type GPUSupplyDemand struct {
	Total     []float64 `json:"total"`
	Available []float64 `json:"available"`
	Rented    []float64 `json:"rented"`
}

// PercentileSeries are price percentiles per step; nil where there was no
// price.
type PercentileSeries struct {
	P10 []*MicroUSD `json:"p10"`
	P25 []*MicroUSD `json:"p25"`
	P50 []*MicroUSD `json:"p50"`
	P75 []*MicroUSD `json:"p75"`
	P90 []*MicroUSD `json:"p90"`
}

// GPUHistoryPoint is the market of a GPU model in one step.
type GPUHistoryPoint struct {
	Time           time.Time
	Total          float64
	Available      float64
	Rented         float64
	AvailablePrice Percentiles
	RentedPrice    Percentiles
}

// Points returns one value per timestamp. A series shorter than Timestamps
// leaves the missing values zero or nil.
func (h GPUHistory) Points() []GPUHistoryPoint {
	points := make([]GPUHistoryPoint, len(h.Timestamps))
	for i, ts := range h.Timestamps {
		points[i] = GPUHistoryPoint{
			Time:           time.Unix(ts, 0),
			Total:          floatAt(h.SupplyDemand.Total, i),
			Available:      floatAt(h.SupplyDemand.Available, i),
			Rented:         floatAt(h.SupplyDemand.Rented, i),
			AvailablePrice: h.AvailablePrice.at(i),
			RentedPrice:    h.RentedPrice.at(i),
		}
	}
	return points
}

func (s PercentileSeries) at(i int) Percentiles {
	return Percentiles{
		P10: priceAt(s.P10, i), P25: priceAt(s.P25, i), P50: priceAt(s.P50, i),
		P75: priceAt(s.P75, i), P90: priceAt(s.P90, i),
	}
}

func floatAt(s []float64, i int) float64 {
	if i < len(s) {
		return s[i]
	}
	return 0
}

func priceAt(s []*MicroUSD, i int) *MicroUSD {
	if i < len(s) {
		return s[i]
	}
	return nil
}

// GPUHistory returns the GPU market over time, by model. Steps of an hour and
// more, and ranges older than 30 days, come from an hourly rollup.
//
// GET /stats/gpu/history
func (r *StatsResource) GPUHistory(ctx context.Context, params *GPUHistoryParams) (map[string]GPUHistory, error) {
	var history map[string]GPUHistory
	return history, r.c.getNullTolerant(ctx, "/stats/gpu/history", gpuHistoryQuery(params), &history)
}

func gpuHistoryQuery(p *GPUHistoryParams) url.Values {
	q := url.Values{}
	if p == nil {
		return q
	}
	if len(p.GPUNames) > 0 {
		q.Set("gpu_name", strings.Join(p.GPUNames, ","))
	}
	if p.Verified != "" {
		q.Set("verified", string(p.Verified))
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

// GPULocation is the number of GPUs of a model in a city, by GeoIP of the
// nodes; never a point per node.
type GPULocation struct {
	City      string   `json:"city"`
	Country   string   `json:"country"`
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	Model     string   `json:"model"`
	GPUs      int      `json:"gpus"`
	Rented    int      `json:"rented"`
	Verified  int      `json:"verified"` // of them on verified nodes
}

// GPULocations returns GPUs by city and model, as of the latest snapshot.
//
// GET /stats/gpu/locations
func (r *StatsResource) GPULocations(ctx context.Context) ([]GPULocation, error) {
	var locations []GPULocation
	return locations, r.c.getNullTolerant(ctx, "/stats/gpu/locations", nil, &locations)
}
