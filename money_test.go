package octaspace

import (
	"math"
	"testing"
)

func TestMicroUSD(t *testing.T) {
	tests := []struct {
		in      MicroUSD
		str     string
		dollars float64
	}{
		{0, "$0.000000", 0},
		{100, "$0.000100", 0.0001},
		{2175268, "$2.175268", 2.175268},
		{-1250000, "-$1.250000", -1.25},
		{math.MinInt64, "-$9223372036854.775808", float64(math.MinInt64) / 1e6},
	}
	for _, tc := range tests {
		if got := tc.in.String(); got != tc.str {
			t.Errorf("MicroUSD(%d).String() = %q, want %q", int64(tc.in), got, tc.str)
		}
		if got := tc.in.Dollars(); got != tc.dollars {
			t.Errorf("MicroUSD(%d).Dollars() = %v, want %v", int64(tc.in), got, tc.dollars)
		}
	}
}
