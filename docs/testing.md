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

No integration test creates a session by default. The live compute checks run
only with `OCTA_RUN_COMPUTE_INTEGRATION=1` and an API key, against
`OCTA_API_URL` (default: the `BaseURLDev` stand).

`TestComputeSessionIntegration` starts a **billed** session and takes it
through its life: start, `WaitReady`, `List`, `Logs`, `Restart`, the restart
cooldown, and `Stop`. It uses `OCTA_NODE` if set, otherwise the cheapest machine
without a minimum rent under $0.05/h, and always stops the session with an
independent cleanup context, waiting until the API no longer reports it.
`TestComputeRejectionIntegration` sends invalid start requests and starts
nothing.

```bash
OCTA_API_KEY=<key> OCTA_RUN_COMPUTE_INTEGRATION=1 OCTA_NODE=<node-number> \
  go test -run 'Integration$' -count=1 -v
```

## CI on macOS

The SDK has no platform-specific code, so CI tests pull requests on Linux
only. The macOS job runs on pushes to `main` and on any pull request labelled
`ci:macos`; adding the label starts it.
