package sdk_test

import (
	"fmt"
	"log"

	sdk "github.com/octaspace/go-sdk"
)

func ExampleNewClient() {
	client := sdk.NewClient("your-api-key")
	_ = client
	// Output:
}

func ExampleClient_ValidateToken() {
	client := sdk.NewClient("your-api-key")

	if err := client.ValidateToken(); err != nil {
		log.Fatalf("invalid token: %v", err)
	}
	fmt.Println("token is valid")
}

func ExampleClient_GetAccount() {
	client := sdk.NewClient("your-api-key")

	account, err := client.GetAccount()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("email:   %s\n", account.Email)
	fmt.Printf("uid:     %d\n", account.UID)
	fmt.Printf("wallet:  %s\n", account.Wallet)
	fmt.Printf("balance: %s wei\n", account.Balance.String())
}

func ExampleClient_GetBalance() {
	client := sdk.NewClient("your-api-key")

	balance, err := client.GetBalance()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("balance: %s wei\n", balance.String())
}

func ExampleClient_ListNodes() {
	client := sdk.NewClient("your-api-key")

	nodes, err := client.ListNodes()
	if err != nil {
		log.Fatal(err)
	}

	for _, n := range nodes {
		fmt.Printf("node %d  state=%-8s  %s, %s  cpu=%s (%d cores)  gpus=%d\n",
			n.ID, n.State,
			n.Location.City, n.Location.Country,
			n.System.CPUModelName, n.System.CPUCores,
			len(n.System.GPUs),
		)
	}
}

func ExampleClient_ListApps() {
	client := sdk.NewClient("your-api-key")

	apps, err := client.ListApps()
	if err != nil {
		log.Fatal(err)
	}

	for _, a := range apps {
		fmt.Printf("[%s] %s  category=%s\n", a.UUID, a.Name, a.Category)
	}
}

func ExampleClient_ListMachinesForRent() {
	client := sdk.NewClient("your-api-key")

	machines, err := client.ListMachinesForRent()
	if err != nil {
		log.Fatal(err)
	}

	for _, m := range machines {
		fmt.Printf("node %d  %s, %s  cpu=%s (%d cores)  ram=%d GB  price=$%.4f/hr\n",
			m.NodeID,
			m.City, m.Country,
			m.CPUModelName, m.CPUCores,
			m.TotalMemory/1024/1024/1024,
			m.TotalPriceUSD,
		)
		for _, g := range m.GPUs {
			fmt.Printf("  gpu: %s  %d MB\n", g.Model, g.MemTotalMB)
		}
	}
}

func ExampleClient_DeployMachine() {
	client := sdk.NewClient("your-api-key")

	resp, err := client.DeployMachine(sdk.DeployRequest{
		App:      "app-uuid",   // from ListApps
		NodeID:   42,           // from ListMachinesForRent
		DiskSize: 50,           // GB
		Envs: map[string]string{
			"JUPYTER_TOKEN": "secret",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("deployed session: %s\n", resp.UUID)
}

func ExampleClient_ListSessions() {
	client := sdk.NewClient("your-api-key")

	sessions, err := client.ListSessions()
	if err != nil {
		log.Fatal(err)
	}

	for _, s := range sessions {
		status := "starting"
		if s.IsReady {
			status = "ready"
		}
		fmt.Printf("[%s] service=%-6s  node=%d  status=%s  ip=%s\n",
			s.UUID, s.Service, s.NodeID, status, s.PublicIP)
		for name, url := range s.URLs {
			fmt.Printf("  %s: %s\n", name, url)
		}
	}
}

func ExampleClient_GetSessionInfo() {
	client := sdk.NewClient("your-api-key")

	info, err := client.GetSessionInfo("session-uuid")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("tx: %d bytes  rx: %d bytes\n", info.TX, info.RX)
	fmt.Printf("charge: %s wei\n", info.ChargeAmount.String())
}

func ExampleClient_GetSessionLogs() {
	client := sdk.NewClient("your-api-key")

	logs, err := client.GetSessionLogs("session-uuid")
	if err != nil {
		log.Fatal(err)
	}

	for _, entry := range logs.System {
		fmt.Printf("[%d] %s\n", entry.TS, entry.Msg)
	}
	if logs.Container != "" {
		fmt.Println("--- container ---")
		fmt.Println(logs.Container)
	}
}

func ExampleClient_StopSession() {
	client := sdk.NewClient("your-api-key")

	if err := client.StopSession("session-uuid"); err != nil {
		log.Fatalf("stop failed: %v", err)
	}

	fmt.Println("session stopped")
}

func ExampleClient_ListVPNRelays() {
	client := sdk.NewClient("your-api-key")

	relays, err := client.ListVPNRelays()
	if err != nil {
		log.Fatal(err)
	}

	for _, r := range relays {
		residential := ""
		if r.Residential {
			residential = " (residential)"
		}
		fmt.Printf("node %d  %s, %s%s  down=%d Mbit  up=%d Mbit  $%.4f/GB\n",
			r.NodeID, r.City, r.Country, residential,
			r.DownloadSpeed, r.UploadSpeed, r.PricePerGB,
		)
	}
}

func ExampleClient_ConnectVPN() {
	client := sdk.NewClient("your-api-key")

	resp, err := client.ConnectVPN(7, "wireguard")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("vpn session: %s\n", resp.UUID)

	// Fetch WireGuard config once the session is ready.
	info, err := client.GetSessionInfo(resp.UUID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(info.VPNConfig) // paste into wg0.conf
}
