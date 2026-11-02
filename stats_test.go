package octaspace

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestGPUMarket_Fixture(t *testing.T) {
	var m GPUMarket
	decodeStrictNullTolerant(t, "testdata/next/stats_gpu_current.json", &m)
	if m.At == 0 || len(m.GPUs) != 2 {
		t.Fatalf("market = %+v", m)
	}
	var rtx, arc GPUModelStats
	for _, g := range m.GPUs {
		switch g.Model {
		case "NVIDIA GeForce RTX 3090":
			rtx = g
		case "Arc B580":
			arc = g
		}
	}
	if rtx.TFLOPS == nil || rtx.AvailablePrice.P50 == nil || *rtx.AvailablePrice.P50 != 33 || rtx.Total != 4 {
		t.Errorf("rtx = %+v", rtx)
	}
	// The Arc B580 has no TFLOPS and no rented price: sent as "null" strings.
	if arc.TFLOPS != nil || arc.TFLOPSPerDollar != nil || arc.RentedPrice.P10 != nil {
		t.Errorf("arc nullable fields should be nil: %+v", arc)
	}
}

func TestStats_GPUCurrent(t *testing.T) {
	fixture := readFixture(t, "testdata/next/stats_gpu_current.json")
	var query string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		if r.URL.Path != "/stats/gpu/current" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write(fixture)
	})
	if _, err := c.Stats.GPUCurrent(context.Background(), &GPUStatsParams{Verified: VerifiedYes}); err != nil || query != "verified=yes" {
		t.Errorf("GPUCurrent: query = %q, err = %v", query, err)
	}
	if _, err := c.Stats.GPUCurrent(context.Background(), nil); err != nil || query != "" {
		t.Errorf("GPUCurrent(nil): query = %q, err = %v", query, err)
	}
}

func TestGPUHistory_FixtureAndPoints(t *testing.T) {
	var h map[string]GPUHistory
	decodeStrictNullTolerant(t, "testdata/next/stats_gpu_history.json", &h)
	arc, ok := h["Arc B580"]
	if !ok || len(arc.Timestamps) == 0 {
		t.Fatalf("history = %+v", h)
	}
	points := arc.Points()
	if len(points) != len(arc.Timestamps) {
		t.Fatalf("points = %d, timestamps = %d", len(points), len(arc.Timestamps))
	}
	if points[0].Time.Unix() != arc.Timestamps[0] || points[0].Total != arc.SupplyDemand.Total[0] {
		t.Errorf("points[0] = %+v", points[0])
	}
	if points[0].RentedPrice.P10 != nil || points[1].RentedPrice.P10 == nil {
		t.Errorf("rented p10 = %v, %v; want nil then a price", points[0].RentedPrice.P10, points[1].RentedPrice.P10)
	}
	short := GPUHistory{Timestamps: []int64{1, 2}, SupplyDemand: GPUSupplyDemand{Total: []float64{5}}}
	if p := short.Points(); p[1].Total != 0 || p[1].AvailablePrice.P10 != nil {
		t.Errorf("short series should leave zero values: %+v", p[1])
	}
}

func TestStats_GPUHistoryQuery(t *testing.T) {
	fixture := readFixture(t, "testdata/next/stats_gpu_history.json")
	var query string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		_, _ = w.Write(fixture)
	})
	start := time.Unix(1793480000, 0)
	_, err := c.Stats.GPUHistory(context.Background(), &GPUHistoryParams{
		GPUNames: []string{"NVIDIA GeForce RTX 4090", "NVIDIA GeForce RTX 3090"},
		Verified: VerifiedNo,
		Start:    start,
		End:      start.Add(24 * time.Hour),
		Step:     2 * time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "end=1793566400&gpu_name=NVIDIA+GeForce+RTX+4090%2CNVIDIA+GeForce+RTX+3090&start=1793480000&step=7200&verified=no"
	if query != want {
		t.Errorf("query =\n%s\nwant\n%s", query, want)
	}
	if _, err := c.Stats.GPUHistory(context.Background(), nil); err != nil || query != "" {
		t.Errorf("nil params: query = %q, err = %v", query, err)
	}
}

func TestStats_GPULocations(t *testing.T) {
	var locs []GPULocation
	fixture := decodeStrictNullTolerant(t, "testdata/next/stats_gpu_locations.json", &locs)
	if len(locs) != 2 || locs[0].Latitude == nil || locs[0].GPUs == 0 {
		t.Errorf("locations = %+v", locs)
	}
	c, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(fixture) })
	got, err := c.Stats.GPULocations(context.Background())
	if err != nil || len(got) != 2 {
		t.Errorf("GPULocations = %d, %v", len(got), err)
	}
}

func TestStats_ValidationError(t *testing.T) {
	c, _ := newTestClient(t, statusHandler(400, `{"message":"step must be at least 300 seconds"}`))
	if _, err := c.Stats.GPUHistory(context.Background(), &GPUHistoryParams{Step: time.Minute}); !IsValidation(err) {
		t.Errorf("err = %v, want ValidationError", err)
	}
}

func readFixture(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
