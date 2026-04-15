package octaspace

import "context"

// NetworkResource provides access to network information endpoints.
// Obtain via [Client.Network].
type NetworkResource struct{ c *Client }

// Info returns current network statistics and configuration.
// This endpoint does not require authentication.
//
// GET /network
func (r *NetworkResource) Info(ctx context.Context) (map[string]any, error) {
	var result map[string]any
	return result, r.c.get(ctx, "/network", nil, &result)
}
