// Basic usage example for the OctaSpace Go SDK.
//
// Without OCTA_API_KEY — only public endpoints are called.
// With OCTA_API_KEY    — the account's balance, sessions and nodes are shown too.
// OCTA_API_URL overrides the base URL (default: octaspace.DefaultBaseURL).
//
// Nothing is started or changed.
//
// Run:
//
//	go run ./examples/basic
//	OCTA_API_KEY=<key> go run ./examples/basic
package main

import (
	"context"
	"fmt"
	"os"

	octaspace "github.com/octaspace/go-sdk"
)

func main() {
	ctx := context.Background()
	apiKey := os.Getenv("OCTA_API_KEY")

	var opts []octaspace.Option
	if u := os.Getenv("OCTA_API_URL"); u != "" {
		opts = append(opts, octaspace.WithBaseURL(u))
	}
	client := octaspace.NewClient(apiKey, opts...)

	// ── Public ──────────────────────────────────────────────────────────────
	fmt.Println("=== Network (no auth required) ===")
	if info, err := client.Network.Info(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "network.info error: %v\n", err)
	} else {
		fmt.Printf("  %d nodes online in %d countries, %d available, %d rented\n",
			info.Nodes.Online, info.Nodes.Countries, info.Nodes.Available, info.Nodes.Rented)
		fmt.Printf("  %d sessions running, %.1f TFLOPS\n", info.Sessions.Running, info.Power.TFLOPS)
	}

	fmt.Println("\n=== Machines for rent ===")
	if machines, err := client.Compute.List(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "compute.list error: %v\n", err)
	} else {
		for i, m := range machines {
			if i >= 5 {
				fmt.Printf("  ... %d more\n", len(machines)-i)
				break
			}
			fmt.Printf("  %s  %s, %s  %d GPU  base %s/h  storage %s/GB·h  traffic %s/GB\n",
				m.NodeNumber, m.City, m.Country, len(m.GPUs), m.BasePrice, m.StoragePrice, m.TrafficPrice)
		}
	}

	fmt.Println("\n=== GPU market ===")
	if market, err := client.Stats.GPUCurrent(ctx, nil); err != nil {
		fmt.Fprintf(os.Stderr, "stats.gpu.current error: %v\n", err)
	} else {
		for _, g := range market.GPUs {
			median := "n/a"
			if g.AvailablePrice.P50 != nil {
				median = g.AvailablePrice.P50.String()
			}
			fmt.Printf("  %-26s %d total, %d rented, median listing %s/GPU·h\n", g.Model, g.Total, g.Rented, median)
		}
	}

	if apiKey == "" {
		fmt.Println("\nTip: set OCTA_API_KEY to see your balance, sessions and nodes.")
		return
	}

	// ── Account ─────────────────────────────────────────────────────────────
	fmt.Println("\n=== Balance ===")
	if b, err := client.Accounts.Balance(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "accounts.balance error: %v\n", err)
	} else {
		fmt.Printf("  %s %s\n", b.Balance, b.Currency)
	}

	fmt.Println("\n=== Running sessions ===")
	if sessions, err := client.Sessions.List(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "sessions.list error: %v\n", err)
	} else {
		fmt.Printf("  %d sessions\n", len(sessions))
		for _, s := range sessions {
			fmt.Printf("  - %s  %-7s  %-8s  node %s  charged %s\n",
				s.UUID, s.Service, s.Status, s.NodeNumber, s.ChargeAmount)
		}
	}

	fmt.Println("\n=== Hosted nodes ===")
	if nodes, err := client.Hosting.Nodes(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "hosting.nodes error: %v\n", err)
	} else {
		fmt.Printf("  %d nodes online\n", len(nodes))
		for _, n := range nodes {
			fmt.Printf("  - %s  %-12s  base %s/h  min rent %d min\n",
				n.Number, n.Status, n.Prices.Base, n.Prices.MinRentMinutes)
		}
	}
}
