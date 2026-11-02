package octaspace

import "fmt"

// MicroUSD is an amount of US dollars in millionths: 1 MicroUSD = $0.000001.
// The API reports every price, charge and balance in this unit as an integer.
type MicroUSD int64

// microUSDPerDollar is the number of MicroUSD in one dollar.
const microUSDPerDollar = 1_000_000

// Dollars returns the amount in dollars. Use it for display or arithmetic
// that tolerates float64 rounding; keep MicroUSD for exact sums.
func (m MicroUSD) Dollars() float64 { return float64(m) / microUSDPerDollar }

// String formats the amount as dollars with all six decimal places, e.g.
// "$0.000100" or "-$1.250000".
func (m MicroUSD) String() string {
	whole, frac := int64(m)/microUSDPerDollar, int64(m)%microUSDPerDollar
	if m < 0 {
		// Negate the parts, not m itself: -m overflows for math.MinInt64.
		return fmt.Sprintf("-$%d.%06d", -whole, -frac)
	}
	return fmt.Sprintf("$%d.%06d", whole, frac)
}
