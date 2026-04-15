package octaspace

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
)

func TestNetwork_Info(t *testing.T) {
	want := map[string]any{"nodes": float64(42), "region": "eu-west"}
	c, _ := newTestClient(t, jsonHandler(want))
	got, err := c.Network.Info(context.Background())
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if got["region"] != "eu-west" {
		t.Errorf("region = %v, want eu-west", got["region"])
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

func TestNodes_DownloadIdent(t *testing.T) {
	wantBytes := []byte("identity-file-contents")
	c, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		w.Write(wantBytes)
	})
	got, err := c.Nodes.DownloadIdent(context.Background(), 7)
	if err != nil {
		t.Fatalf("DownloadIdent: %v", err)
	}
	if string(got) != string(wantBytes) {
		t.Errorf("got %q, want %q", got, wantBytes)
	}
}

func TestNodes_DownloadLogs(t *testing.T) {
	wantBytes := []byte("node log data\n")
	c, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		w.Write(wantBytes)
	})
	got, err := c.Nodes.DownloadLogs(context.Background(), 7)
	if err != nil {
		t.Fatalf("DownloadLogs: %v", err)
	}
	if string(got) != string(wantBytes) {
		t.Errorf("got %q, want %q", got, wantBytes)
	}
}
