package octaspace

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	// hostingIntegrationFlag enables the live hosting test. It changes the
	// prices of a real node of the account, so it is opt-in and needs the
	// node named in OCTA_HOSTING_NODE.
	hostingIntegrationFlag = "OCTA_RUN_HOSTING_INTEGRATION"
	// priceVisibleTimeout bounds the wait for a new price to show in
	// GET /hosting/nodes, which normally takes a few seconds.
	priceVisibleTimeout = 30 * time.Second
)

// TestHostingSetPricesIntegration changes the traffic price of
// OCTA_HOSTING_NODE by one micro-USD, waits until GET /hosting/nodes shows the
// new value, and restores the original price. A 204 from the PATCH alone does
// not prove the change took effect.
func TestHostingSetPricesIntegration(t *testing.T) {
	if os.Getenv(hostingIntegrationFlag) != "1" {
		t.Skipf("set %s=1 to run the live hosting test", hostingIntegrationFlag)
	}
	key := os.Getenv("OCTA_API_KEY")
	if key == "" {
		t.Skip("OCTA_API_KEY is not set")
	}
	number := os.Getenv("OCTA_HOSTING_NODE")
	if number == "" {
		t.Skip("OCTA_HOSTING_NODE is not set")
	}
	var opts []Option
	if u := os.Getenv("OCTA_API_URL"); u != "" {
		opts = append(opts, WithBaseURL(u))
	}
	c := NewClient(key, opts...)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	orig := hostingNode(ctx, t, c, number).Prices
	want := orig.Traffic + 1
	if want > 1_000_000_000 {
		want = orig.Traffic - 1
	}

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if err := c.Hosting.SetPrices(ctx, number, NodePrices{Traffic: Price(orig.Traffic)}); err != nil {
			t.Errorf("CLEANUP: restore traffic price %d on %s: %v — restore it manually", orig.Traffic, number, err)
			return
		}
		if !waitTrafficPrice(ctx, t, c, number, orig.Traffic) {
			t.Errorf("CLEANUP: traffic price on %s did not return to %d within %s — check it manually", number, orig.Traffic, priceVisibleTimeout)
		}
	})

	if err := c.Hosting.SetPrices(ctx, number, NodePrices{Traffic: Price(want)}); err != nil {
		t.Fatalf("SetPrices: %v", err)
	}
	if !waitTrafficPrice(ctx, t, c, number, want) {
		t.Fatalf("SetPrices succeeded, but GET /hosting/nodes did not show traffic = %d within %s", want, priceVisibleTimeout)
	}
}

// waitTrafficPrice polls GET /hosting/nodes until the node's traffic price is
// want, for at most priceVisibleTimeout.
func waitTrafficPrice(ctx context.Context, t *testing.T, c *Client, number string, want MicroUSD) bool {
	t.Helper()
	deadline := time.Now().Add(priceVisibleTimeout)
	for {
		if hostingNode(ctx, t, c, number).Prices.Traffic == want {
			return true
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			return false
		}
		time.Sleep(2 * time.Second)
	}
}

// hostingNode returns the account's node with the given number (case, dashes
// and spaces ignored), or fails the test.
func hostingNode(ctx context.Context, t *testing.T, c *Client, number string) HostingNode {
	t.Helper()
	nodes, err := c.Hosting.Nodes(ctx)
	if err != nil {
		t.Fatalf("Hosting.Nodes: %v", err)
	}
	key := normalizeNodeNumber(number)
	for _, n := range nodes {
		if normalizeNodeNumber(n.Number) == key {
			return n
		}
	}
	t.Fatalf("node %s is not among the account's online nodes", number)
	return HostingNode{}
}

func normalizeNodeNumber(s string) string {
	return strings.ToUpper(strings.NewReplacer("-", "", " ", "").Replace(s))
}
