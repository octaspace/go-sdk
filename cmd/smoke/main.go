// smoke is a CLI tool for verifying the OctaSpace Go SDK against a live API.
//
// Without OCTA_API_KEY — only the public /network endpoint is tested.
// With OCTA_API_KEY    — all endpoints are tested.
//
// Usage:
//
//	go run ./cmd/smoke
//	go run ./cmd/smoke -json
//	go run ./cmd/smoke -base-url https://api.octa.space -timeout 10s
//	OCTA_API_KEY=<key> go run ./cmd/smoke
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	octaspace "github.com/octaspace/go-sdk"
)

// check describes a single smoke test case.
type check struct {
	id           string
	label        string
	method       string
	path         string
	requiresAuth bool
	fn           func(ctx context.Context, c *octaspace.Client) (any, error)
}

// result holds the outcome of running one check.
type result struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	Method       string `json:"method"`
	Path         string `json:"path"`
	RequiresAuth bool   `json:"requires_auth"`
	Status       string `json:"status"` // "passed" | "failed" | "skipped"
	DurationMs   int64  `json:"duration_ms"`
	Summary      string `json:"summary,omitempty"`
	Error        string `json:"error,omitempty"`
	StatusCode   int    `json:"status_code,omitempty"`
}

type skippedError string

func (e skippedError) Error() string { return string(e) }

// smokeReport is the top-level JSON output.
type smokeReport struct {
	Kind       string   `json:"kind"`
	Status     string   `json:"status"`
	StartedAt  string   `json:"started_at"`
	DurationMs int64    `json:"duration_ms"`
	BaseURL    string   `json:"base_url"`
	Auth       string   `json:"auth"`
	Summary    summary  `json:"summary"`
	Checks     []result `json:"checks"`
}

type summary struct {
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

// checks is the canonical list of smoke test cases, mirroring ruby-sdk SmokeRunner.
var checks = []check{
	{
		id: "network.info", label: "client.Network.Info",
		method: "GET", path: "/network", requiresAuth: false,
		fn: func(ctx context.Context, c *octaspace.Client) (any, error) {
			return c.Network.Info(ctx)
		},
	},
	{
		id: "accounts.profile", label: "client.Accounts.Profile",
		method: "GET", path: "/accounts", requiresAuth: true,
		fn: func(ctx context.Context, c *octaspace.Client) (any, error) {
			return c.Accounts.Profile(ctx)
		},
	},
	{
		id: "accounts.balance", label: "client.Accounts.Balance",
		method: "GET", path: "/accounts/balance", requiresAuth: true,
		fn: func(ctx context.Context, c *octaspace.Client) (any, error) {
			bal, err := c.Accounts.Balance(ctx)
			if bal != nil {
				return bal.String(), err
			}
			return nil, err
		},
	},
	{
		id: "apps.list", label: "client.Apps.List",
		method: "GET", path: "/apps", requiresAuth: false,
		fn: func(ctx context.Context, c *octaspace.Client) (any, error) {
			return c.Apps.List(ctx)
		},
	},
	{
		id: "nodes.list", label: "client.Nodes.List",
		method: "GET", path: "/nodes", requiresAuth: true,
		fn: func(ctx context.Context, c *octaspace.Client) (any, error) {
			return c.Nodes.List(ctx, nil)
		},
	},
	{
		id: "nodes.list.filtered", label: "client.Nodes.List(Country)",
		method: "GET", path: "/nodes?country=US", requiresAuth: true,
		fn: func(ctx context.Context, c *octaspace.Client) (any, error) {
			country := "US"
			return c.Nodes.List(ctx, &octaspace.ListNodesParams{Country: &country})
		},
	},
	{
		id: "services.mr.list", label: "client.Services.MR.List",
		method: "GET", path: "/services/mr", requiresAuth: false,
		fn: func(ctx context.Context, c *octaspace.Client) (any, error) {
			return c.Services.MR.List(ctx, nil)
		},
	},
	{
		id: "services.r.list", label: "client.Services.R.List",
		method: "GET", path: "/services/r", requiresAuth: true,
		fn: func(ctx context.Context, c *octaspace.Client) (any, error) {
			return c.Services.R.List(ctx, nil)
		},
	},
	{
		id: "services.vpn.list", label: "client.Services.VPN.List",
		method: "GET", path: "/services/vpn", requiresAuth: false,
		fn: func(ctx context.Context, c *octaspace.Client) (any, error) {
			return c.Services.VPN.List(ctx, nil)
		},
	},
	{
		id: "render.list", label: "client.Render.List",
		method: "GET", path: "/services/render", requiresAuth: false,
		fn: func(ctx context.Context, c *octaspace.Client) (any, error) {
			return c.Render.List(ctx)
		},
	},
	{
		id: "sessions.list", label: "client.Sessions.List",
		method: "GET", path: "/sessions", requiresAuth: true,
		fn: func(ctx context.Context, c *octaspace.Client) (any, error) {
			return c.Sessions.List(ctx)
		},
	},
	{
		id: "sessions.get", label: "client.Sessions.Get",
		method: "GET", path: "/services/:uuid/info", requiresAuth: true,
		fn: func(ctx context.Context, c *octaspace.Client) (any, error) {
			uuid, err := firstActiveSessionUUID(ctx, c, false)
			if err != nil {
				return nil, err
			}
			if uuid == "" {
				return nil, skippedError("no active sessions")
			}
			return c.Sessions.Get(ctx, uuid)
		},
	},
	{
		id: "sessions.logs", label: "client.Sessions.Logs",
		method: "GET", path: "/services/:uuid/logs", requiresAuth: true,
		fn: func(ctx context.Context, c *octaspace.Client) (any, error) {
			uuid, err := firstActiveSessionUUID(ctx, c, true)
			if err != nil {
				return nil, err
			}
			if uuid == "" {
				return nil, skippedError("no active compute sessions")
			}
			return c.Sessions.Logs(ctx, uuid, &octaspace.LogsParams{Tail: 20})
		},
	},
}

// firstActiveSessionUUID returns the UUID of the first running session (the
// first compute one when computeOnly is set, as only those have logs), or ""
// if there is none.
func firstActiveSessionUUID(ctx context.Context, c *octaspace.Client, computeOnly bool) (string, error) {
	sessions, err := c.Sessions.List(ctx)
	if err != nil {
		return "", err
	}
	for _, s := range sessions {
		if !computeOnly || s.IsCompute() {
			return s.UUID, nil
		}
	}
	return "", nil
}

func main() {
	var (
		baseURL   = flag.String("base-url", "https://api.octa.space", "API base URL")
		timeout   = flag.Duration("timeout", 30*time.Second, "per-request timeout")
		jsonOut   = flag.Bool("json", false, "output results as JSON")
		onlyCheck = flag.String("only", "", "run only the check with this id")
	)
	flag.Parse()

	apiKey := os.Getenv("OCTA_API_KEY")
	hasAuth := apiKey != ""

	client := octaspace.NewClient(apiKey,
		octaspace.WithBaseURL(*baseURL),
		octaspace.WithTimeout(*timeout),
		octaspace.WithMaxRetries(0), // no retries in smoke tests
	)

	ctx := context.Background()
	startedAt := time.Now()

	if !*jsonOut {
		fmt.Printf("OctaSpace Go SDK — smoke test\n")
		fmt.Printf("Base URL : %s\n", *baseURL)
		if hasAuth {
			fmt.Printf("Auth     : present (%s)\n", maskKey(apiKey))
		} else {
			fmt.Printf("Auth     : none (only public endpoints will be tested)\n")
		}
		fmt.Println()
	}

	results := make([]result, 0, len(checks))
	for _, chk := range checks {
		if *onlyCheck != "" && chk.id != *onlyCheck {
			continue
		}
		r := runCheck(ctx, client, chk, hasAuth)
		results = append(results, r)
		if !*jsonOut {
			printResult(r)
		}
	}

	totalMs := time.Since(startedAt).Milliseconds()

	summary := countResults(results)

	if *jsonOut {
		authDesc := "none"
		if hasAuth {
			authDesc = maskKey(apiKey)
		}
		status := "passed"
		if summary.Failed > 0 {
			status = "failed"
		}
		report := smokeReport{
			Kind:       "sdk_smoke",
			Status:     status,
			StartedAt:  startedAt.UTC().Format(time.RFC3339),
			DurationMs: totalMs,
			BaseURL:    *baseURL,
			Auth:       authDesc,
			Summary:    summary,
			Checks:     results,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(report)
	} else {
		fmt.Println()
		fmt.Printf("Summary: %d/%d passed", summary.Passed, len(results)-summary.Skipped)
		if summary.Failed > 0 {
			fmt.Printf(", %d FAILED", summary.Failed)
		}
		if summary.Skipped > 0 {
			fmt.Printf(", %d skipped", summary.Skipped)
		}
		fmt.Printf("   total: %dms\n", totalMs)
	}

	if summary.Failed > 0 {
		os.Exit(1)
	}
}

func countResults(results []result) summary {
	var s summary
	s.Total = len(results)
	for _, r := range results {
		switch r.Status {
		case "passed":
			s.Passed++
		case "failed":
			s.Failed++
		case "skipped":
			s.Skipped++
		}
	}
	return s
}

// runCheck executes a single smoke check and returns its result.
func runCheck(ctx context.Context, c *octaspace.Client, chk check, hasAuth bool) result {
	r := result{
		ID:           chk.id,
		Label:        chk.label,
		Method:       chk.method,
		Path:         chk.path,
		RequiresAuth: chk.requiresAuth,
	}

	if chk.requiresAuth && !hasAuth {
		r.Status = "skipped"
		r.Summary = "no auth"
		return r
	}

	start := time.Now()
	val, err := chk.fn(ctx, c)
	r.DurationMs = time.Since(start).Milliseconds()

	if err != nil {
		var skipped skippedError
		if errors.As(err, &skipped) {
			r.Status = "skipped"
			r.Summary = string(skipped)
			return r
		}
		r.Status = "failed"
		r.Error = err.Error()
		var apiErr *octaspace.APIError
		if errors.As(err, &apiErr) {
			r.StatusCode = apiErr.StatusCode
		}
		return r
	}

	r.Status = "passed"
	r.Summary = summarize(val)
	return r
}

// printResult prints a single check result in human-readable format.
func printResult(r result) {
	switch r.Status {
	case "passed":
		fmt.Printf("  [PASS]  %-42s  %s  %4dms", r.Label, r.Method+" "+r.Path, r.DurationMs)
		if r.Summary != "" {
			fmt.Printf("   %s", r.Summary)
		}
		fmt.Println()
	case "failed":
		fmt.Printf("  [FAIL]  %-42s  %s  %4dms", r.Label, r.Method+" "+r.Path, r.DurationMs)
		if r.StatusCode != 0 {
			fmt.Printf("   HTTP %d", r.StatusCode)
		}
		fmt.Printf("\n          error: %s\n", r.Error)
	case "skipped":
		fmt.Printf("  [SKIP]  %-42s  %s  %4s", r.Label, r.Method+" "+r.Path, "—")
		if r.Summary != "" {
			fmt.Printf("   %s", r.Summary)
		}
		fmt.Println()
	}
}

// summarize returns a short human-readable description of a response value.
// It round-trips through JSON so that typed structs are treated uniformly.
func summarize(v any) string {
	if v == nil {
		return ""
	}
	// Scalars can contain account data (for example a balance), so report only
	// their type rather than their value.
	if _, ok := v.(string); ok {
		return "string"
	}

	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}

	var raw any
	if err := json.Unmarshal(b, &raw); err != nil {
		return ""
	}

	switch x := raw.(type) {
	case []any:
		if len(x) == 0 {
			return "empty array"
		}
		s := fmt.Sprintf("%d items", len(x))
		if obj, ok := x[0].(map[string]any); ok {
			keys := topKeys(obj, 5)
			s += fmt.Sprintf("  keys: [%s]", strings.Join(keys, ", "))
		}
		return s
	case map[string]any:
		keys := topKeys(x, 8)
		return fmt.Sprintf("{%s}", strings.Join(keys, ", "))
	default:
		return fmt.Sprintf("%T", v)
	}
}

// topKeys returns up to n keys from a map, sorted for deterministic output.
func topKeys(m map[string]any, n int) []string {
	all := make([]string, 0, len(m))
	for k := range m {
		all = append(all, k)
	}
	sort.Strings(all)
	if len(all) > n {
		return all[:n]
	}
	return all
}

// maskKey returns the API key with the middle portion replaced by asterisks.
// Shows first 8 and last 4 characters.
func maskKey(key string) string {
	const show = 8
	if len(key) <= show+4 {
		return strings.Repeat("*", len(key))
	}
	return key[:show] + strings.Repeat("*", len(key)-show-4) + key[len(key)-4:]
}
