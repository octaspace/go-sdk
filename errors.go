package octaspace

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// NetworkError wraps low-level transport failures such as DNS resolution errors,
// TLS handshake failures, or connection refused. It is never returned for HTTP
// 4xx/5xx responses — those produce an [APIError] or a subtype.
type NetworkError struct {
	Err error
}

func (e *NetworkError) Error() string { return fmt.Sprintf("octaspace: network error: %v", e.Err) }
func (e *NetworkError) Unwrap() error { return e.Err }

// APIError represents an error response received from the OctaSpace API
// (HTTP 4xx or 5xx). Callers should prefer [errors.As] to unwrap a specific
// subtype such as [*NotFoundError] rather than comparing StatusCode directly.
type APIError struct {
	StatusCode int    // HTTP status code from the response
	RequestID  string // value of the X-Request-Id response header, if present
	RetryAfter int    // seconds parsed from the Retry-After header; 0 if absent
	Message    string // human-readable message extracted from the response body
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("octaspace: API error %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("octaspace: API error %d", e.StatusCode)
}

// AuthenticationError is returned for HTTP 401 Unauthorized responses.
type AuthenticationError struct{ *APIError }

func (e *AuthenticationError) Unwrap() error { return e.APIError }

// PermissionError is returned for HTTP 403 Forbidden responses.
type PermissionError struct{ *APIError }

func (e *PermissionError) Unwrap() error { return e.APIError }

// NotFoundError is returned for HTTP 404 Not Found responses.
type NotFoundError struct{ *APIError }

func (e *NotFoundError) Unwrap() error { return e.APIError }

// ValidationError is returned for HTTP 422 Unprocessable Entity responses.
type ValidationError struct{ *APIError }

func (e *ValidationError) Unwrap() error { return e.APIError }

// RateLimitError is returned for HTTP 429 Too Many Requests responses.
// Check [APIError.RetryAfter] for the recommended wait duration in seconds.
type RateLimitError struct{ *APIError }

func (e *RateLimitError) Unwrap() error { return e.APIError }

// ServerError is returned for HTTP 5xx responses not covered by a more specific
// subtype (502, 503, 504 are still wrapped as ServerError).
type ServerError struct{ *APIError }

func (e *ServerError) Unwrap() error { return e.APIError }

// IsNotFound reports whether err (or any error in its chain) is a [*NotFoundError].
func IsNotFound(err error) bool {
	var e *NotFoundError
	return errors.As(err, &e)
}

// IsRateLimit reports whether err (or any error in its chain) is a [*RateLimitError].
func IsRateLimit(err error) bool {
	var e *RateLimitError
	return errors.As(err, &e)
}

// IsAuthError reports whether err (or any error in its chain) is an [*AuthenticationError].
func IsAuthError(err error) bool {
	var e *AuthenticationError
	return errors.As(err, &e)
}

// errorFromResponse builds the appropriate typed error for a non-2xx HTTP response.
func errorFromResponse(resp *http.Response, body []byte) error {
	base := &APIError{
		StatusCode: resp.StatusCode,
		RequestID:  resp.Header.Get("X-Request-Id"),
		Message:    extractMessage(body),
	}
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		if n, err := strconv.Atoi(ra); err == nil {
			base.RetryAfter = n
		}
	}
	switch resp.StatusCode {
	case 401:
		return &AuthenticationError{base}
	case 403:
		return &PermissionError{base}
	case 404:
		return &NotFoundError{base}
	case 422:
		return &ValidationError{base}
	case 429:
		return &RateLimitError{base}
	default:
		if resp.StatusCode >= 500 {
			return &ServerError{base}
		}
		return base
	}
}

// maxErrorMessageLen is the maximum number of bytes retained from an error
// response body. Anything longer is truncated to keep error strings sane
// (e.g. when the server returns a full HTML page).
const maxErrorMessageLen = 512

// extractMessage attempts to pull a human-readable message from an error response body.
// It checks common keys ("error", "message", "detail") before falling back to the raw body.
// The result is always truncated to [maxErrorMessageLen] bytes.
func extractMessage(body []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return truncate(strings.TrimSpace(string(body)))
	}
	for _, key := range []string{"error", "message", "detail"} {
		if v, ok := payload[key]; ok {
			if s, ok := v.(string); ok && s != "" {
				return truncate(s)
			}
		}
	}
	return truncate(strings.TrimSpace(string(body)))
}

func truncate(s string) string {
	if len(s) <= maxErrorMessageLen {
		return s
	}
	return s[:maxErrorMessageLen] + "…"
}
