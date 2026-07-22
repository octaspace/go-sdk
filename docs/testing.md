# Testing and smoke checks

Unit tests use `httptest` and do not require a token or network access.

```bash
go test ./...
go test -race ./...
go vet ./...
```

`cmd/smoke` is a separate live API probe. It only reads public endpoints unless
`OCTA_API_KEY` is set, and reports endpoint results without printing account
values. Run it explicitly:

```bash
go run ./cmd/smoke
OCTA_API_KEY=<key> go run ./cmd/smoke -json
go run ./cmd/smoke -only network.info
```

No integration test creates a machine-rental session by default. The
object-form `CreateSingle` integration check is opt-in and selects the cheapest
eligible machine under an explicit USD/hour cap. It stops the created session
with an independent two-minute cleanup context and waits until it leaves active
sessions.

```bash
OCTA_API_KEY=<key> OCTA_RUN_CREATE_SINGLE_INTEGRATION=1 \
  go test -run '^TestMachineRentalCreateSingleIntegration$' -count=1
```

For diagnosing an object-form failure, a separately gated array-form control
uses the same candidate-selection and cleanup rules:

```bash
OCTA_API_KEY=<key> OCTA_RUN_CREATE_ARRAY_CONTROL_INTEGRATION=1 \
  go test -run '^TestMachineRentalCreateArrayControlIntegration$' -count=1
```

Safe object/array rejection contracts use a nonexistent node and do not create
a billed session:

```bash
OCTA_API_KEY=<key> OCTA_RUN_CREATE_REJECTION_INTEGRATION=1 \
  go test -run '^TestMachineRentalCreateRejectionIntegration$' -count=1
```
