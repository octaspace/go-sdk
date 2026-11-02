package octaspace

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSession_ReadyFixtureCoversEveryField(t *testing.T) {
	var s Session
	decodeStrict(t, "testdata/next/session_compute_ready.json", &s)
	if !s.IsCompute() || !s.IsReady() || s.UUID == "" || s.NodeNumber != "M95S-WZG8-7" {
		t.Errorf("session = %+v", s)
	}
	if s.BasePrice != 100 || s.StoragePrice != 100 || s.TrafficPrice != 100 || s.MinRentMinutes != 0 {
		t.Errorf("prices = %v %v %v %d", s.BasePrice, s.StoragePrice, s.TrafficPrice, s.MinRentMinutes)
	}
	if s.NodeHW == nil || s.NodeHW.Disk != 1 || len(s.NodeHW.GPUs) != 3 || s.NodeHW.NetDownMbits < 100 {
		t.Errorf("node_hw = %+v", s.NodeHW)
	}
	if s.SSH.Direct != nil || s.SSH.Proxy != nil || s.SSH.Web != "" {
		t.Errorf("ssh without ssh requested = %+v", s.SSH)
	}
	if s.Envs["SDK_AUDIT"] != "1" || s.Ports == nil || s.URLs == nil {
		t.Errorf("envs/ports/urls = %v %v %v", s.Envs, s.Ports, s.URLs)
	}
	if got := s.StartedTime(); got.UnixMilli() != s.StartedAt || got.Year() < 2026 {
		t.Errorf("StartedTime = %v", got)
	}
}

func TestSession_StartingFixtureHasNulls(t *testing.T) {
	var s Session
	decodeStrict(t, "testdata/next/session_compute_starting.json", &s)
	if s.IsReady() || s.Status != SessionStarting || s.Progress == "" {
		t.Errorf("status = %q %q", s.Status, s.Progress)
	}
	if s.NodeHW != nil || s.BasePrice != 0 || s.Country != "" {
		t.Errorf("fields unknown while starting should be zero: %+v", s)
	}
}

func TestSession_SSHAndRender(t *testing.T) {
	var compute Session
	err := json.Unmarshal([]byte(`{"service":"compute","uuid":"c","status":"ready","duration":1,
		"ssh":{"direct":{"host":"203.0.113.10","port":40022,"user":"root"},
		       "proxy":{"host":"proxy.example.com","port":2201,"user":"root"},
		       "web":"https://term.example.com/c"},
		"ports":{"8080":40080},"fm":"https://fm.example.com/c","pull_percent":42}`), &compute)
	if err != nil {
		t.Fatal(err)
	}
	if compute.SSH.Direct == nil || compute.SSH.Direct.Port != 40022 || compute.SSH.Proxy.User != "root" || compute.SSH.Web == "" {
		t.Errorf("ssh = %+v", compute.SSH)
	}
	if compute.Ports["8080"] != 40080 || compute.FM == "" || compute.PullPercent == nil || *compute.PullPercent != 42 {
		t.Errorf("compute = %+v", compute)
	}

	var render Session
	err = json.Unmarshal([]byte(`{"service":"render","uuid":"r","status":"ready","duration":5,"base_price":250000,
		"charge_amount":17,"node_number":"7KQ4-M2XD-4","started_at":1793567847622,
		"node_hw":{"cpu_cores":48,"total_memory":1024,"gpus":[{"model":"RTX","mem_total_mb":24576}]}}`), &render)
	if err != nil {
		t.Fatal(err)
	}
	if render.IsCompute() || render.Service != ServiceRender || render.BasePrice != 250000 || render.NodeHW.CPUCores != 48 {
		t.Errorf("render = %+v", render)
	}

	var future Session // an unknown service still decodes
	if err := json.Unmarshal([]byte(`{"service":"vpn","uuid":"v","status":"ready","duration":1,"config":"x"}`), &future); err != nil || future.Service != "vpn" {
		t.Errorf("unknown service: %+v, %v", future, err)
	}
}

func TestSessionLogs_Fixture(t *testing.T) {
	var logs SessionLogs
	decodeStrict(t, "testdata/next/session_logs.json", &logs)
	if len(logs.System) < 3 || logs.System[0].TS == 0 || logs.System[0].Time().IsZero() {
		t.Errorf("system = %+v", logs.System)
	}
	if logs.Container == nil || !strings.Contains(*logs.Container, "restarted-ok") || logs.ContainerTruncated {
		t.Errorf("container = %v truncated=%v", logs.Container, logs.ContainerTruncated)
	}
	var unreachable SessionLogs
	if err := json.Unmarshal([]byte(`{"system":[],"container":null,"container_truncated":false}`), &unreachable); err != nil || unreachable.Container != nil {
		t.Errorf("null container should stay nil: %v, %v", unreachable.Container, err)
	}
}

func TestSessions_List(t *testing.T) {
	fixture, _ := os.ReadFile("testdata/next/session_compute_ready.json")
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/sessions" || r.URL.RawQuery != "" {
			t.Errorf("%s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
		}
		_, _ = w.Write([]byte("[" + string(fixture) + "]"))
	})
	got, err := c.Sessions.List(context.Background())
	if err != nil || len(got) != 1 || got[0].NodeNumber != "M95S-WZG8-7" {
		t.Fatalf("List = %+v, %v", got, err)
	}
}

func TestSessions_Get(t *testing.T) {
	var path string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.EscapedPath()
		_, _ = w.Write([]byte(`{"service":"compute","uuid":"a/b","status":"ready","duration":1}`))
	})
	s, err := c.Sessions.Get(context.Background(), "a/b")
	if err != nil || s.UUID != "a/b" {
		t.Fatalf("Get = %+v, %v", s, err)
	}
	if path != "/services/a%2Fb/info" {
		t.Errorf("path = %q", path)
	}
	if _, err := c.Sessions.Get(context.Background(), ""); err == nil {
		t.Error("empty uuid should be rejected before the request")
	}

	c404, _ := newTestClient(t, statusHandler(404, `{"message":"Session not found"}`))
	if _, err := c404.Sessions.Get(context.Background(), "gone"); !IsNotFound(err) {
		t.Errorf("err = %v, want NotFoundError", err)
	}
	c400, _ := newTestClient(t, statusHandler(400, `{"message":"Invalid session uuid"}`))
	if _, err := c400.Sessions.Get(context.Background(), "x"); !IsValidation(err) {
		t.Errorf("err = %v, want ValidationError", err)
	}
}

func TestSessions_Logs(t *testing.T) {
	var path, query string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		path, query = r.URL.Path, r.URL.RawQuery
		_, _ = w.Write([]byte(`{"system":[{"msg":"m","ts":1}],"container":"out\r\n","container_truncated":true}`))
	})
	logs, err := c.Sessions.Logs(context.Background(), "u", &LogsParams{Tail: 50})
	if err != nil || logs.Container == nil || *logs.Container != "out\r\n" || !logs.ContainerTruncated {
		t.Fatalf("Logs = %+v, %v", logs, err)
	}
	if path != "/services/u/logs" || query != "tail=50" {
		t.Errorf("request = %s?%s", path, query)
	}
	if _, err := c.Sessions.Logs(context.Background(), "u", nil); err != nil || query != "" {
		t.Errorf("nil params: query = %q, err = %v", query, err)
	}
	if _, err := c.Sessions.Logs(context.Background(), "", nil); err == nil {
		t.Error("empty uuid should be rejected before the request")
	}
}

func TestSessions_Stop(t *testing.T) {
	var method, path string
	var body []byte
	c, _ := newTestClient(t, captureBody(t, &method, &path, &body, 202, `{"uuid":"u"}`))
	uuid, err := c.Sessions.Stop(context.Background(), "u")
	if err != nil || uuid != "u" {
		t.Fatalf("Stop = %q, %v", uuid, err)
	}
	if method != http.MethodPost || path != "/services/u/stop" || len(body) != 0 {
		t.Errorf("request = %s %s body=%q", method, path, body)
	}
	if _, err := c.Sessions.Stop(context.Background(), ""); err == nil {
		t.Error("empty uuid should be rejected before the request")
	}
}

// sessionSequence answers GET info with the given statuses in turn, then a 404
// when stopAfter is set.
func sessionSequence(t *testing.T, statuses []string, stopAfter bool) (*Client, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	c, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		n := int(calls.Add(1)) - 1
		if n >= len(statuses) {
			if stopAfter {
				w.WriteHeader(404)
				_, _ = w.Write([]byte(`{"message":"Session not found"}`))
				return
			}
			n = len(statuses) - 1
		}
		_, _ = w.Write([]byte(`{"service":"compute","uuid":"u","duration":1,"status":"` + statuses[n] + `","progress":"step ` + statuses[n] + `"}`))
	})
	return c, &calls
}

func TestSessions_WaitReady(t *testing.T) {
	c, calls := sessionSequence(t, []string{"starting", "starting", "ready"}, false)
	var seen []string
	s, err := c.Sessions.WaitReady(context.Background(), "u", &WaitOptions{
		Interval:   time.Millisecond,
		OnProgress: func(s *Session) { seen = append(seen, s.Status) },
	})
	if err != nil || !s.IsReady() {
		t.Fatalf("WaitReady = %+v, %v", s, err)
	}
	if calls.Load() != 3 || strings.Join(seen, ",") != "starting,starting,ready" {
		t.Errorf("calls = %d, progress = %v", calls.Load(), seen)
	}
}

func TestSessions_WaitReady_Stopped(t *testing.T) {
	c, _ := sessionSequence(t, []string{"starting"}, true)
	_, err := c.Sessions.WaitReady(context.Background(), "u", &WaitOptions{Interval: time.Millisecond})
	if !errors.Is(err, ErrSessionStopped) || !IsNotFound(err) {
		t.Errorf("err = %v, want ErrSessionStopped wrapping NotFoundError", err)
	}
}

func TestSessions_WaitReady_ContextDone(t *testing.T) {
	c, _ := sessionSequence(t, []string{"starting"}, false)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := c.Sessions.WaitReady(ctx, "u", &WaitOptions{Interval: 5 * time.Millisecond})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want context.DeadlineExceeded", err)
	}
}

func TestSessions_WaitReady_OtherError(t *testing.T) {
	c, _ := newTestClient(t, statusHandler(401, `{"message":"Invalid bearer token"}`))
	if _, err := c.Sessions.WaitReady(context.Background(), "u", nil); !IsAuthError(err) || errors.Is(err, ErrSessionStopped) {
		t.Errorf("err = %v, want AuthenticationError only", err)
	}
}
