package octaspace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"testing"
	"time"
)

const (
	createSingleIntegrationFlag = "OCTA_RUN_CREATE_SINGLE_INTEGRATION"
	createArrayControlFlag      = "OCTA_RUN_CREATE_ARRAY_CONTROL_INTEGRATION"
	createRejectionFlag         = "OCTA_RUN_CREATE_REJECTION_INTEGRATION"
	createSingleMaxHourlyUSD    = 5.00
	createSingleMinDiskBytes    = 10 * 1024 * 1024 * 1024
)

// statusRecorder records the service-create HTTP status without retaining any
// request or response body (which could contain credentials or a session UUID).
type statusRecorder struct {
	base   http.RoundTripper
	status int
}

func (r *statusRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := r.base.RoundTrip(req)
	if err == nil && req.Method == http.MethodPost && req.URL.Path == "/services/mr" {
		r.status = resp.StatusCode
	}
	return resp, err
}

// TestMachineRentalCreateSingleIntegration verifies the documented object form
// against the live API. It is deliberately opt-in because it creates a billed
// session. The test selects the cheapest suitable machine below the explicit
// hourly cap, stops it with an independent cleanup context, and verifies that
// it leaves the active sessions list.
func TestMachineRentalCreateSingleIntegration(t *testing.T) {
	if os.Getenv(createSingleIntegrationFlag) != "1" {
		t.Skipf("set %s=1 to run the billed CreateSingle integration test", createSingleIntegrationFlag)
	}
	apiKey := os.Getenv("OCTA_API_KEY")
	if apiKey == "" {
		t.Skip("OCTA_API_KEY is not set")
	}

	recorder := &statusRecorder{base: http.DefaultTransport}
	client := NewClient(apiKey,
		WithHTTPClient(&http.Client{Transport: recorder}),
		WithMaxRetries(0),
		WithTimeout(90*time.Second),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	machines, err := client.Services.MR.List(ctx, nil)
	if err != nil {
		t.Fatalf("list machines: %v", err)
	}
	candidate, ok := cheapestCreateSingleCandidate(machines)
	if !ok {
		t.Fatalf("no machine with at least %d GiB free disk below $%.2f/hour", createSingleMinDiskBytes/(1024*1024*1024), createSingleMaxHourlyUSD)
	}
	t.Logf("selected node %d at $%.4f/hour", candidate.NodeID, candidate.TotalPriceUSD)
	app, err := createIntegrationApp(ctx, client)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("selected application %q", app.Name)

	result, err := client.Services.MR.CreateSingle(ctx, createIntegrationParams(candidate.NodeID, app))
	if err != nil {
		t.Fatalf("CreateSingle: %v", err)
	}
	if recorder.status != http.StatusCreated {
		t.Errorf("CreateSingle HTTP status = %d, want %d", recorder.status, http.StatusCreated)
	}
	if result.UUID == "" {
		t.Fatal("CreateSingle returned an empty UUID")
	}
	cleaned := false
	defer func() {
		if !cleaned {
			if err := stopAndVerifySession(client, result.UUID); err != nil {
				t.Errorf("deferred cleanup: %v", err)
			}
		}
	}()

	if err := inspectLiveSessionContract(t, ctx, client, result.UUID); err != nil {
		t.Fatalf("inspect live session contract: %v", err)
	}
	if err := stopAndVerifySession(client, result.UUID); err != nil {
		t.Fatal(err)
	}
	cleaned = true
}

// TestMachineRentalCreateArrayControlIntegration verifies that the existing
// production array form accepts the same lowest-cost node and image selected
// for the object-form test. It is a diagnostic control for CreateSingle and is
// separately opt-in because it creates a billed session.
func TestMachineRentalCreateArrayControlIntegration(t *testing.T) {
	if os.Getenv(createArrayControlFlag) != "1" {
		t.Skipf("set %s=1 to run the billed array-form control test", createArrayControlFlag)
	}
	apiKey := os.Getenv("OCTA_API_KEY")
	if apiKey == "" {
		t.Skip("OCTA_API_KEY is not set")
	}

	recorder := &statusRecorder{base: http.DefaultTransport}
	client := NewClient(apiKey,
		WithHTTPClient(&http.Client{Transport: recorder}),
		WithMaxRetries(0),
		WithTimeout(90*time.Second),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	machines, err := client.Services.MR.List(ctx, nil)
	if err != nil {
		t.Fatalf("list machines: %v", err)
	}
	candidate, ok := cheapestCreateSingleCandidate(machines)
	if !ok {
		t.Fatalf("no machine with at least %d GiB free disk below $%.2f/hour", createSingleMinDiskBytes/(1024*1024*1024), createSingleMaxHourlyUSD)
	}
	t.Logf("selected node %d at $%.4f/hour", candidate.NodeID, candidate.TotalPriceUSD)
	app, err := createIntegrationApp(ctx, client)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("selected application %q", app.Name)

	result, err := client.Services.MR.Create(ctx, createIntegrationParams(candidate.NodeID, app))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if recorder.status != http.StatusOK {
		t.Errorf("Create HTTP status = %d, want %d", recorder.status, http.StatusOK)
	}
	if result.UUID == "" {
		t.Fatal("Create returned no successful UUID")
	}
	cleaned := false
	defer func() {
		if !cleaned {
			if err := stopAndVerifySession(client, result.UUID); err != nil {
				t.Errorf("deferred cleanup: %v", err)
			}
		}
	}()
	if err := stopAndVerifySession(client, result.UUID); err != nil {
		t.Fatal(err)
	}
	cleaned = true
}

// TestMachineRentalCreateRejectionIntegration verifies both documented object
// rejection and production per-item array rejection without creating a session.
func TestMachineRentalCreateRejectionIntegration(t *testing.T) {
	if os.Getenv(createRejectionFlag) != "1" {
		t.Skipf("set %s=1 to run live create rejection checks", createRejectionFlag)
	}
	apiKey := os.Getenv("OCTA_API_KEY")
	if apiKey == "" {
		t.Skip("OCTA_API_KEY is not set")
	}
	recorder := &statusRecorder{base: http.DefaultTransport}
	client := NewClient(apiKey,
		WithHTTPClient(&http.Client{Transport: recorder}),
		WithMaxRetries(0),
		WithTimeout(30*time.Second),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	app, err := createIntegrationApp(ctx, client)
	if err != nil {
		t.Fatal(err)
	}
	params := createIntegrationParams(-1, app)

	_, err = client.Services.MR.CreateSingle(ctx, params)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("object rejection = %v, want APIError 400", err)
	}
	if recorder.status != http.StatusBadRequest {
		t.Errorf("object rejection HTTP status = %d, want 400", recorder.status)
	}

	recorder.status = 0
	response, err := client.Services.MR.Create(ctx, params)
	if err != nil {
		t.Fatalf("array rejection: %v", err)
	}
	if recorder.status != http.StatusOK {
		t.Errorf("array rejection HTTP status = %d, want 200", recorder.status)
	}
	if response.UUID != "" || len(response.Results) != 1 {
		t.Fatal("array rejection unexpectedly contained a successful session")
	}
	result := response.Results[0]
	if result.Status == nil || (result.Reason == "" && result.Message == "") {
		t.Errorf("array rejection did not preserve status/reason/message presence")
	}
}

func createIntegrationApp(ctx context.Context, client *Client) (App, error) {
	apps, err := client.Apps.List(ctx)
	if err != nil {
		return App{}, fmt.Errorf("list applications: %w", err)
	}
	for _, preferred := range []string{"Ubuntu 24.04 LTS", "Ubuntu 22.04 LTS"} {
		for _, app := range apps {
			if app.Name == preferred {
				return app, nil
			}
		}
	}
	return App{}, fmt.Errorf("no supported Ubuntu application template found")
}

func createIntegrationParams(nodeID int64, app App) *MachineRentalCreateParams {
	envs := make(map[string]string, len(app.Envs))
	for key, value := range app.Envs {
		if text, ok := value.(string); ok {
			envs[key] = text
		}
	}
	return &MachineRentalCreateParams{
		NodeID:       nodeID,
		DiskSize:     10,
		Image:        app.Image,
		App:          app.UUID,
		Envs:         envs,
		Ports:        app.Ports,
		HTTPPorts:    app.HTTPPorts,
		StartCommand: app.StartCommand,
	}
}

func inspectLiveSessionContract(t *testing.T, ctx context.Context, client *Client, uuid string) error {
	t.Helper()
	infoPath := "/services/" + url.PathEscape(uuid) + "/info"
	infoRaw, err := waitForRawEndpoint(ctx, client, infoPath)
	if err != nil {
		return err
	}
	t.Logf("live info contract: %s", jsonKeyTypes(infoRaw))
	var infoObject map[string]json.RawMessage
	if err := json.Unmarshal(infoRaw, &infoObject); err != nil {
		return fmt.Errorf("decode info raw JSON: %w", err)
	}
	if started := infoObject["started_at"]; len(started) == 0 || strings.HasPrefix(strings.TrimSpace(string(started)), `"`) {
		return fmt.Errorf("info started_at is absent or not numeric")
	}
	info, err := client.Services.Session(uuid).Info(ctx)
	if err != nil {
		return fmt.Errorf("typed session info: %w", err)
	}
	if info.StartedAt.Int64() == 0 {
		return fmt.Errorf("typed session info lost started_at")
	}

	sessionsRaw, err := client.Sessions.ListRaw(ctx, nil)
	if err != nil {
		return fmt.Errorf("active sessions raw: %w", err)
	}
	var sessions []map[string]json.RawMessage
	if err := json.Unmarshal(sessionsRaw, &sessions); err != nil {
		return fmt.Errorf("decode active sessions raw: %w", err)
	}
	found := false
	for _, session := range sessions {
		var candidate string
		_ = json.Unmarshal(session["uuid"], &candidate)
		if candidate != uuid {
			continue
		}
		found = true
		encoded, _ := json.Marshal(session)
		t.Logf("live active-session contract: %s", jsonKeyTypes(encoded))
		if started := session["started_at"]; len(started) == 0 || strings.HasPrefix(strings.TrimSpace(string(started)), `"`) {
			return fmt.Errorf("active-session started_at is absent or not numeric")
		}
		break
	}
	if !found {
		return fmt.Errorf("created session not found in active-session response")
	}

	logsRaw, err := waitForRawEndpoint(ctx, client, "/services/"+url.PathEscape(uuid)+"/logs")
	if err != nil {
		return err
	}
	t.Logf("live logs contract: %s", jsonKeyTypes(logsRaw))
	var logs SessionLogs
	if err := json.Unmarshal(logsRaw, &logs); err != nil {
		return fmt.Errorf("decode typed logs: %w", err)
	}
	return nil
}

func waitForRawEndpoint(ctx context.Context, client *Client, path string) ([]byte, error) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		body, err := client.getRawParams(ctx, path, nil)
		if err == nil {
			return body, nil
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("wait for %s: %w", path, ctx.Err())
		case <-ticker.C:
		}
	}
}

func jsonKeyTypes(data []byte) string {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil {
		return "invalid-json"
	}
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+":"+rawJSONType(object[key]))
	}
	return strings.Join(parts, ",")
}

func rawJSONType(raw json.RawMessage) string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return "missing"
	}
	switch trimmed[0] {
	case '{':
		return "object"
	case '[':
		return "array"
	case '"':
		return "string"
	case 't', 'f':
		return "boolean"
	case 'n':
		return "null"
	default:
		return "number"
	}
}

func stopAndVerifySession(client *Client, uuid string) error {
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cleanupCancel()
	if err := client.Services.Session(uuid).Stop(cleanupCtx, nil); err != nil {
		return fmt.Errorf("stop created session: %w", err)
	}
	return waitForInactiveSession(cleanupCtx, client, uuid)
}

func cheapestCreateSingleCandidate(machines []MachineRental) (MachineRental, bool) {
	var cheapest MachineRental
	found := false
	for _, machine := range machines {
		if machine.TotalPriceUSD <= 0 || machine.TotalPriceUSD > createSingleMaxHourlyUSD || machine.FreeDisk < createSingleMinDiskBytes {
			continue
		}
		if !found || machine.TotalPriceUSD < cheapest.TotalPriceUSD {
			cheapest = machine
			found = true
		}
	}
	return cheapest, found
}

func waitForInactiveSession(ctx context.Context, client *Client, uuid string) error {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		sessions, err := client.Sessions.List(ctx, nil)
		if err != nil {
			return fmt.Errorf("list active sessions after stop: %w", err)
		}
		active := false
		for _, session := range sessions {
			if session.UUID == uuid {
				active = true
				break
			}
		}
		if !active {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("created session remained active after stop: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}
