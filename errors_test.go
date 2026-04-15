package octaspace

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestExtractMessage_JSONErrorKey(t *testing.T) {
	msg := extractMessage([]byte(`{"error":"something went wrong"}`))
	if msg != "something went wrong" {
		t.Errorf("got %q, want %q", msg, "something went wrong")
	}
}

func TestExtractMessage_JSONMessageKey(t *testing.T) {
	msg := extractMessage([]byte(`{"message":"bad input"}`))
	if msg != "bad input" {
		t.Errorf("got %q, want %q", msg, "bad input")
	}
}

func TestExtractMessage_JSONDetailKey(t *testing.T) {
	msg := extractMessage([]byte(`{"detail":"validation failed"}`))
	if msg != "validation failed" {
		t.Errorf("got %q, want %q", msg, "validation failed")
	}
}

func TestExtractMessage_FallbackToRaw(t *testing.T) {
	msg := extractMessage([]byte("plain text error"))
	if msg != "plain text error" {
		t.Errorf("got %q, want %q", msg, "plain text error")
	}
}

func TestAPIError_Error(t *testing.T) {
	e := &APIError{StatusCode: 404, Message: "not found"}
	want := "octaspace: API error 404: not found"
	if e.Error() != want {
		t.Errorf("got %q, want %q", e.Error(), want)
	}
}

func TestAPIError_Error_NoMessage(t *testing.T) {
	e := &APIError{StatusCode: 500}
	if !strings.Contains(e.Error(), "500") {
		t.Errorf("error %q should contain status code", e.Error())
	}
}

func TestNetworkError_Unwrap(t *testing.T) {
	inner := errors.New("dial tcp: connection refused")
	netErr := &NetworkError{Err: inner}
	if !errors.Is(netErr, inner) {
		t.Error("errors.Is should find the wrapped error")
	}
}

func TestErrorFromResponse_Types(t *testing.T) {
	tests := []struct {
		code     int
		wantType string
	}{
		{401, "*octaspace.AuthenticationError"},
		{403, "*octaspace.PermissionError"},
		{404, "*octaspace.NotFoundError"},
		{422, "*octaspace.ValidationError"},
		{429, "*octaspace.RateLimitError"},
		{500, "*octaspace.ServerError"},
		{503, "*octaspace.ServerError"},
		{400, "*octaspace.APIError"},
	}
	for _, tc := range tests {
		t.Run(http.StatusText(tc.code), func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tc.code,
				Header:     http.Header{},
			}
			err := errorFromResponse(resp, []byte(`{}`))
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Errorf("errors.As(*APIError) failed for %T", err)
			}
			if apiErr.StatusCode != tc.code {
				t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, tc.code)
			}
		})
	}
}

func TestErrorFromResponse_RetryAfter(t *testing.T) {
	resp := &http.Response{
		StatusCode: 429,
		Header:     http.Header{"Retry-After": []string{"30"}},
	}
	err := errorFromResponse(resp, []byte(`{}`))
	var apiErr *APIError
	errors.As(err, &apiErr)
	if apiErr.RetryAfter != 30 {
		t.Errorf("RetryAfter = %d, want 30", apiErr.RetryAfter)
	}
}

func TestErrorFromResponse_RequestID(t *testing.T) {
	resp := &http.Response{
		StatusCode: 500,
		Header:     http.Header{"X-Request-Id": []string{"req-xyz"}},
	}
	err := errorFromResponse(resp, []byte(`{}`))
	var apiErr *APIError
	errors.As(err, &apiErr)
	if apiErr.RequestID != "req-xyz" {
		t.Errorf("RequestID = %q, want req-xyz", apiErr.RequestID)
	}
}

func TestIsHelpers(t *testing.T) {
	t.Run("IsNotFound", func(t *testing.T) {
		resp := &http.Response{StatusCode: 404, Header: http.Header{}}
		err := errorFromResponse(resp, []byte(`{}`))
		if !IsNotFound(err) {
			t.Error("IsNotFound should return true")
		}
		if IsRateLimit(err) || IsAuthError(err) {
			t.Error("other helpers should return false")
		}
	})
	t.Run("IsRateLimit", func(t *testing.T) {
		resp := &http.Response{StatusCode: 429, Header: http.Header{}}
		err := errorFromResponse(resp, []byte(`{}`))
		if !IsRateLimit(err) {
			t.Error("IsRateLimit should return true")
		}
	})
	t.Run("IsAuthError", func(t *testing.T) {
		resp := &http.Response{StatusCode: 401, Header: http.Header{}}
		err := errorFromResponse(resp, []byte(`{}`))
		if !IsAuthError(err) {
			t.Error("IsAuthError should return true")
		}
	})
}
