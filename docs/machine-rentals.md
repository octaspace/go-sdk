# Machine rentals

Machine-rental APIs currently expose two intentionally separate create forms.
Choose one explicitly; the SDK never changes one into the other.

## Production-compatible batch form

`Create` preserves the existing production request: a single-item JSON array
with `id: 0`. Its response is a `ServiceCreateResponse`, including per-item
results and the convenience `UUID` of the first success.

```go
result, err := client.Services.MR.Create(ctx, &octaspace.MachineRentalCreateParams{
	NodeID: 42, DiskSize: 50, Image: "ubuntu:24.04",
	Ports: []int{22}, HTTPPorts: []int{8888},
})
if err != nil { return err }
fmt.Println(result.UUID)
```

Inspect `result.Results` for partial rejections; each item can contain
`id`, `uuid`, `status`, `reason`, and `message`.

## Documented single-object form

`CreateSingle` sends a JSON object without `id` and reads the documented
`201 {"uuid":"..."}` response.

```go
result, err := client.Services.MR.CreateSingle(ctx, &octaspace.MachineRentalCreateParams{
	NodeID: 42, DiskSize: 50, Image: "ubuntu:24.04",
	StartCommand: "jupyter lab --ip=0.0.0.0",
})
if err != nil { return err }
fmt.Println(result.UUID)
```

The single-object endpoint passed an opt-in production contract check on
2026-07-22: it returned HTTP 201, and the temporary session was stopped and
confirmed absent from the active-session list. `CreateSingle` rejects
`Entrypoint`, `OrganizationID`, and `ProjectID` locally because the current
Swagger object schema does not advertise them; use `Create` for those fields.

## Listing machines

`client.Services.MR.List(ctx, params)` returns `MachineRental` values. Wei,
Ether, and USD prices remain distinct. The model supports both flat `gpus` and
vendor-grouped legacy `gpu` payloads. For server-shaped output, use `ListRaw`.
