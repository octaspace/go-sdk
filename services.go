package octaspace

import "net/url"

// Service kind constants — these are the values returned in [Session.Service]
// and used as URL path segments for service-specific endpoints.
const (
	ServiceMR     = "mr"     // machine rental (compute)
	ServiceVPN    = "vpn"    // VPN relay
	ServiceRender = "render" // render job
)

// ServicesResource aggregates all service sub-resources and per-session operations.
// Obtain via [Client.Services].
//
// Example:
//
//	// Start a machine rental
//	resp, err := client.Services.MR.Create(ctx, &octaspace.MachineRentalCreateParams{...})
//
//	// Inspect a running session
//	info, err := client.Services.Session("uuid").Info(ctx)
type ServicesResource struct {
	c      *Client
	MR     *MachineRentalResource
	VPN    *VPNResource
	Render *RenderResource
}

// Session returns a [*SessionProxy] for operations on a specific service session.
// The UUID is URL-encoded automatically.
func (r *ServicesResource) Session(uuid string) *SessionProxy {
	return &SessionProxy{c: r.c, uuid: url.PathEscape(uuid)}
}
