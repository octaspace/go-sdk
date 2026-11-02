package octaspace

// ServicesResource groups the legacy service sub-resources. They target the
// legacy API and are removed together with it; use [Client.Compute],
// [Client.Render] and [Client.Sessions] for the new one.
// Obtain via [Client.Services].
type ServicesResource struct {
	c   *Client
	MR  *MachineRentalResource
	R   *RenderWorkerResource
	VPN *VPNResource
}
