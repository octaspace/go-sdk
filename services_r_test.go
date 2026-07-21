package octaspace

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRenderWorker_List(t *testing.T) {
	want := []RenderWorker{{NodeID: 7, Country: "DE"}}
	c, _ := newTestClient(t, jsonHandler(want))
	got, err := c.Services.R.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 || got[0].NodeID != 7 {
		t.Errorf("List = %+v, want node 7", got)
	}
}

func TestRenderWorker_Create_WireFormatAndResults(t *testing.T) {
	var gotMethod string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotBody, _ = readBody(r)
		jsonHandler([]ServiceCreateResult{
			{ID: 0, UUID: "render-session-7"},
			{ID: 1, Reason: "node busy"},
		})(w, r)
	}))
	t.Cleanup(srv.Close)
	c := NewClient("key", WithBaseURL(srv.URL), WithMaxRetries(0))

	response, err := c.Services.R.Create(context.Background(), &RenderWorkerCreateParams{
		NodeIDs:        []int64{7, 9},
		OrganizationID: "org-1",
		ProjectID:      "project-1",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	var body []map[string]any
	if err := json.Unmarshal(gotBody, &body); err != nil {
		t.Fatalf("request body: %v", err)
	}
	if len(body) != 2 || int(body[0]["id"].(float64)) != 0 || int(body[1]["id"].(float64)) != 1 {
		t.Errorf("body ids = %v, want indexed items", body)
	}
	if body[0]["organization_id"] != "org-1" || body[0]["project_id"] != "project-1" {
		t.Errorf("body context = %v, want organization and project", body[0])
	}
	if response.UUID != "render-session-7" || len(response.Results) != 2 {
		t.Errorf("response = %+v, want one success and one rejection", response)
	}
}

func TestRenderWorker_Create_ValidatesParams(t *testing.T) {
	c := NewClient("key")
	if _, err := c.Services.R.Create(context.Background(), nil); err == nil {
		t.Fatal("Create(nil) error = nil, want error")
	}
	if _, err := c.Services.R.Create(context.Background(), &RenderWorkerCreateParams{}); err == nil {
		t.Fatal("Create(empty) error = nil, want error")
	}
}
