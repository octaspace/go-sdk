package octaspace

import "context"

// App represents an application available for deployment.
type App struct {
	UUID     string `json:"uuid"`
	Name     string `json:"name"`
	Image    string `json:"image"`
	Category string `json:"category"`
	Extra    struct {
		MinDiskSize int `json:"min_disk_size"`
	} `json:"extra"`
}

// AppsResource provides access to the application catalog endpoint.
// Obtain via [Client.Apps].
type AppsResource struct{ c *Client }

// List returns all available applications.
//
// GET /apps
func (r *AppsResource) List(ctx context.Context) ([]App, error) {
	var apps []App
	return apps, r.c.get(ctx, "/apps", nil, &apps)
}
