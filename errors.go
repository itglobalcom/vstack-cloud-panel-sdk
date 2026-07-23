package sdk

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// Well-known API error codes.
const (
	// APICodeConflict — "a conflict occurred during the competitive change of the
	// object" — the API serializes changes to an object (e.g. a DNS zone); the
	// code is transient and is retried by default (see Config.RetryableCodes).
	APICodeConflict = -4000
	// APICodeAlreadyExists — "Record already exists" (for example, a second
	// CNAME for the same name, or a record with identical rdata).
	APICodeAlreadyExists = -5542
	// APICodeNetworkInUse — "Servers are connected to the network": the network
	// cannot be deleted while servers or gateways are connected to it.
	APICodeNetworkInUse = -19511
	// APICodeAffinityGroupNotEmpty — "There must not be any servers in the group":
	// the group cannot be deleted while it contains servers.
	APICodeAffinityGroupNotEmpty = -19619
)

// RequestError represents an HTTP request error with detailed information
type RequestError struct {
	Status     string
	StatusCode int
	Message    string
	Body       []byte
	// Codes — API error codes from the response body ({"errors":[{"code":...}]}).
	Codes []int
	Err   error
}

// Error implements the error interface
func (e *RequestError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("HTTP %s: %s (%v)", e.Status, e.Message, e.Err)
	}
	if e.Message != "" {
		return fmt.Sprintf("HTTP %s: %s", e.Status, e.Message)
	}
	return fmt.Sprintf("HTTP %s", e.Status)
}

// Unwrap returns the underlying error
func (e *RequestError) Unwrap() error {
	return e.Err
}

// HasCode reports whether the API returned the given error code.
func (e *RequestError) HasCode(code int) bool {
	for _, c := range e.Codes {
		if c == code {
			return true
		}
	}
	return false
}

// ErrNotFound — a sentinel for a semantic "object not found" without an HTTP
// 404: some endpoints respond with 200 and an empty object, and
// objects-within-an-object (a server's volume/NIC/snapshot) are found by
// filtering a list. The SDK wraps such errors via %w so that IsNotFound
// treats them the same as a 404.
var ErrNotFound = errors.New("not found")

// IsNotFound reports whether err means the requested object does not exist:
// either an HTTP 404 from the API or a semantic not-found (see ErrNotFound).
func IsNotFound(err error) bool {
	if errors.Is(err, ErrNotFound) {
		return true
	}
	var re *RequestError
	return errors.As(err, &re) && re.StatusCode == http.StatusNotFound
}

// IsAlreadyExists reports whether err is the API "already exists" error (-5542).
func IsAlreadyExists(err error) bool {
	return HasAPICode(err, APICodeAlreadyExists)
}

// IsConflict reports whether err is the transient API conflict error (-4000).
func IsConflict(err error) bool {
	return HasAPICode(err, APICodeConflict)
}

// HasAPICode reports whether err carries the given API error code.
func HasAPICode(err error, code int) bool {
	var re *RequestError
	return errors.As(err, &re) && re.HasCode(code)
}

// apiErrorBody — the actual API error format: {"errors":[{"code":-4000,"message":"..."}]}.
type apiErrorBody struct {
	Errors []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
}

// legacyErrorBody — a flat format for non-standard responses.
type legacyErrorBody struct {
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

// parseAPIError extracts codes and a human-readable message from the error body.
func parseAPIError(body []byte) (codes []int, message string) {
	if len(body) == 0 {
		return nil, ""
	}

	var apiErr apiErrorBody
	if err := json.Unmarshal(body, &apiErr); err == nil && len(apiErr.Errors) > 0 {
		for _, e := range apiErr.Errors {
			codes = append(codes, e.Code)
			if message == "" && e.Message != "" {
				message = e.Message
			}
		}
		if message == "" {
			message = string(body)
		}
		return codes, message
	}

	var legacy legacyErrorBody
	if err := json.Unmarshal(body, &legacy); err == nil {
		switch {
		case legacy.Error != "":
			return nil, legacy.Error
		case legacy.Message != "":
			return nil, legacy.Message
		case legacy.Code != "":
			return nil, legacy.Code
		}
	}

	return nil, string(body)
}

// parseErrorResponse attempts to parse API error from response body.
// Deprecated: kept for backward compatibility; see parseAPIError.
func parseErrorResponse(body []byte) string {
	_, msg := parseAPIError(body)
	return msg
}

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s", e.Message)
}

func NewValidationError(message string) *ValidationError {
	return &ValidationError{Message: message}
}
