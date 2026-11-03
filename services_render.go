package octaspace

import (
	"context"
	"errors"
)

// RenderResource rents render nodes. Obtain via [Client.Render].
type RenderResource struct{ c *Client }

// List returns the nodes available for render: an NVIDIA GPU with a Blender
// score. Authentication is optional, as for [ComputeResource.List].
//
// GET /services/render
func (r *RenderResource) List(ctx context.Context) ([]RenderMachine, error) {
	var result []RenderMachine
	return result, r.c.get(ctx, "/services/render", nil, &result)
}

// ListRaw returns the raw JSON of GET /services/render.
func (r *RenderResource) ListRaw(ctx context.Context) ([]byte, error) {
	return r.c.getRaw(ctx, "/services/render")
}

// RenderBatchItem is one request of [RenderResource.StartBatch]. ID is chosen
// by the caller and echoed in the matching [BatchResult].
type RenderBatchItem struct {
	ID         string `json:"id"`
	NodeNumber string `json:"node_number"`
}

type renderStartRequest struct {
	NodeNumber string `json:"node_number"`
}

// Start rents a render node for the account and returns the session UUID.
// The node runs the render node until the session is stopped and is billed
// by the hour (BasePrice). Start is never retried automatically.
//
// POST /services/render
func (r *RenderResource) Start(ctx context.Context, nodeNumber string) (string, error) {
	if nodeNumber == "" {
		return "", errors.New("octaspace: node number is required")
	}
	var resp serviceUUID
	if err := r.c.post(ctx, "/services/render", renderStartRequest{NodeNumber: nodeNumber}, &resp); err != nil {
		return "", err
	}
	return resp.UUID, nil
}

// StartBatch rents several render nodes in one request; each request is
// validated on its own, as in [ComputeResource.StartBatch].
//
// POST /services/render
func (r *RenderResource) StartBatch(ctx context.Context, items []RenderBatchItem) ([]BatchResult, error) {
	if len(items) == 0 {
		return nil, errors.New("octaspace: render batch is empty")
	}
	var results []BatchResult
	return results, r.c.post(ctx, "/services/render", items, &results)
}
