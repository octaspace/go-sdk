package octaspace

import (
	"encoding/json"
	"testing"
)

func TestServiceCreateResponse_UnmarshalArray(t *testing.T) {
	var response ServiceCreateResponse
	if err := json.Unmarshal([]byte(`[
		{"id":0,"uuid":"session-1","status":0},
		{"id":1,"reason":"node busy"}
	]`), &response); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if response.UUID != "session-1" {
		t.Errorf("UUID = %q, want session-1", response.UUID)
	}
	if len(response.Results) != 2 {
		t.Fatalf("len(Results) = %d, want 2", len(response.Results))
	}
	if response.Results[1].Reason != "node busy" {
		t.Errorf("Results[1].Reason = %q, want node busy", response.Results[1].Reason)
	}
	if response.Results[0].Status == nil || *response.Results[0].Status != 0 {
		t.Errorf("Results[0].Status = %v, want present 0", response.Results[0].Status)
	}
	if got := response.UUIDs(); len(got) != 1 || got[0] != "session-1" {
		t.Errorf("UUIDs() = %v, want [session-1]", got)
	}
}

func TestServiceCreateResponse_UnmarshalLegacyObject(t *testing.T) {
	var response ServiceCreateResponse
	if err := json.Unmarshal([]byte(`{"uuid":"session-1"}`), &response); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if response.UUID != "session-1" || len(response.Results) != 1 {
		t.Fatalf("response = %+v, want one successful result", response)
	}
}
