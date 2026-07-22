package octaspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func contractFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "contracts", name))
	if err != nil {
		t.Fatalf("read contract fixture %s: %v", name, err)
	}
	return data
}

func TestContractFixtures_DecodeTypedModels(t *testing.T) {
	var apps []App
	if err := json.Unmarshal(contractFixture(t, "apps-live.json"), &apps); err != nil {
		t.Fatalf("apps: %v", err)
	}
	if apps[0].StartCommand != "run-app" || apps[0].Envs["BOOLEAN_VALUE"] != false || len(apps[0].Ports) != 2 {
		t.Errorf("app contract fields lost: %+v", apps[0])
	}

	var machines []MachineRental
	if err := json.Unmarshal(contractFixture(t, "mr-live.json"), &machines); err != nil {
		t.Fatalf("machine rentals: %v", err)
	}
	machine := machines[0]
	if machine.TotalPriceWei.String() != "252000000000000000" || machine.Reliability.SFailed != 1 || machine.GPU.Undefined == nil {
		t.Errorf("machine-rental contract fields lost: %+v", machine)
	}

	var relays []VPNRelay
	if err := json.Unmarshal(contractFixture(t, "vpn-live.json"), &relays); err != nil {
		t.Fatalf("VPN relays: %v", err)
	}
	if relays[0].NetDownMbits != 1000000 || relays[0].CountryISO != "EX" {
		t.Errorf("VPN contract fields lost: %+v", relays[0])
	}

	var nodes []Node
	if err := json.Unmarshal(contractFixture(t, "nodes-live.json"), &nodes); err != nil {
		t.Fatalf("nodes: %v", err)
	}
	if nodes[0].System.Memory.Used != 1000 || nodes[0].System.Memory.UsedPercent != 50 || nodes[0].System.GPUs[0].Model != "Example GPU" {
		t.Errorf("node contract fields lost: %+v", nodes[0])
	}

	var sessions []Session
	if err := json.Unmarshal(contractFixture(t, "sessions-recent-live.json"), &sessions); err != nil {
		t.Fatalf("recent sessions: %v", err)
	}
	if sessions[0].StartedAt.Int64() == 0 || sessions[0].FinishedAt.Int64() == 0 || sessions[0].ChargeAmount.Int == nil {
		t.Errorf("recent-session contract fields lost: %+v", sessions[0])
	}

	var info SessionInfo
	if err := json.Unmarshal(contractFixture(t, "session-info-full.json"), &info); err != nil {
		t.Fatalf("session info: %v", err)
	}
	if info.AccountID != 404 || info.VPNConfig != "sanitized-vpn-config" || info.PortsMatrix["22"].Int64() != 10022 || info.UserEnvs["BOOLEAN"] != false {
		t.Errorf("session-info contract fields lost: %+v", info)
	}

	// Compute (MR) sessions return container_id as a JSON array (one entry per
	// container), unlike VPN sessions where the field is absent entirely. This
	// previously broke decoding of the whole /sessions list and /services/:uuid/info
	// for any account with an active compute rental — see FlexString.
	var activeSessions []Session
	if err := json.Unmarshal(contractFixture(t, "sessions-active-mr-live.json"), &activeSessions); err != nil {
		t.Fatalf("active mr sessions: %v", err)
	}
	if activeSessions[0].ContainerID.String() == "" || activeSessions[0].ChargeAmount.Int == nil {
		t.Errorf("active-mr-session contract fields lost: %+v", activeSessions[0])
	}

	var mrInfo SessionInfo
	if err := json.Unmarshal(contractFixture(t, "session-info-mr-live.json"), &mrInfo); err != nil {
		t.Fatalf("mr session info: %v", err)
	}
	if mrInfo.ContainerID.String() == "" || mrInfo.TX == 0 {
		t.Errorf("mr-session-info contract fields lost: %+v", mrInfo)
	}

	var logs SessionLogs
	if err := json.Unmarshal(contractFixture(t, "session-logs-live.json"), &logs); err != nil {
		t.Fatalf("session logs: %v", err)
	}
	if len(logs.System) != 1 || logs.System[0].TS == 0 || logs.Container == "" {
		t.Errorf("logs contract fields lost: %+v", logs)
	}
}

func TestContractFixtures_AreSanitized(t *testing.T) {
	entries, err := os.ReadDir(filepath.Join("testdata", "contracts"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data := string(contractFixture(t, entry.Name()))
		for _, forbidden := range []string{"Authorization", "api_key", "@gmail.", "@octa.", "0xC9D2", "9f26e506"} {
			if strings.Contains(data, forbidden) {
				t.Errorf("%s contains forbidden sensitive marker %q", entry.Name(), forbidden)
			}
		}
	}
}

func TestMachineReliability_AcceptsSwaggerNumericForm(t *testing.T) {
	var machine MachineRental
	if err := json.Unmarshal([]byte(`{"reliability": 85.7, "base_price":"1","storage_price":2,"traffic_price":3,"total_price":"6"}`), &machine); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if machine.Reliability.LegacyScore == nil || *machine.Reliability.LegacyScore != 85.7 {
		t.Errorf("LegacyScore = %v", machine.Reliability.LegacyScore)
	}
	if machine.BasePrice.String() != "1" || machine.TotalPrice.String() != "6" {
		t.Errorf("legacy prices were not retained: %+v", machine)
	}
}

func TestMachineReliability_AcceptsWhitespaceAroundNumericForm(t *testing.T) {
	var reliability MachineReliability
	if err := json.Unmarshal([]byte(" \n 85.7 \t"), &reliability); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if reliability.LegacyScore == nil || *reliability.LegacyScore != 85.7 {
		t.Errorf("LegacyScore = %v", reliability.LegacyScore)
	}
}
