package octaspace

import (
	"context"
	"strings"
	"testing"
)

func TestApps_List(t *testing.T) {
	want := []App{
		{UUID: "app-1", Name: "Jupyter", Image: "jupyter:latest", Category: "ml"},
		{UUID: "app-2", Name: "VS Code", Category: "dev"},
	}
	c, _ := newTestClient(t, jsonHandler(want))
	got, err := c.Apps.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i, a := range got {
		if a.UUID != want[i].UUID {
			t.Errorf("[%d] UUID = %q, want %q", i, a.UUID, want[i].UUID)
		}
		if a.Name != want[i].Name {
			t.Errorf("[%d] Name = %q, want %q", i, a.Name, want[i].Name)
		}
		if a.Category != want[i].Category {
			t.Errorf("[%d] Category = %q, want %q", i, a.Category, want[i].Category)
		}
	}
}

func TestApps_List_Empty(t *testing.T) {
	c, _ := newTestClient(t, jsonHandler([]App{}))
	got, err := c.Apps.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty slice, got %d items", len(got))
	}
}

func TestApps_List_Error(t *testing.T) {
	c, _ := newTestClient(t, statusHandler(401, `{"error":"unauthorized"}`))
	_, err := c.Apps.List(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsAuthError(err) {
		t.Errorf("expected AuthenticationError, got %T", err)
	}
}

func TestApps_List_FullPayload(t *testing.T) {
	// Simulates the current live /apps payload, which string-encodes ports arrays.
	payload := `[{
		"uuid": "app-full",
		"name": "Jupyter Lab",
		"image": "jupyter/datascience:latest",
		"category": "ml",
		"description": "GPU-accelerated Jupyter Lab",
		"envs": {"JUPYTER_TOKEN": ""},
		"ports": "[8888]",
		"http_ports": "[8888]",
		"personal": false,
		"extra": {"min_disk_size": 20, "gpu_required": true}
	}]`
	c, _ := newTestClient(t, statusHandler(200, payload))
	apps, err := c.Apps.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(apps) != 1 {
		t.Fatalf("len = %d, want 1", len(apps))
	}
	a := apps[0]
	if a.UUID != "app-full" {
		t.Errorf("UUID = %q", a.UUID)
	}
	if a.Description != "GPU-accelerated Jupyter Lab" {
		t.Errorf("Description = %q", a.Description)
	}
	if len(a.Ports) != 1 || a.Ports[0] != 8888 {
		t.Errorf("Ports = %v", a.Ports)
	}
	if len(a.HTTPPorts) != 1 || a.HTTPPorts[0] != 8888 {
		t.Errorf("HTTPPorts = %v", a.HTTPPorts)
	}
	if a.Personal {
		t.Error("Personal = true, want false")
	}
	if a.Extra == nil {
		t.Fatal("Extra is nil")
	}
	if v, ok := a.Extra["min_disk_size"]; !ok || v.(float64) != 20 {
		t.Errorf("Extra[min_disk_size] = %v", v)
	}
	if _, ok := a.Envs["JUPYTER_TOKEN"]; !ok {
		t.Error("Envs missing JUPYTER_TOKEN key")
	}
}

func TestApps_List_ArrayPortsPayload(t *testing.T) {
	payload := `[{
		"uuid": "app-array",
		"name": "Array App",
		"ports": [8080, 8081],
		"http_ports": [80]
	}]`
	c, _ := newTestClient(t, statusHandler(200, payload))
	apps, err := c.Apps.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(apps) != 1 {
		t.Fatalf("len = %d, want 1", len(apps))
	}
	a := apps[0]
	if len(a.Ports) != 2 || a.Ports[0] != 8080 || a.Ports[1] != 8081 {
		t.Errorf("Ports = %v", a.Ports)
	}
	if len(a.HTTPPorts) != 1 || a.HTTPPorts[0] != 80 {
		t.Errorf("HTTPPorts = %v", a.HTTPPorts)
	}
}

func TestApps_List_EmptyAndNullStringPorts(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{
			name: "empty strings",
			payload: `[{
				"uuid": "app-empty",
				"name": "Empty Ports App",
				"ports": "",
				"http_ports": ""
			}]`,
		},
		{
			name: "nulls",
			payload: `[{
				"uuid": "app-null",
				"name": "Null Ports App",
				"ports": null,
				"http_ports": null
			}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := newTestClient(t, statusHandler(200, tt.payload))
			apps, err := c.Apps.List(context.Background())
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(apps) != 1 {
				t.Fatalf("len = %d, want 1", len(apps))
			}
			if len(apps[0].Ports) != 0 {
				t.Errorf("Ports = %v, want empty", apps[0].Ports)
			}
			if len(apps[0].HTTPPorts) != 0 {
				t.Errorf("HTTPPorts = %v, want empty", apps[0].HTTPPorts)
			}
		})
	}
}

func TestApps_List_MalformedStringPorts(t *testing.T) {
	payload := `[{
		"uuid": "app-bad",
		"name": "Bad Ports App",
		"ports": "not-json",
		"http_ports": "[]"
	}]`
	c, _ := newTestClient(t, statusHandler(200, payload))
	_, err := c.Apps.List(context.Background())
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
	if !strings.Contains(err.Error(), "decode ports") {
		t.Fatalf("error = %q, want decode ports context", err.Error())
	}
}
