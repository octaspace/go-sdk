package octaspace

import (
	"context"
	"errors"
	"net/http"
	"os"
	"testing"
)

func TestRenderMachine_FixtureCoversEveryField(t *testing.T) {
	var machines []RenderMachine
	decodeStrict(t, "testdata/next/services_render.json", &machines)
	if len(machines) != 2 {
		t.Fatalf("len = %d, want 2", len(machines))
	}
	for _, m := range machines {
		if m.NodeNumber == "" || m.BasePrice <= 0 || m.BlenderScore <= 0 || len(m.GPUs) == 0 {
			t.Errorf("machine = %+v", m)
		}
	}
}

func TestRender_List(t *testing.T) {
	fixture, err := os.ReadFile("testdata/next/services_render.json")
	if err != nil {
		t.Fatal(err)
	}
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/services/render" {
			t.Errorf("%s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write(fixture)
	})
	got, err := c.Render.List(context.Background())
	if err != nil || len(got) != 2 {
		t.Fatalf("List = %d, %v", len(got), err)
	}
	if _, err := c.Render.ListRaw(context.Background()); err != nil {
		t.Errorf("ListRaw: %v", err)
	}
}

func TestRender_Start(t *testing.T) {
	var method, path string
	var body []byte
	c, _ := newTestClient(t, captureBody(t, &method, &path, &body, 201, `{"uuid":"r-1"}`))
	uuid, err := c.Render.Start(context.Background(), "7KQ4-M2XD-4")
	if err != nil || uuid != "r-1" {
		t.Fatalf("Start = %q, %v", uuid, err)
	}
	if method != http.MethodPost || path != "/services/render" || string(body) != `{"node_number":"7KQ4-M2XD-4"}` {
		t.Errorf("request = %s %s %s", method, path, body)
	}
	if _, err := c.Render.Start(context.Background(), ""); err == nil {
		t.Error("empty node number should be rejected before the request")
	}
}

func TestRender_Start_Unavailable(t *testing.T) {
	c, _ := newTestClient(t, statusHandler(503, `{"message":"Service is temporarily unavailable"}`))
	_, err := c.Render.Start(context.Background(), "7KQ4-M2XD-4")
	var se *ServerError
	if !errors.As(err, &se) || se.StatusCode != 503 {
		t.Errorf("err = %v, want ServerError 503", err)
	}
}

func TestRender_StartBatch(t *testing.T) {
	var method, path string
	var body []byte
	c, _ := newTestClient(t, captureBody(t, &method, &path, &body, 200,
		`[{"id":"a","status":0,"uuid":"r-1"},{"id":"b","status":1,"reason":"Node not available"}]`))
	got, err := c.Render.StartBatch(context.Background(), []RenderBatchItem{
		{ID: "a", NodeNumber: "7KQ4-M2XD-4"}, {ID: "b", NodeNumber: "3HW9-P6TR-8"},
	})
	if err != nil || len(got) != 2 || !got[0].OK() || got[1].OK() {
		t.Fatalf("StartBatch = %+v, %v", got, err)
	}
	if string(body) != `[{"id":"a","node_number":"7KQ4-M2XD-4"},{"id":"b","node_number":"3HW9-P6TR-8"}]` {
		t.Errorf("body = %s", body)
	}
	if _, err := c.Render.StartBatch(context.Background(), nil); err == nil {
		t.Error("empty batch should be rejected before the request")
	}
}
