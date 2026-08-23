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
	// APICodeDCLocationDoesNotExist — "the data center location does not exist":
	// the VMware endpoints answer 400 with this code for an unknown
	// location_id instead of silently returning an empty list.
	APICodeDCLocationDoesNotExist = -8049
	// APICodeVmwareNoFreePublicNetwork — "There is no free network at the moment":
	// CreateVmwarePublicNetwork asked for a valid capacity, but the location has no
	// free public address block left. This is an infrastructure condition rather
	// than a bad request, and worth telling apart from the neighbouring -12042.
	APICodeVmwareNoFreePublicNetwork = -12043
	// APICodeVmwareInvalidPublicNetworkCapacity — "The capacity of public network
	// is invalid": the requested size is not one the location offers.
	APICodeVmwareInvalidPublicNetworkCapacity = -12042
	// APICodeVmwareOperationNotSupportedForGpuServer — "This operation is not
	// supported for GPU VMs": the operation is refused because the server has a GPU
	// allocation. Nested virtualization is mutually exclusive with GPU, so both
	// ordering a server with gpu and nested_hypervisor together and enabling nested
	// virtualization on an existing GPU server are answered with this code.
	APICodeVmwareOperationNotSupportedForGpuServer = -8149
	// APICodeVmwareServerIsSuspended — "The operation is not available for a
	// suspended VM": the server is in state suspended
	// (entities.VmwareServerStateSuspended) and must be resumed first. Unlike the
	// GPU and location refusals this one cannot be predicted from the catalog.
	APICodeVmwareServerIsSuspended = -8154
	// APICodeVmwareNestedHypervisorNotSupportedInLocation — "The location has no
	// available VDC that supports nested hypervisor": the order asked for
	// nested_hypervisor in a location where no VDC available to the caller supports
	// it. VmwareLocation.NestedHypervisorSupported reports the same capability up
	// front, so this code means the catalog was not consulted or has since changed.
	APICodeVmwareNestedHypervisorNotSupportedInLocation = -8155
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
	// ErrorParams — the parsed error_params of every error in the response
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

// ErrTaskFailed — a sentinel for "the API accepted the request and the backend
// task then failed". Such a failure carries no error code: the task object only
// reports the status. It is often transient (the same payload succeeds on a
// retry, typically when the project is busy with other operations on the same
// kind of object), so callers of idempotent operations can retry on it.
var ErrTaskFailed = errors.New("backend task failed")

// IsTaskFailed reports whether err is a failed backend task (see ErrTaskFailed).
func IsTaskFailed(err error) bool {
	return errors.Is(err, ErrTaskFailed)
}

// IsNotFound reports whether err means the requested object does not exist:
// either an HTTP 404 from the API or a semantic not-found (see ErrNotFound).
//
// This intentionally does NOT cover the "location does not exist" error
// (APICodeDCLocationDoesNotExist, -8049), which the backend returns as HTTP 400.
// Recognizing a 400 as not-found here would be a leaky hack that misclassifies
// other 400s, so that case has its own helper — IsInvalidLocation.
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
//
// The API serializes concurrent changes to one object, so a mutation issued while
// another is still running on the same server or network is rejected with this
// code. It is retried automatically (Config.RetryableCodes); this helper is for
// callers that drive their own sequencing.
func IsConflict(err error) bool {
	return HasAPICode(err, APICodeConflict)
}

// IsNetworkInUse reports whether err is the API "servers are connected to the
// network" error (-19511) — a network cannot be deleted while servers or gateways
// are still attached to it.
func IsNetworkInUse(err error) bool {
	return HasAPICode(err, APICodeNetworkInUse)
}

// IsInvalidLocation reports whether err is the API "location does not exist"
// error (-8049) — the 400 returned for an unknown location_id filter.
func IsInvalidLocation(err error) bool {
	return HasAPICode(err, APICodeDCLocationDoesNotExist)
}

// IsVmwareNoFreePublicNetwork reports whether err is the VMware "there is no free
// network at the moment" error (-12043) — the requested public network capacity is
// valid, but the location has no free address block left. Distinct from
// APICodeVmwareInvalidPublicNetworkCapacity (-12042), which means the size itself
// is not offered.
func IsVmwareNoFreePublicNetwork(err error) bool {
	return HasAPICode(err, APICodeVmwareNoFreePublicNetwork)
}

// IsVmwareOperationNotSupportedForGpuServer reports whether err is the VMware
// "this operation is not supported for GPU VMs" error (-8149) — the server has a
// GPU allocation, which rules the operation out. Nested virtualization is the
// case that hits it: it cannot be combined with a GPU, neither at order time nor
// by enabling it later.
func IsVmwareOperationNotSupportedForGpuServer(err error) bool {
	return HasAPICode(err, APICodeVmwareOperationNotSupportedForGpuServer)
}

// IsVmwareServerSuspended reports whether err is the VMware "the operation is not
// available for a suspended VM" error (-8154) — the server has to be resumed
// before the operation can be retried.
func IsVmwareServerSuspended(err error) bool {
	return HasAPICode(err, APICodeVmwareServerIsSuspended)
}

// IsVmwareNestedHypervisorNotSupportedInLocation reports whether err is the
// VMware "the location has no available VDC that supports nested hypervisor"
// error (-8155) — the order asked for nested_hypervisor where no VDC available to
// the caller offers it. Check VmwareLocation.NestedHypervisorSupported before
// ordering to avoid it.
func IsVmwareNestedHypervisorNotSupportedInLocation(err error) bool {
	return HasAPICode(err, APICodeVmwareNestedHypervisorNotSupportedInLocation)
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
		// Error_params is part of the documented error envelope
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
// Params carries the flattened error_params of every error entry, in the
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

// ValidationError reports a request that failed the SDK's own checks and was
// therefore never sent. Errors originating from the API are *RequestError instead.
type ValidationError struct {
	Message string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s", e.Message)
}

// NewValidationError builds a ValidationError with the given message.
func NewValidationError(message string) *ValidationError {
	return &ValidationError{Message: message}
}
