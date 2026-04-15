package octaspace

import (
	"context"
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
