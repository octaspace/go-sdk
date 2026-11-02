package octaspace

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// The next API sometimes sends a missing value as the string "null" instead
// of JSON null: in GPU statistics (price percentiles, TFLOPS) and in what
// nodes report about their hardware. A strictly typed decode then fails on
// the whole response. Endpoints known to do this are read through
// getNullTolerant, which turns every "null" string into null first.
//
// It is not applied where a "null" string can be real data, such as the
// environment variables of a session.

// getNullTolerant performs a GET like get, but decodes the response after
// replacing "null" string values with JSON null.
func (c *Client) getNullTolerant(ctx context.Context, path string, params url.Values, dst any) error {
	body, err := c.getRawParams(ctx, path, params)
	if err != nil {
		return err
	}
	return decodeNullTolerant(body, dst)
}

// decodeNullTolerant decodes data into dst, treating "null" string values
// (not object keys) as JSON null. Numbers keep their exact text.
func decodeNullTolerant(data []byte, dst any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return fmt.Errorf("octaspace: decode response: %w", err)
	}
	normalized, err := json.Marshal(nullStrings(v))
	if err != nil {
		return fmt.Errorf("octaspace: decode response: %w", err)
	}
	if err := json.Unmarshal(normalized, dst); err != nil {
		return fmt.Errorf("octaspace: decode response: %w", err)
	}
	return nil
}

// nullStrings replaces the string "null" with nil throughout v.
func nullStrings(v any) any {
	switch t := v.(type) {
	case string:
		if t == "null" {
			return nil
		}
	case map[string]any:
		for k, e := range t {
			t[k] = nullStrings(e)
		}
	case []any:
		for i, e := range t {
			t[i] = nullStrings(e)
		}
	}
	return v
}
