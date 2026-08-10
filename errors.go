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
	// SDK-6: APICodeVmwareLocationNotFound — "Location not found" for an unknown
	// VMware location_id. The backend returns this as HTTP 400 (not 404), so
	// IsNotFound does NOT recognize it (see the note on IsNotFound). Callers that
	// need to treat a missing VMware location as not-found should match this code
	// explicitly via HasAPICode until the API is fixed (backend workaround API-5).
	APICodeVmwareLocationNotFound = -8049
)

// ErrorParam is a single name/value pair from an API error's error_params block.
// The backend uses it to point at the specific field or list element that failed
// (for example {"name":"Index","value":0} or {"name":"server_ids","value":"999999"}).
// Value is decoded as-is from JSON, so it may be a string, number, bool or nil.
type ErrorParam struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

// RequestError represents an HTTP request error with detailed information
type RequestError struct {
	Status     string
	StatusCode int
	Message    string
	Body       []byte
	// Codes — API error codes from the response body ({"errors":[{"code":...}]}).
	Codes []int
	// SDK-S5: ErrorParams — the parsed error_params of every error in the response
	// body, flattened across all entries, in the order the API returned them. For
	// batch operations (for example ConnectVmwareServers with several NICs) this is
	// the only way to tell which element failed. Empty when the API sends none.
	ErrorParams []ErrorParam
	Err         error
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
//
// SDK-6: this intentionally does NOT cover the VMware "Location not found" error
// (APICodeVmwareLocationNotFound, -8049), which the backend returns as HTTP 400.
// Recognizing a 400 as not-found here would be a leaky hack that misclassifies
// other 400s, so callers must match that code explicitly (see HasAPICode).
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

// apiErrorBody — the actual API error format:
// {"errors":[{"code":-4000,"message":"...","error_params":[{"name":..,"value":..}]}]}.
type apiErrorBody struct {
	Errors []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		// SDK-S5: error_params is part of the documented error envelope
		// (vmware_error_response) and is preserved here rather than dropped.
		ErrorParams []ErrorParam `json:"error_params"`
	} `json:"errors"`
}

// legacyErrorBody — a flat format for non-standard responses.
type legacyErrorBody struct {
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

// parseAPIError extracts codes, error_params and a human-readable message from
// the error body.
//
// SDK-S5: params carries the flattened error_params of every error entry, in the
// order the API returned them, so batch callers can map a failure to its element.
func parseAPIError(body []byte) (codes []int, params []ErrorParam, message string) {
	if len(body) == 0 {
		return nil, nil, ""
	}

	var apiErr apiErrorBody
	if err := json.Unmarshal(body, &apiErr); err == nil && len(apiErr.Errors) > 0 {
		for _, e := range apiErr.Errors {
			codes = append(codes, e.Code)
			params = append(params, e.ErrorParams...)
			if message == "" && e.Message != "" {
				message = e.Message
			}
		}
		if message == "" {
			message = string(body)
		}
		return codes, params, message
	}

	var legacy legacyErrorBody
	if err := json.Unmarshal(body, &legacy); err == nil {
		switch {
		case legacy.Error != "":
			return nil, nil, legacy.Error
		case legacy.Message != "":
			return nil, nil, legacy.Message
		case legacy.Code != "":
			return nil, nil, legacy.Code
		}
	}

	return nil, nil, string(body)
}

// parseErrorResponse attempts to parse API error from response body.
// Deprecated: kept for backward compatibility; see parseAPIError.
func parseErrorResponse(body []byte) string {
	_, _, msg := parseAPIError(body)
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
