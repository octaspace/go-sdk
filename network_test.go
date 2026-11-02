package octaspace

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"testing"
)

func TestNetwork_Info(t *testing.T) {
	var want NetworkInfo
	decodeStrict(t, "testdata/next/network.json", &want)
	fixture, err := os.ReadFile("testdata/next/network.json")
	if err != nil {
		t.Fatal(err)
	}
	c, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(fixture) })
	got, err := c.Network.Info(context.Background())
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if got.At == 0 || got.Time().Unix() != got.At || got.Nodes.Online != 3 || got.Power.TFLOPS <= 0 || got.Render.FramesTotal == 0 {
		t.Errorf("Info = %+v", got)
	}
	raw, err := c.Network.InfoRaw(context.Background())
	if err != nil || string(raw) != string(fixture) {
		t.Errorf("InfoRaw: %v", err)
	}
}

func TestNetwork_Info_Error(t *testing.T) {
	c, _ := newTestClient(t, statusHandler(500, `{"error":"server error"}`))
	_, err := c.Network.Info(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestNetworkError_Wraps(t *testing.T) {
	// Connect to an address that refuses connections to trigger a NetworkError.
	c := NewClient("key", WithBaseURL("http://127.0.0.1:1"), WithMaxRetries(0))
	_, err := c.Network.Info(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var netErr *NetworkError
	if !errors.As(err, &netErr) {
		t.Errorf("expected *NetworkError, got %T: %v", err, err)
	}
	// The wrapped error should be the original dial error.
	var opErr *net.OpError
	if !errors.As(netErr.Err, &opErr) {
		t.Errorf("inner error should be *net.OpError, got %T", netErr.Err)
	}
}
