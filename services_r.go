package octaspace

import (
	"context"
	"fmt"
	"net/url"
)

// RenderWorker is an available worker returned by the distinct /services/r
// render-worker marketplace. Its machine fields use the same wire format as a
// machine rental listing.
type RenderWorker = MachineRental

// RenderWorkerCreateParams selects one or more render-worker nodes to rent.
// OrganizationID and ProjectID are forwarded when the account is operating in
// an organization context.
type RenderWorkerCreateParams struct {
	NodeIDs        []int64
	OrganizationID string
	ProjectID      string
}

// RenderWorkerCreateResponse contains the result for each requested worker.
// UUID is the first successful session UUID for convenience.
type RenderWorkerCreateResponse = ServiceCreateResponse

type renderWorkerCreateItem struct {
	ID             int    `json:"id"`
	NodeID         int64  `json:"node_id"`
	OrganizationID string `json:"organization_id,omitempty"`
	ProjectID      string `json:"project_id,omitempty"`
}

// RenderWorkerResource provides access to the /services/r render-worker API.
// Obtain it through [ServicesResource.R].
type RenderWorkerResource struct{ c *Client }

// List returns render workers currently available for rent.
//
// GET /services/r
func (r *RenderWorkerResource) List(ctx context.Context, params url.Values) ([]RenderWorker, error) {
	var workers []RenderWorker
	return workers, r.c.get(ctx, "/services/r", params, &workers)
}

// Create rents the selected render workers. The API receives an indexed array
// and returns one result per requested node, including partial rejections.
//
// POST /services/r
func (r *RenderWorkerResource) Create(ctx context.Context, params *RenderWorkerCreateParams) (*RenderWorkerCreateResponse, error) {
	if params == nil {
		return nil, fmt.Errorf("octaspace: RenderWorkerCreateParams must not be nil")
	}
	if len(params.NodeIDs) == 0 {
		return nil, fmt.Errorf("octaspace: RenderWorkerCreateParams.NodeIDs must not be empty")
	}

	items := make([]renderWorkerCreateItem, len(params.NodeIDs))
	for i, nodeID := range params.NodeIDs {
		items[i] = renderWorkerCreateItem{
			ID:             i,
			NodeID:         nodeID,
			OrganizationID: params.OrganizationID,
			ProjectID:      params.ProjectID,
		}
	}

	var response RenderWorkerCreateResponse
	return &response, r.c.post(ctx, "/services/r", items, &response)
}
