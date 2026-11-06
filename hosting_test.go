package octaspace

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
)

// decodeStrictNullTolerant decodes a fixture the way getNullTolerant does
// ("null" strings become null), rejecting fields the model does not declare.
func decodeStrictNullTolerant(t *testing.T, path string, dst any) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		t.Fatal(err)
	}
	normalized, err := json.Marshal(nullStrings(v))
	if err != nil {
		t.Fatal(err)
	}
	strict := json.NewDecoder(bytes.NewReader(normalized))
	strict.DisallowUnknownFields()
	if err := strict.Decode(dst); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return data
}

func TestHostingNode_FixtureCoversEveryField(t *testing.T) {
	var nodes []HostingNode
	decodeStrictNullTolerant(t, "testdata/next/hosting_nodes.json", &nodes)
	if len(nodes) != 3 {
		t.Fatalf("len = %d", len(nodes))
	}
	for _, n := range nodes {
		if n.Number == "" || n.Status != NodeIdle || n.Verified == nil || n.OnlineAt.IsZero() || n.Uptime == nil || n.UptimePercent == nil {
			t.Errorf("node = %+v", n)
		}
		if n.System.Memory == nil || n.System.Memory.Size == 0 || n.System.Disk == nil || len(n.System.GPUs) == 0 {
			t.Errorf("system = %+v", n.System)
		}
		if !n.Services["compute"].Offered || n.Location.City == "" {
			t.Errorf("services/location = %+v %+v", n.Services, n.Location)
		}
	}
}

func TestHosting_Nodes_ToleratesNullStrings(t *testing.T) {
	fixture, err := os.ReadFile("testdata/next/hosting_nodes.json")
	if err != nil {
		t.Fatal(err)
	}
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/hosting/nodes" {
			t.Errorf("%s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write(fixture)
	})
	nodes, err := c.Hosting.Nodes(context.Background())
	if err != nil {
		t.Fatalf("Nodes: %v", err)
	}
	var arc *HostingNode
	for i := range nodes {
		if nodes[i].Number == "5NBT-C8VJ-2" {
			arc = &nodes[i]
		}
	}
	if arc == nil {
		t.Fatal("5NBT-C8VJ-2 not found")
	}
	gpu := arc.System.GPUs[0]
	// 5NBT-C8VJ-2 reports these as the string "null".
	for _, k := range []string{"mem_clock", "power_limit", "fan_speed"} {
		if v, ok := gpu[k]; !ok || v != nil {
			t.Errorf("gpu[%q] = %#v, want nil", k, v)
		}
	}
	if gpu["model"] == nil {
		t.Errorf("gpu model missing: %v", gpu)
	}
	if raw, err := c.Hosting.NodesRaw(context.Background()); err != nil || !bytes.Equal(raw, fixture) {
		t.Errorf("NodesRaw: %v", err)
	}
}

func TestHosting_SetPrices(t *testing.T) {
	var method, path string
	var body []byte
	c, _ := newTestClient(t, captureBody(t, &method, &path, &body, http.StatusNoContent, ""))
	err := c.Hosting.SetPrices(context.Background(), "7kq4 m2xd/4", NodePrices{
		Base:           Price(250_000),
		Storage:        Price(0), // zero is a valid price and must be sent
		MinRentMinutes: Int(0),
	})
	if err != nil {
		t.Fatalf("SetPrices: %v", err)
	}
	if method != http.MethodPatch || path != "/hosting/nodes/7kq4%20m2xd%2F4/prices" {
		t.Errorf("request = %s %s", method, path)
	}
	if want := `{"base":250000,"storage":0,"min_rent_minutes":0}`; string(body) != want {
		t.Errorf("body = %s, want %s", body, want)
	}

	if err := c.Hosting.SetPrices(context.Background(), "n", NodePrices{}); err == nil {
		t.Error("empty prices should be rejected before the request")
	}
	if err := c.Hosting.SetPrices(context.Background(), "", NodePrices{Base: Price(100)}); err == nil {
		t.Error("empty node number should be rejected before the request")
	}

	c400, _ := newTestClient(t, statusHandler(400, `{"message":"base must be an integer between 100 and 1000000000"}`))
	if err := c400.Hosting.SetPrices(context.Background(), "n", NodePrices{Base: Price(1)}); !IsValidation(err) {
		t.Errorf("err = %v, want ValidationError", err)
	}
	c404, _ := newTestClient(t, statusHandler(404, `{"message":"Node not found"}`))
	if err := c404.Hosting.SetPrices(context.Background(), "0000-0000-0", NodePrices{Base: Price(100)}); !IsNotFound(err) {
		t.Errorf("err = %v, want NotFoundError", err)
	}
}

func TestDecodeNullTolerant(t *testing.T) {
	var got struct {
		A *int            `json:"a"`
		B []*int          `json:"b"`
		C map[string]any  `json:"c"`
		D string          `json:"d"`
		E json.RawMessage `json:"null"` // keys are left alone
		F int64           `json:"f"`
	}
	in := `{"a":"null","b":["null",2],"c":{"x":"null","y":"keep"},"d":"Null","null":1,"f":9007199254740993}`
	if err := decodeNullTolerant([]byte(in), &got); err != nil {
		t.Fatal(err)
	}
	if got.A != nil || got.B[0] != nil || *got.B[1] != 2 || got.C["x"] != nil || got.C["y"] != "keep" || got.D != "Null" || string(got.E) != "1" {
		t.Errorf("got = %+v", got)
	}
	if got.F != 9007199254740993 { // beyond float64 precision: numbers keep their text
		t.Errorf("F = %d", got.F)
	}
	if err := decodeNullTolerant([]byte(`{`), &got); err == nil {
		t.Error("malformed JSON should be an error")
	}
}
