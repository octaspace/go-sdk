package octaspace

import (
	"encoding/json"
	"testing"
)

func TestFlexString_UnmarshalJSON(t *testing.T) {
	cases := []struct {
		name string
		data string
		want string
	}{
		{"bare string", `"abc123"`, "abc123"},
		{"empty string", `""`, ""},
		{"null", `null`, ""},
		{"single-element array", `["abc123"]`, "abc123"},
		{"multi-element array", `["abc123","def456"]`, "abc123, def456"},
		{"empty array", `[]`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var f FlexString
			if err := json.Unmarshal([]byte(tc.data), &f); err != nil {
				t.Fatalf("Unmarshal(%s): %v", tc.data, err)
			}
			if f.String() != tc.want {
				t.Errorf("Unmarshal(%s) = %q, want %q", tc.data, f.String(), tc.want)
			}
		})
	}
}

func TestFlexString_UnmarshalJSON_RejectsUnsupportedShape(t *testing.T) {
	var f FlexString
	if err := json.Unmarshal([]byte(`42`), &f); err == nil {
		t.Fatal("expected error for bare number, got nil")
	}
}

func TestFlexString_MarshalJSON(t *testing.T) {
	f := FlexString("abc123, def456")
	out, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(out) != `"abc123, def456"` {
		t.Errorf("Marshal = %s, want %q", out, "abc123, def456")
	}
}
