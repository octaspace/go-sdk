package octaspace_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"

	octaspace "github.com/octaspace/go-sdk"
)

// ExampleNewClient shows the minimal client setup using an API key from the
// environment.
func ExampleNewClient() {
	client := octaspace.NewClient(os.Getenv("OCTA_API_KEY"))
	_ = client
}

// ExampleNewClient_options shows all available configuration options.
func ExampleNewClient_options() {
	l := log.New(os.Stderr, "[octa] ", log.LstdFlags)

	client := octaspace.NewClient(
		os.Getenv("OCTA_API_KEY"),
		octaspace.WithBaseURLs([]string{
			"https://api.octa.space",
			"https://api-backup.octa.space", // failover if primary is unreachable
		}),
		octaspace.WithTimeout(15_000_000_000), // 15s per-request timeout
		octaspace.WithMaxRetries(3),
		octaspace.WithLogger(octaspace.NewStdLogger(l)),
		octaspace.WithUserAgent("my-app/1.0"),
	)
	_ = client
}

// ExampleWithBaseURLs demonstrates multi-URL failover: the client automatically
// tries the next URL when a network-level error occurs on the current one.
func ExampleWithBaseURLs() {
	client := octaspace.NewClient(
		os.Getenv("OCTA_API_KEY"),
		octaspace.WithBaseURLs([]string{
			"https://api.octa.space",
			"https://api-backup.octa.space",
		}),
	)
	_ = client
}

// ExampleAPIError_errorsAs shows how to inspect typed API errors returned by
// the SDK. All errors produced by SDK methods can be checked with errors.As.
func ExampleAPIError_errorsAs() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid api key"}`))
	}))
	defer srv.Close()

	client := octaspace.NewClient("bad-key",
		octaspace.WithBaseURL(srv.URL),
		octaspace.WithMaxRetries(0),
	)

	_, err := client.Accounts.Profile(context.Background())
	if err != nil {
		var apiErr *octaspace.APIError
		switch {
		case errors.As(err, &apiErr):
			fmt.Printf("HTTP %d: %s\n", apiErr.StatusCode, apiErr.Message)
		case octaspace.IsAuthError(err):
			fmt.Println("authentication failed")
		case octaspace.IsNotFound(err):
			fmt.Println("resource not found")
		case octaspace.IsRateLimit(err):
			fmt.Println("rate limited — check Retry-After")
		default:
			fmt.Printf("unexpected error: %v\n", err)
		}
	}
	// Output:
	// HTTP 401: invalid api key
}

// ExampleClient_Network shows how to fetch live network statistics.
func ExampleClient_Network() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"nodes": map[string]any{"count": 690, "vpn": 147},
		})
	}))
	defer srv.Close()

	client := octaspace.NewClient("",
		octaspace.WithBaseURL(srv.URL),
		octaspace.WithMaxRetries(0),
	)

	info, err := client.Network.Info(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	// info is map[string]any — fields vary by response.
	fmt.Printf("network keys: %d\n", len(info))
	// Output:
	// network keys: 1
}

// ExampleVPNCreateParams shows how to start a WireGuard VPN session and a
// v2ray session, which requires the Protocol field.
func ExampleVPNCreateParams() {
	client := octaspace.NewClient(os.Getenv("OCTA_API_KEY"))
	ctx := context.Background()

	// WireGuard — only NodeID and SubKind are needed.
	_, _ = client.Services.VPN.Create(ctx, &octaspace.VPNCreateParams{
		NodeID:  12345,
		SubKind: "wg",
	})

	// v2ray — Protocol is required.
	_, _ = client.Services.VPN.Create(ctx, &octaspace.VPNCreateParams{
		NodeID:   12345,
		SubKind:  "v2ray",
		Protocol: "vmess",
	})

	// Forward arbitrary fields the API may accept via Extra.
	_, _ = client.Services.VPN.Create(ctx, &octaspace.VPNCreateParams{
		NodeID:  12345,
		SubKind: "wg",
		Extra:   map[string]any{"custom_field": "value"},
	})
}

// ExampleLogsParams shows how to fetch logs for a finished (recent) session.
// Active sessions do not need LogsParams; pass nil instead.
func ExampleLogsParams() {
	client := octaspace.NewClient(os.Getenv("OCTA_API_KEY"))
	ctx := context.Background()
	uuid := "550e8400-e29b-41d4-a716-446655440000"

	proxy := client.Services.Session(uuid)

	// Active session — no params needed.
	_, _ = proxy.Logs(ctx, nil)

	// Finished (recent) session — recent=true is required by the API.
	recent := true
	_, _ = proxy.Logs(ctx, &octaspace.LogsParams{Recent: &recent})
}

// ExampleMachineRentalCreateParams shows how to start a machine rental session
// with a GPU workload using a custom Docker image.
func ExampleMachineRentalCreateParams() {
	client := octaspace.NewClient(os.Getenv("OCTA_API_KEY"))

	resp, err := client.Services.MR.Create(context.Background(), &octaspace.MachineRentalCreateParams{
		NodeID:   98765,
		DiskSize: 50,
		Image:    "nvidia/cuda:12.0-base",
		Ports:    []int{22},
		HTTPPorts: []int{8888}, // Jupyter notebook
		Envs: map[string]string{
			"JUPYTER_TOKEN": "secret",
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("session UUID: %s\n", resp.UUID)
}

// ExampleNewStdLogger shows how to attach a standard library logger for
// request/response diagnostics. Note: *log.Logger does not directly satisfy
// the Logger interface; use NewStdLogger to wrap it.
func ExampleNewStdLogger() {
	l := log.New(os.Stderr, "[octaspace] ", log.LstdFlags)
	client := octaspace.NewClient(
		os.Getenv("OCTA_API_KEY"),
		octaspace.WithLogger(octaspace.NewStdLogger(l)),
	)
	_ = client
}

// ExampleListNodesParams shows how to filter compute nodes by country and
// compatible application.
func ExampleListNodesParams() {
	client := octaspace.NewClient(os.Getenv("OCTA_API_KEY"))
	ctx := context.Background()

	country := "DE"
	nodes, err := client.Nodes.List(ctx, &octaspace.ListNodesParams{
		Country: &country,
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, n := range nodes {
		fmt.Printf("node %d  %s  %s\n", n.ID, n.Location.City, n.State)
	}
}
