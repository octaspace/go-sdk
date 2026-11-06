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
	"time"

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
			octaspace.DefaultBaseURL,
			"https://api-backup.example.com", // failover if primary is unreachable
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
			octaspace.DefaultBaseURL,
			"https://api-backup.example.com",
		}),
	)
	_ = client
}

// ExampleAPIError_errorsAs shows how to inspect typed API errors returned by
// the SDK. All errors produced by SDK methods can be checked with errors.As.
func ExampleAPIError_errorsAs() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"Invalid bearer token"}`))
	}))
	defer srv.Close()

	client := octaspace.NewClient("bad-key",
		octaspace.WithBaseURL(srv.URL),
		octaspace.WithMaxRetries(0),
	)

	_, err := client.Accounts.Balance(context.Background())
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
	// HTTP 401: Invalid bearer token
}

// ExampleClient_Network shows how to fetch live network statistics.
func ExampleClient_Network() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"at":    1793566134,
			"nodes": map[string]any{"online": 690, "available": 420, "rented": 250, "countries": 41},
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
	fmt.Printf("online %d, rented %d, in %d countries\n",
		info.Nodes.Online, info.Nodes.Rented, info.Nodes.Countries)
	// Output:
	// online 690, rented 250, in 41 countries
}

// ExampleSessionsResource_WaitReady starts a compute session, waits until it
// is configured, reads its logs and stops it.
func ExampleSessionsResource_WaitReady() {
	client := octaspace.NewClient(os.Getenv("OCTA_API_KEY"))
	ctx := context.Background()

	uuid, err := client.Compute.Start(ctx, &octaspace.ComputeStartParams{
		NodeNumber: "7KQ4-M2XD-4",
		Image:      "ubuntu:24.04",
		Entrypoint: "sh -c",
		Args:       `"echo hello; sleep infinity"`,
	})
	if err != nil {
		return
	}
	defer client.Sessions.Stop(context.Background(), uuid)

	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	session, err := client.Sessions.WaitReady(waitCtx, uuid, &octaspace.WaitOptions{
		OnProgress: func(s *octaspace.Session) { log.Printf("%s %s", s.Status, s.Progress) },
	})
	if errors.Is(err, octaspace.ErrSessionStopped) {
		return // configuration failed and the session was stopped
	}
	if err != nil {
		return
	}
	_ = session.NodeHW

	logs, err := client.Sessions.Logs(ctx, uuid, &octaspace.LogsParams{Tail: 50})
	if err == nil && logs.Container != nil {
		log.Print(*logs.Container)
	}
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

// ExampleComputeResource_Start picks the cheapest machine for rent and starts
// a session on it.
func ExampleComputeResource_Start() {
	client := octaspace.NewClient(os.Getenv("OCTA_API_KEY"))
	ctx := context.Background()

	machines, err := client.Compute.List(ctx)
	if err != nil || len(machines) == 0 {
		return
	}
	cheapest := machines[0]
	for _, m := range machines[1:] {
		if m.BasePrice < cheapest.BasePrice {
			cheapest = m
		}
	}
	fmt.Printf("%s: %s/h, minimum rent %d min\n", cheapest.NodeNumber, cheapest.BasePrice, cheapest.MinRentMinutes)

	uuid, err := client.Compute.Start(ctx, &octaspace.ComputeStartParams{
		NodeNumber: cheapest.NodeNumber,
		Image:      "ubuntu:24.04",
		DiskSize:   20,
		HTTPPorts:  []int{8888},
		Envs:       map[string]string{"TOKEN": "secret"},
		SSH:        octaspace.Bool(true),
	})
	if octaspace.IsValidation(err) {
		log.Printf("rejected: %v", err) // e.g. insufficient balance, node not available
		return
	}
	if err != nil {
		return
	}
	fmt.Println("started", uuid)
}

// ExampleHostingResource_SetPrices changes the base price and the minimum
// rent of a hosted node; the other prices stay as they are.
func ExampleHostingResource_SetPrices() {
	client := octaspace.NewClient(os.Getenv("OCTA_API_KEY"))
	err := client.Hosting.SetPrices(context.Background(), "7KQ4-M2XD-4", octaspace.NodePrices{
		Base:           octaspace.Price(250_000), // $0.25 per hour
		MinRentMinutes: octaspace.Int(15),
	})
	if octaspace.IsValidation(err) {
		log.Printf("price out of range: %v", err)
	}
}
