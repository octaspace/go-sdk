package octaspace

import (
	"context"
	"testing"
)

func TestRender_List(t *testing.T) {
	want := []map[string]any{
		{"uuid": "render-1", "status": "queued"},
		{"uuid": "render-2", "status": "running"},
	}
	c, _ := newTestClient(t, jsonHandler(want))
	got, err := c.Services.Render.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0]["uuid"] != "render-1" {
		t.Errorf("got[0].uuid = %v, want render-1", got[0]["uuid"])
	}
}

func TestRender_List_Error(t *testing.T) {
	c, _ := newTestClient(t, statusHandler(401, `{"error":"unauthorized"}`))
	_, err := c.Services.Render.List(context.Background(), nil)
	if !IsAuthError(err) {
		t.Errorf("expected AuthenticationError, got %T: %v", err, err)
	}
}

func TestRender_Create(t *testing.T) {
	want := map[string]any{"uuid": "created-render-uuid", "status": "queued"}
	c, _ := newTestClient(t, jsonHandler(want))
	got, err := c.Services.Render.Create(context.Background(), map[string]any{
		"node_id": 42,
		"image":   "blender:latest",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got["uuid"] != "created-render-uuid" {
		t.Errorf("uuid = %v, want created-render-uuid", got["uuid"])
	}
}

func TestRender_Create_Error(t *testing.T) {
	c, _ := newTestClient(t, statusHandler(422, `{"error":"validation failed"}`))
	_, err := c.Services.Render.Create(context.Background(), map[string]any{})
	var valErr *ValidationError
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	_ = valErr
}
