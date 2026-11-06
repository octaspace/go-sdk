package octaspace

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	// computeIntegrationFlag enables the live compute tests. The session test
	// starts a billed session, so it is opt-in.
	computeIntegrationFlag = "OCTA_RUN_COMPUTE_INTEGRATION"
	// computeIntegrationMaxHourly caps base + storage per hour of the machine
	// chosen when OCTA_NODE is not set: $0.05.
	computeIntegrationMaxHourly MicroUSD = 50_000
	computeIntegrationMarker             = "octaspace-go-integration"
)

// integrationClient returns a client for the live API, or skips the test
// unless OCTA_RUN_COMPUTE_INTEGRATION=1 and OCTA_API_KEY are set. OCTA_API_URL
// overrides the base URL (default: DefaultBaseURL).
func integrationClient(t *testing.T) *Client {
	t.Helper()
	if os.Getenv(computeIntegrationFlag) != "1" {
		t.Skipf("set %s=1 to run live compute tests", computeIntegrationFlag)
	}
	key := os.Getenv("OCTA_API_KEY")
	if key == "" {
		t.Skip("OCTA_API_KEY is not set")
	}
	var opts []Option
	if u := os.Getenv("OCTA_API_URL"); u != "" {
		opts = append(opts, WithBaseURL(u))
	}
	return NewClient(key, opts...)
}

// TestComputeRejectionIntegration checks that invalid start requests are
// rejected without starting anything.
func TestComputeRejectionIntegration(t *testing.T) {
	c := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	_, err := c.Compute.Start(ctx, &ComputeStartParams{NodeNumber: "ZZZZ-ZZZZ-Z", Image: "alpine:3.20"})
	if !IsValidation(err) {
		t.Errorf("Start with an invalid node number: err = %v, want ValidationError", err)
	}
	results, err := c.Compute.StartBatch(ctx, []ComputeBatchItem{
		{ID: "bad-node", ComputeStartParams: ComputeStartParams{NodeNumber: "ZZZZ-ZZZZ-Z", Image: "alpine:3.20"}},
		{ID: "no-image", ComputeStartParams: ComputeStartParams{NodeNumber: "0000-0000-0"}},
	})
	if err != nil {
		t.Fatalf("StartBatch: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %+v", results)
	}
	for _, r := range results {
		if r.OK() || r.Reason == "" {
			t.Errorf("result %+v: want a failure with a reason", r)
		}
	}
}

// TestComputeSessionIntegration runs a compute session through its life:
// start, wait until ready, logs, restart, stop. It starts a billed session on
// OCTA_NODE, or on the cheapest machine without a minimum rent under
// computeIntegrationMaxHourly, and always stops it.
func TestComputeSessionIntegration(t *testing.T) {
	c := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	node := os.Getenv("OCTA_NODE")
	if node == "" {
		node = cheapestComputeNode(ctx, t, c)
	}

	uuid, err := c.Compute.Start(ctx, &ComputeStartParams{
		NodeNumber: node,
		Image:      "alpine:3.20",
		DiskSize:   1,
		Entrypoint: "sh -c",
		Args:       `"echo ` + computeIntegrationMarker + `; sleep 3600"`,
		SSH:        Bool(false),
		Envs:       map[string]string{"OCTASPACE_GO_IT": "1"},
	})
	if err != nil {
		t.Fatalf("Start on %s: %v", node, err)
	}
	t.Cleanup(func() { stopSession(t, c, uuid) })

	waitCtx, waitCancel := context.WithTimeout(ctx, 5*time.Minute)
	defer waitCancel()
	s, err := c.Sessions.WaitReady(waitCtx, uuid, nil)
	if err != nil {
		t.Fatalf("WaitReady: %v", err)
	}
	if !s.IsCompute() || s.NodeNumber == "" || s.NodeHW == nil || s.Envs["OCTASPACE_GO_IT"] != "1" {
		t.Errorf("session = %+v", s)
	}

	sessions, err := c.Sessions.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if !containsSession(sessions, uuid) {
		t.Errorf("List does not contain %s", uuid)
	}

	logs := waitForLog(ctx, t, c, uuid, computeIntegrationMarker)
	if len(logs.System) == 0 {
		t.Errorf("system log is empty")
	}

	if _, err := c.Compute.Restart(ctx, uuid, nil); err != nil {
		t.Fatalf("Restart: %v", err)
	}
	waitCtx2, waitCancel2 := context.WithTimeout(ctx, 5*time.Minute)
	defer waitCancel2()
	if _, err := c.Sessions.WaitReady(waitCtx2, uuid, nil); err != nil {
		t.Fatalf("WaitReady after restart: %v", err)
	}
	_, err = c.Compute.Restart(ctx, uuid, nil)
	var rl *RateLimitError
	if !errors.As(err, &rl) || rl.RetryAfter <= 0 {
		t.Errorf("Restart within the cooldown: err = %v, want RateLimitError with RetryAfter", err)
	}
}

// cheapestComputeNode picks the cheapest listed machine without a minimum rent
// whose base + storage price per hour is at most computeIntegrationMaxHourly.
func cheapestComputeNode(ctx context.Context, t *testing.T, c *Client) string {
	t.Helper()
	machines, err := c.Compute.List(ctx)
	if err != nil {
		t.Fatalf("Compute.List: %v", err)
	}
	best, bestPrice := "", MicroUSD(0)
	for _, m := range machines {
		price := m.BasePrice + m.StoragePrice
		if m.MinRentMinutes != 0 || price > computeIntegrationMaxHourly {
			continue
		}
		if best == "" || price < bestPrice {
			best, bestPrice = m.NodeNumber, price
		}
	}
	if best == "" {
		t.Skipf("no machine without a minimum rent under %s/h; set OCTA_NODE", computeIntegrationMaxHourly)
	}
	return best
}

// waitForLog polls the logs until the container output contains marker.
func waitForLog(ctx context.Context, t *testing.T, c *Client, uuid, marker string) *SessionLogs {
	t.Helper()
	deadline := time.Now().Add(time.Minute)
	for {
		logs, err := c.Sessions.Logs(ctx, uuid, &LogsParams{Tail: 20})
		if err != nil {
			t.Fatalf("Logs: %v", err)
		}
		if logs.Container != nil && strings.Contains(*logs.Container, marker) {
			return logs
		}
		if time.Now().After(deadline) {
			t.Fatalf("container log never contained %q", marker)
		}
		time.Sleep(3 * time.Second)
	}
}

func containsSession(sessions []Session, uuid string) bool {
	for _, s := range sessions {
		if s.UUID == uuid {
			return true
		}
	}
	return false
}

// stopSession stops the session with its own context, so a failed or timed
// out test still cleans up, and waits until the API no longer reports it.
func stopSession(t *testing.T, c *Client, uuid string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if _, err := c.Sessions.Stop(ctx, uuid); err != nil && !IsNotFound(err) {
		t.Errorf("CLEANUP: stop %s: %v — stop it manually", uuid, err)
		return
	}
	for {
		_, err := c.Sessions.Get(ctx, uuid)
		if IsNotFound(err) {
			return
		}
		if ctx.Err() != nil {
			t.Errorf("CLEANUP: %s still reported after stop — check it manually", uuid)
			return
		}
		time.Sleep(3 * time.Second)
	}
}
