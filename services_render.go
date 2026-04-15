package octaspace

import (
	"context"
	"net/url"
)

// RenderResource provides access to render service endpoints.
// Obtain via [Client.Services.Render].
type RenderResource struct{ c *Client }

// List returns available render jobs or configurations.
// Pass nil or an empty [url.Values] to list all.
//
// GET /services/render
func (r *RenderResource) List(ctx context.Context, params url.Values) ([]map[string]any, error) {
	var result []map[string]any
	return result, r.c.get(ctx, "/services/render", params, &result)
}

// Create starts a new render job.
// attrs may contain any fields accepted by the render API.
//
// POST /services/render
func (r *RenderResource) Create(ctx context.Context, attrs map[string]any) (map[string]any, error) {
	var resp map[string]any
	return resp, r.c.post(ctx, "/services/render", attrs, &resp)
}
