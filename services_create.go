package octaspace

import (
	"bytes"
	"encoding/json"
)

// ServiceCreateResult is one result returned for an item in a service creation
// request. The API may accept several requested nodes at once and can therefore
// report a mix of successful and rejected items.
type ServiceCreateResult struct {
	ID   int    `json:"id,omitempty"`
	UUID string `json:"uuid,omitempty"`
	// Status is present in per-item responses, including successful status 0.
	// A pointer distinguishes an omitted status from that successful value.
	Status  *int   `json:"status,omitempty"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

// ServiceCreateResponse is returned by service creation endpoints that answer
// with one result per requested item.
//
// UUID is the first successful session UUID and is retained as a convenience
// for callers that create one item. Use Results for batch requests and partial
// failures. The decoder also accepts the documented single-object response.
type ServiceCreateResponse struct {
	UUID    string                `json:"uuid,omitempty"`
	Results []ServiceCreateResult `json:"results,omitempty"`
}

// UUIDs returns all successful session UUIDs in response order.
func (r ServiceCreateResponse) UUIDs() []string {
	uuid := make([]string, 0, len(r.Results))
	for _, result := range r.Results {
		if result.UUID != "" {
			uuid = append(uuid, result.UUID)
		}
	}
	if len(uuid) == 0 && r.UUID != "" {
		return []string{r.UUID}
	}
	return uuid
}

// UnmarshalJSON accepts the production per-item array response and the
// documented single-object response.
func (r *ServiceCreateResponse) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) > 0 && data[0] == '[' {
		var results []ServiceCreateResult
		if err := json.Unmarshal(data, &results); err != nil {
			return err
		}
		r.Results = results
		r.UUID = ""
		for _, result := range results {
			if result.UUID != "" {
				r.UUID = result.UUID
				break
			}
		}
		return nil
	}

	var result ServiceCreateResult
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	r.Results = []ServiceCreateResult{result}
	r.UUID = result.UUID
	return nil
}
