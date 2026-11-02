package octaspace

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// decodeStrict decodes a fixture rejecting fields the model does not declare,
// so a model that misses an API field fails the test.
func decodeStrict(t *testing.T, path string, dst any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
}

func TestComputeMachine_FixtureCoversEveryField(t *testing.T) {
	var machines []ComputeMachine
	decodeStrict(t, "testdata/next/services_compute.json", &machines)
	if len(machines) != 3 {
		t.Fatalf("len = %d, want 3", len(machines))
	}
	var m ComputeMachine
	for _, x := range machines {
		if x.NodeNumber == "M95S-WZG8-7" {
			m = x
		}
	}
	if m.NodeNumber == "" {
		t.Fatal("M95S-WZG8-7 not in fixture")
	}
	if m.BasePrice != 100 || m.StoragePrice != 100 || m.TrafficPrice != 100 || m.MinRentMinutes != 0 {
		t.Errorf("prices = %v/%v/%v min %d", m.BasePrice, m.StoragePrice, m.TrafficPrice, m.MinRentMinutes)
	}
	if m.NetDownMbits < 100 || m.NetDownMbits > 1000 {
		t.Errorf("NetDownMbits = %v, want Mbit/s", m.NetDownMbits)
	}
	if m.DiskType != "nvme" || m.Arch != "x86_64" || len(m.GPUs) == 0 {
		t.Errorf("hardware = %q %q %d GPUs", m.DiskType, m.Arch, len(m.GPUs))
	}
	g := m.GPUs[0]
	if g.Model == "" || g.MemTotalMB == 0 || g.CoreClock == nil || g.PowerLimit == nil {
		t.Errorf("gpu = %+v", g)
	}
	if m.Reliability.Uptime <= 0 {
		t.Errorf("reliability = %+v", m.Reliability)
	}
}

func TestComputeMachine_NullsDecodeToZero(t *testing.T) {
	var m ComputeMachine
	err := json.Unmarshal([]byte(`{"node_number":"7KQ4-M2XD-4","country":null,"cuda_version":null,"disk_type":null,
		"gpus":[{"model":"X","core_clock":null,"mem_clock":null,"power_limit":null}]}`), &m)
	if err != nil {
		t.Fatal(err)
	}
	if m.Country != "" || m.CUDAVersion != "" || m.DiskType != "" {
		t.Errorf("nullable strings = %q %q %q", m.Country, m.CUDAVersion, m.DiskType)
	}
	if g := m.GPUs[0]; g.CoreClock != nil || g.MemClock != nil || g.PowerLimit != nil {
		t.Errorf("nullable ints should stay nil: %+v", g)
	}
}

func TestCompute_List(t *testing.T) {
	fixture, err := os.ReadFile("testdata/next/services_compute.json")
	if err != nil {
		t.Fatal(err)
	}
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/services/compute" {
			t.Errorf("%s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write(fixture)
	})
	got, err := c.Compute.List(context.Background())
	if err != nil || len(got) != 3 {
		t.Fatalf("List = %d machines, %v", len(got), err)
	}
	raw, err := c.Compute.ListRaw(context.Background())
	if err != nil || !bytes.Equal(raw, fixture) {
		t.Errorf("ListRaw differs from the response body: %v", err)
	}
}

// captureBody returns a handler that records the request body and answers
// with code and resp.
func captureBody(t *testing.T, gotMethod, gotPath *string, gotBody *[]byte, code int, resp string) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		*gotMethod, *gotPath = r.Method, r.URL.EscapedPath()
		b, _ := io.ReadAll(r.Body)
		*gotBody = b
		w.WriteHeader(code)
		_, _ = w.Write([]byte(resp))
	}
}

func TestCompute_Start_SendsOnlySetFields(t *testing.T) {
	var method, path string
	var body []byte
	c, _ := newTestClient(t, captureBody(t, &method, &path, &body, 201, `{"uuid":"2a63b54f-f243-45ab-8416-2728e178a384"}`))

	uuid, err := c.Compute.Start(context.Background(), &ComputeStartParams{
		NodeNumber: "M95S-WZG8-7",
		Image:      "alpine:3.20",
		Entrypoint: "sh -c",
		Args:       `"sleep 3600"`,
		Envs:       map[string]string{"A": "1"},
		SSH:        Bool(false), // explicit false must be sent
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if uuid != "2a63b54f-f243-45ab-8416-2728e178a384" {
		t.Errorf("uuid = %q", uuid)
	}
	if method != http.MethodPost || path != "/services/compute" {
		t.Errorf("request = %s %s", method, path)
	}
	want := `{"node_number":"M95S-WZG8-7","image":"alpine:3.20","envs":{"A":"1"},"entrypoint":"sh -c","args":"\"sleep 3600\"","ssh":false}`
	if string(body) != want {
		t.Errorf("body =\n%s\nwant\n%s", body, want)
	}
}

func TestCompute_Start_ValidationError(t *testing.T) {
	c, _ := newTestClient(t, statusHandler(400, `{"message":"node_number must be a node number, e.g. 7KQ4-M2XD-4"}`))
	_, err := c.Compute.Start(context.Background(), &ComputeStartParams{NodeNumber: "ZZZZ-ZZZZ-Z", Image: "alpine"})
	if !IsValidation(err) {
		t.Fatalf("err = %v, want ValidationError", err)
	}
	if _, err := c.Compute.Start(context.Background(), nil); err == nil {
		t.Error("nil params should be rejected before the request")
	}
}

func TestCompute_Start_NotRetried(t *testing.T) {
	attempts := 0
	srv := newCountingServer(t, &attempts, 503, `{"message":"Service is temporarily unavailable"}`)
	c := NewClient("k", WithBaseURL(srv), WithMaxRetries(3))
	if _, err := c.Compute.Start(context.Background(), &ComputeStartParams{NodeNumber: "n", Image: "i"}); err == nil {
		t.Fatal("expected error")
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1 (starting a service is never retried)", attempts)
	}
}

func TestCompute_StartBatch(t *testing.T) {
	var method, path string
	var body []byte
	resp := `[{"id":"1","status":0,"uuid":"eeda4f74-0ccb-420e-b718-3f0ec56e397f"},
		{"id":"two","reason":"node_number must be a node number, e.g. 7KQ4-M2XD-4","status":1},
		{"id":3,"reason":"image is required","status":1}]`
	c, _ := newTestClient(t, captureBody(t, &method, &path, &body, 200, resp))

	got, err := c.Compute.StartBatch(context.Background(), []ComputeBatchItem{
		{ID: "1", ComputeStartParams: ComputeStartParams{NodeNumber: "WJFM-MYWR-9", Image: "alpine:3.20"}},
		{ID: "two", ComputeStartParams: ComputeStartParams{NodeNumber: "ZZZZ-ZZZZ-Z", Image: "alpine:3.20"}},
	})
	if err != nil {
		t.Fatalf("StartBatch: %v", err)
	}
	wantBody := `[{"id":"1","node_number":"WJFM-MYWR-9","image":"alpine:3.20"},{"id":"two","node_number":"ZZZZ-ZZZZ-Z","image":"alpine:3.20"}]`
	if string(body) != wantBody {
		t.Errorf("body =\n%s\nwant\n%s", body, wantBody)
	}
	if len(got) != 3 {
		t.Fatalf("results = %d", len(got))
	}
	if !got[0].OK() || got[0].ID != "1" || got[0].UUID == "" {
		t.Errorf("got[0] = %+v", got[0])
	}
	if got[1].OK() || got[1].ID != "two" || got[1].Reason == "" {
		t.Errorf("got[1] = %+v", got[1])
	}
	if got[2].ID != "3" { // numeric IDs are accepted too
		t.Errorf("got[2].ID = %q, want 3", got[2].ID)
	}
	if _, err := c.Compute.StartBatch(context.Background(), nil); err == nil {
		t.Error("empty batch should be rejected before the request")
	}
}

func TestCompute_Restart(t *testing.T) {
	t.Run("nil params sends no body", func(t *testing.T) {
		var method, path string
		var body []byte
		c, _ := newTestClient(t, captureBody(t, &method, &path, &body, 202, `{"uuid":"u/1"}`))
		uuid, err := c.Compute.Restart(context.Background(), "u/1", nil)
		if err != nil || uuid != "u/1" {
			t.Fatalf("Restart = %q, %v", uuid, err)
		}
		if method != http.MethodPost || path != "/services/compute/u%2F1/restart" {
			t.Errorf("request = %s %s", method, path)
		}
		if len(body) != 0 {
			t.Errorf("body = %q, want empty", body)
		}
	})
	t.Run("params replace the session's", func(t *testing.T) {
		var method, path string
		var body []byte
		c, _ := newTestClient(t, captureBody(t, &method, &path, &body, 202, `{"uuid":"u"}`))
		_, err := c.Compute.Restart(context.Background(), "u", &ComputeRestartParams{
			Entrypoint: "sh -c", Args: `"echo ok"`, Envs: map[string]string{"A": "2"}, DiskSize: 1,
		})
		if err != nil {
			t.Fatal(err)
		}
		want := `{"disk_size":1,"envs":{"A":"2"},"entrypoint":"sh -c","args":"\"echo ok\""}`
		if string(body) != want {
			t.Errorf("body =\n%s\nwant\n%s", body, want)
		}
	})
	t.Run("cooldown and conflict", func(t *testing.T) {
		c, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Retry-After", "288")
			w.WriteHeader(429)
			_, _ = w.Write([]byte(`{"message":"Session was restarted too recently, retry in 288 seconds"}`))
		})
		_, err := c.Compute.Restart(context.Background(), "u", nil)
		var rl *RateLimitError
		if !errors.As(err, &rl) || rl.RetryAfter != 288 {
			t.Errorf("err = %v, want RateLimitError with RetryAfter 288", err)
		}
		c2, _ := newTestClient(t, statusHandler(409, `{"message":"Session is still starting or restarting"}`))
		if _, err := c2.Compute.Restart(context.Background(), "u", nil); !IsConflict(err) {
			t.Errorf("err = %v, want ConflictError", err)
		}
		if _, err := c2.Compute.Restart(context.Background(), "", nil); err == nil {
			t.Error("empty uuid should be rejected before the request")
		}
	})
}

// newCountingServer answers every request with code and body, counting them,
// and returns its URL.
func newCountingServer(t *testing.T, attempts *int, code int, body string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		*attempts++
		w.WriteHeader(code)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestMachineGPU_OptionalIntsAsStrings(t *testing.T) {
	// WJFM-MYWR-9 (Intel Arc B580) on the next stand reports missing values
	// as the string "null" instead of JSON null.
	var g MachineGPU
	err := json.Unmarshal([]byte(`{"model":"Arc B580","core_clock":2850,"mem_clock":"null","power_limit":"","tuned":true}`), &g)
	if err != nil {
		t.Fatal(err)
	}
	if g.CoreClock == nil || *g.CoreClock != 2850 || g.MemClock != nil || g.PowerLimit != nil || !g.Tuned || g.Model != "Arc B580" {
		t.Errorf("gpu = %+v", g)
	}
	if err := json.Unmarshal([]byte(`{"power_limit":"390"}`), &g); err != nil || g.PowerLimit == nil || *g.PowerLimit != 390 {
		t.Errorf("numeric string: %+v, %v", g, err)
	}
	if err := json.Unmarshal([]byte(`{"power_limit":"lots"}`), &g); err == nil {
		t.Error("a non-numeric string should be an error")
	}
}
