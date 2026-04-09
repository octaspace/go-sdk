package sdk

import (
	"net/http"
	"testing"
)

func TestListNodes(t *testing.T) {
	want := []Node{
		{
			ID:    1,
			IP:    "1.2.3.4",
			State: "online",
			OSN:   "linux",
		},
		{
			ID:    2,
			IP:    "5.6.7.8",
			State: "offline",
		},
	}
	c, cleanup := withServer(t, jsonHandler(http.StatusOK, want))
	defer cleanup()

	got, err := c.ListNodes()
	if err != nil {
		t.Fatalf("ListNodes: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i, n := range got {
		if n.ID != want[i].ID {
			t.Errorf("[%d] ID = %d, want %d", i, n.ID, want[i].ID)
		}
		if n.IP != want[i].IP {
			t.Errorf("[%d] IP = %q, want %q", i, n.IP, want[i].IP)
		}
		if n.State != want[i].State {
			t.Errorf("[%d] State = %q, want %q", i, n.State, want[i].State)
		}
	}
}

func TestListNodesNestedFields(t *testing.T) {
	want := []Node{
		{
			ID:    3,
			State: "online",
		},
	}
	want[0].Location.City = "Berlin"
	want[0].Location.Country = "DE"
	want[0].Prices.Base = 10
	want[0].Prices.Storage = 2
	want[0].System.CPUCores = 16
	want[0].System.CPUModelName = "AMD EPYC"
	want[0].System.GPUs = []struct {
		Model      string `json:"model"`
		MemTotalMB int    `json:"mem_total_mb"`
		MemFreeMB  int    `json:"mem_free_mb"`
	}{
		{Model: "A100", MemTotalMB: 40960, MemFreeMB: 30000},
	}

	c, cleanup := withServer(t, jsonHandler(http.StatusOK, want))
	defer cleanup()

	got, err := c.ListNodes()
	if err != nil {
		t.Fatalf("ListNodes: %v", err)
	}
	n := got[0]
	if n.Location.City != "Berlin" {
		t.Errorf("Location.City = %q, want Berlin", n.Location.City)
	}
	if n.Prices.Base != 10 {
		t.Errorf("Prices.Base = %d, want 10", n.Prices.Base)
	}
	if n.System.CPUCores != 16 {
		t.Errorf("System.CPUCores = %d, want 16", n.System.CPUCores)
	}
	if len(n.System.GPUs) != 1 || n.System.GPUs[0].Model != "A100" {
		t.Errorf("System.GPUs unexpected: %+v", n.System.GPUs)
	}
}

func TestListNodesEmpty(t *testing.T) {
	c, cleanup := withServer(t, jsonHandler(http.StatusOK, []Node{}))
	defer cleanup()

	got, err := c.ListNodes()
	if err != nil {
		t.Fatalf("ListNodes: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty slice, got %d items", len(got))
	}
}

func TestListNodesError(t *testing.T) {
	c, cleanup := withServer(t, rawHandler(http.StatusInternalServerError, `err`))
	defer cleanup()

	_, err := c.ListNodes()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListNodesRaw(t *testing.T) {
	body := `[{"id":1,"state":"online"}]`
	c, cleanup := withServer(t, rawHandler(http.StatusOK, body))
	defer cleanup()

	got, err := c.ListNodesRaw()
	if err != nil {
		t.Fatalf("ListNodesRaw: %v", err)
	}
	if string(got) != body {
		t.Fatalf("body = %q, want %q", got, body)
	}
}

func TestListNodesRawError(t *testing.T) {
	c, cleanup := withServer(t, rawHandler(http.StatusForbidden, `forbidden`))
	defer cleanup()

	_, err := c.ListNodesRaw()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
