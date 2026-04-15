package octaspace

import (
	"context"
	"fmt"
)

// IdleJobsResource provides access to idle job endpoints.
// Obtain via [Client.IdleJobs].
type IdleJobsResource struct{ c *Client }

// Find returns the status of an idle job identified by nodeID and jobID.
//
// GET /idle_jobs/:node_id/:job_id
func (r *IdleJobsResource) Find(ctx context.Context, nodeID, jobID int64) (map[string]any, error) {
	var result map[string]any
	return result, r.c.get(ctx, fmt.Sprintf("/idle_jobs/%d/%d", nodeID, jobID), nil, &result)
}

// Logs returns the log output of an idle job as raw bytes.
//
// GET /idle_jobs/:node_id/:job_id/logs
func (r *IdleJobsResource) Logs(ctx context.Context, nodeID, jobID int64) ([]byte, error) {
	return r.c.getRaw(ctx, fmt.Sprintf("/idle_jobs/%d/%d/logs", nodeID, jobID))
}
