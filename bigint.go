package octaspace

import (
	"fmt"
	"math/big"
	"strconv"
)

// FlexInt64 unmarshals a JSON value that may be a bare number, a quoted string
// containing a decimal integer, or null. It is used for fields whose JSON type
// varies between API endpoints (e.g. Session.Duration is returned as 3600 by
// /sessions and as "3600" by /sessions?recent=true).
//
// A JSON null or empty string unmarshals to zero.
type FlexInt64 int64

// UnmarshalJSON implements [encoding/json.Unmarshaler].
func (f *FlexInt64) UnmarshalJSON(data []byte) error {
	s := string(data)
	if s == "null" || s == "" {
		return nil
	}
	// Strip quotes if the value is a JSON string.
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	if s == "" {
		return nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("octaspace: cannot parse %s as int64: %w", data, err)
	}
	*f = FlexInt64(v)
	return nil
}

// Int64 returns the underlying int64 value.
func (f FlexInt64) Int64() int64 { return int64(f) }

// BigInt is a [*big.Int] that marshals and unmarshals as a JSON number
// (bare or quoted). It is used for large integer values such as Wei-denominated
// balances that exceed the safe range of float64.
//
// A JSON null unmarshals to a zero-value BigInt (Int == nil); callers should
// check this before dereferencing.
type BigInt struct {
	*big.Int
}

// UnmarshalJSON implements [encoding/json.Unmarshaler].
func (b *BigInt) UnmarshalJSON(data []byte) error {
	s := string(data)
	if s == "null" || s == "" {
		return nil
	}
	// Accept quoted JSON strings too — some endpoints return large integers
	// as strings to avoid JavaScript's 2^53 precision limit.
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	if s == "" {
		return nil
	}
	n, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return fmt.Errorf("octaspace: invalid big integer: %s", data)
	}
	b.Int = n
	return nil
}

// MarshalJSON implements [encoding/json.Marshaler].
// A nil underlying Int marshals to the bare number 0.
func (b BigInt) MarshalJSON() ([]byte, error) {
	if b.Int == nil {
		return []byte("0"), nil
	}
	return []byte(b.String()), nil
}
