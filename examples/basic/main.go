// Basic usage example for the OctaSpace Go SDK.
//
// Without OCTA_API_KEY — only public endpoints are called.
// With OCTA_API_KEY    — all authenticated endpoints are demonstrated.
//
// Run:
//
//	go run ./examples/basic
//	OCTA_API_KEY=<key> go run ./examples/basic
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	octaspace "github.com/octaspace/go-sdk"
)

func main() {
	ctx := context.Background()
	apiKey := os.Getenv("OCTA_API_KEY")

	client := octaspace.NewClient(apiKey)

	// ── Public ──────────────────────────────────────────────────────────────
	fmt.Println("=== Network Info (no auth required) ===")
	info, err := client.Network.Info(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "network.info error: %v\n", err)
	} else {
		prettyPrint(info)
	}

	if apiKey == "" {
		fmt.Println("\nTip: set OCTA_API_KEY to access authenticated endpoints.")
		return
	}

	// ── Account ─────────────────────────────────────────────────────────────
	fmt.Println("\n=== Account Profile ===")
	account, err := client.Accounts.Profile(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "accounts.profile error: %v\n", err)
	} else {
		fmt.Printf("  UUID:    %s\n  Email:   %s\n  Wallet:  %s\n",
			account.AccountUUID, account.Email, account.Wallet)
	}

	fmt.Println("\n=== Balance ===")
	balance, err := client.Accounts.Balance(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "accounts.balance error: %v\n", err)
	} else {
		fmt.Printf("  Balance (wei): %s\n", balance.String())
	}

	// ── Apps ─────────────────────────────────────────────────────────────────
	fmt.Println("\n=== Apps ===")
	apps, err := client.Apps.List(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "apps.list error: %v\n", err)
	} else {
		fmt.Printf("  %d apps available\n", len(apps))
		for i, a := range apps {
			if i >= 5 {
				fmt.Println("  ...")
				break
			}
			fmt.Printf("  - %-30s  category=%-12s  image=%s\n", a.Name, a.Category, a.Image)
		}
	}

	// ── Nodes ────────────────────────────────────────────────────────────────
	fmt.Println("\n=== Nodes (first 5) ===")
	nodes, err := client.Nodes.List(ctx, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "nodes.list error: %v\n", err)
	} else {
		fmt.Printf("  %d nodes total\n", len(nodes))
		for i, n := range nodes {
			if i >= 5 {
				fmt.Println("  ...")
				break
			}
			fmt.Printf("  - [%4d]  %-3s  %-8s  CPUs: %2d  GPUs: %d\n",
				n.ID, n.Location.Country, n.State,
				n.System.CPUCores, len(n.System.GPUs))
		}
	}

	// ── Machines for rent ────────────────────────────────────────────────────
	fmt.Println("\n=== Machines for Rent (first 5) ===")
	machines, err := client.Services.MR.List(ctx, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "services.mr.list error: %v\n", err)
	} else {
		fmt.Printf("  %d machines available\n", len(machines))
		for i, m := range machines {
			if i >= 5 {
				fmt.Println("  ...")
				break
			}
			fmt.Printf("  - [%4d]  %-3s  %s  CPUs: %2d  GPUs: %d  $%.4f/hr\n",
				m.NodeID, m.Country, m.CPUModelName,
				m.CPUCores, len(m.GPUs), m.TotalPriceUSD)
		}
	}

	// ── VPN Relays ───────────────────────────────────────────────────────────
	fmt.Println("\n=== VPN Relays (first 5) ===")
	relays, err := client.Services.VPN.List(ctx, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "services.vpn.list error: %v\n", err)
	} else {
		fmt.Printf("  %d relays available\n", len(relays))
		for i, r := range relays {
			if i >= 5 {
				fmt.Println("  ...")
				break
			}
			fmt.Printf("  - [%4d]  %-3s  %-15s  ↓%.0f Mbps  $%.4f/GB  residential=%v\n",
				r.NodeID, r.Country, r.City,
				r.DownloadMbps(), r.PricePerGB, r.Residential)
		}
	}

	// ── Sessions ─────────────────────────────────────────────────────────────
	fmt.Println("\n=== Active Sessions ===")
	sessions, err := client.Sessions.List(ctx, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sessions.list error: %v\n", err)
	} else {
		fmt.Printf("  %d sessions\n", len(sessions))
		for i, s := range sessions {
			if i >= 5 {
				fmt.Println("  ...")
				break
			}
			fmt.Printf("  - %s  service=%-6s  ready=%-5v  node=%d\n",
				s.UUID, s.Service, s.IsReady, s.NodeID)
		}
	}
}

func prettyPrint(v any) {
	b, err := json.MarshalIndent(v, "  ", "  ")
	if err != nil {
		fmt.Printf("  %+v\n", v)
		return
	}
	fmt.Printf("  %s\n", b)
}
