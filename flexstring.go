package octaspace

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FlexString unmarshals a JSON value that may be a bare string, a JSON array
// of strings, or null. Multiple values from an array are joined with ", ".
//
// It is used for fields whose JSON shape varies by session kind — e.g.
// Session.ContainerID and SessionInfo.ContainerID are absent for VPN
// sessions but a JSON array of container IDs for compute (MR) sessions,
// which previously broke decoding of the whole /sessions list.
type FlexString string

// UnmarshalJSON implements [encoding/json.Unmarshaler].
func (f *FlexString) UnmarshalJSON(data []byte) error {
	s := string(data)
	if s == "null" || s == "" {
		return nil
	}
	switch s[0] {
	case '"':
		var v string
		if err := json.Unmarshal(data, &v); err != nil {
			return fmt.Errorf("octaspace: cannot parse %s as string: %w", data, err)
		}
		*f = FlexString(v)
	case '[':
		var v []string
		if err := json.Unmarshal(data, &v); err != nil {
			return fmt.Errorf("octaspace: cannot parse %s as []string: %w", data, err)
		}
		*f = FlexString(strings.Join(v, ", "))
	default:
		return fmt.Errorf("octaspace: unsupported JSON shape for string field: %s", data)
	}
	return nil
}

// MarshalJSON implements [encoding/json.Marshaler]. FlexString always
// marshals back out as a plain JSON string, regardless of the shape it was
// decoded from.
func (f FlexString) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(f))
}

// String returns the underlying value.
func (f FlexString) String() string { return string(f) }
