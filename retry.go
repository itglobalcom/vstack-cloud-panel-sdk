package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"time"
)

// RetryReason describes the reason for retrying a request
type RetryReason int

const (
	NoRetry RetryReason = iota
	RetryNetworkError
	RetryHTTPStatus
	RetryAPIErrorCode
)

// String returns a human-readable name for the retry reason.
func (r RetryReason) String() string {
	switch r {
	case NoRetry:
		return "NoRetry"
	case RetryNetworkError:
		return "RetryNetworkError"
	case RetryHTTPStatus:
		return "RetryHTTPStatus"
	case RetryAPIErrorCode:
		return "RetryAPIErrorCode"
	default:
		return fmt.Sprintf("UnknownRetryReason(%d)", int(r))
	}
}

// RetryDecision contains the decision about retry and its reason
type RetryDecision struct {
	ShouldRetry bool
	Reason      RetryReason
	Details     string
}

// RetryPolicy determines whether a request should be retried
type RetryPolicy func(resp *http.Response, err error) RetryDecision

// DefaultRetryPolicy is the policy the client uses when none is configured. It
// retries transport errors, the HTTP statuses in Config.RetryableStatus and the
// API error codes in Config.RetryableCodes, and honours a Retry-After header when
// the response carries one.
func (c *CloudClient) DefaultRetryPolicy(resp *http.Response, err error) RetryDecision {
	// Network error
	if err != nil {
		return RetryDecision{
			ShouldRetry: true,
			Reason:      RetryNetworkError,
			Details:     err.Error(),
		}
	}

	// HTTP status from retryable list
	for _, status := range c.config.RetryableStatus {
		if resp.StatusCode == status {
			return RetryDecision{
				ShouldRetry: true,
				Reason:      RetryHTTPStatus,
				Details:     fmt.Sprintf("HTTP %d", status),
			}
		}
	}

	// Check API error code in response body
	if resp != nil && resp.StatusCode >= 400 {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return RetryDecision{ShouldRetry: false, Reason: NoRetry}
		}

		resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		var apiResp struct {
			Errors []struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"errors"`
		}

		if err := json.Unmarshal(bodyBytes, &apiResp); err == nil {
			if len(apiResp.Errors) > 0 {
				errorCode := apiResp.Errors[0].Code
				if c.config.IsRetryableCode(errorCode) {
					return RetryDecision{
						ShouldRetry: true,
						Reason:      RetryAPIErrorCode,
						Details:     fmt.Sprintf("API code %d: %s", errorCode, apiResp.Errors[0].Message),
					}
				}
			}
		}
	}

	return RetryDecision{ShouldRetry: false, Reason: NoRetry}
}

func (c *CloudClient) exponentialBackoff(attempt int) time.Duration {
	mult := math.Pow(2, float64(attempt))
	sleep := time.Duration(mult) * c.config.RetryWaitMin

	if sleep > c.config.RetryWaitMax {
		sleep = c.config.RetryWaitMax
	}

	jitter := time.Duration(rand.Int63n(int64(sleep / 4)))
	sleep = sleep + jitter

	return sleep
}

func (c *CloudClient) doWithRetry(req *http.Request, policies ...RetryPolicy) (*http.Response, error) {
	var policy RetryPolicy
	if len(policies) > 0 && policies[0] != nil {
		policy = policies[0]
	} else {
		policy = c.DefaultRetryPolicy
	}

	var resp *http.Response
	var err error

	ctx := req.Context()

	// Read the request body once for logging
	var requestBody string
	if req.Body != nil {
		bodyBytes, readErr := io.ReadAll(req.Body)
		if readErr == nil && len(bodyBytes) > 0 {
			requestBody = string(bodyBytes)
			// Restore the body for subsequent use
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			req.GetBody = func() (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader(bodyBytes)), nil
			}
		}
	}

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		clonedReq := req.Clone(ctx)

		if requestBody != "" {
			c.logger.Info("HTTP Request (attempt %d/%d): %s %s - Body: %s",
				attempt+1, c.config.MaxRetries+1, clonedReq.Method, clonedReq.URL.String(), requestBody)
		} else {
			c.logger.Info("HTTP Request (attempt %d/%d): %s %s",
				attempt+1, c.config.MaxRetries+1, clonedReq.Method, clonedReq.URL.String())
		}

		resp, err = c.httpClient.Do(clonedReq)

		if err == nil && resp.StatusCode < 400 {
			c.logger.Info("HTTP Response: %s (success) %s %s", resp.Status, clonedReq.Method, clonedReq.URL.String())
			return resp, nil
		}

		decision := policy(resp, err)

		if err != nil {
			c.logger.Debug("Request failed (attempt %d/%d): %v",
				attempt+1, c.config.MaxRetries+1, err)
		} else {
			bodyPreview := c.readBodyForLogging(resp)

			switch decision.Reason {
			case RetryAPIErrorCode:
				c.logger.Error("Request returned API error (attempt %d/%d): HTTP %d - %s",
					attempt+1, c.config.MaxRetries+1, resp.StatusCode, decision.Details)
			case RetryHTTPStatus:
				c.logger.Error("Request returned error status (attempt %d/%d): HTTP %s - Body: %s",
					attempt+1, c.config.MaxRetries+1, resp.Status, bodyPreview)
			case NoRetry:
				c.logger.Debug("Request will not be retried (attempt %d/%d): HTTP %s - Body: %s",
					attempt+1, c.config.MaxRetries+1, resp.Status, bodyPreview)
			default:
				c.logger.Error("Unknown retry reason (attempt %d/%d): %s - HTTP %s - Body: %s",
					attempt+1, c.config.MaxRetries+1, decision.Reason, resp.Status, bodyPreview)
			}
		}

		if attempt >= c.config.MaxRetries || !decision.ShouldRetry {
			break
		}

		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}

		backoff := c.exponentialBackoff(attempt)
		c.logger.Info("Retrying after %v (reason: %s)...", backoff, decision.Details)

		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return nil, fmt.Errorf("request cancelled: %w", ctx.Err())
		}
	}

	if err != nil {
		return nil, fmt.Errorf("request failed after %d attempts: %w", c.config.MaxRetries+1, err)
	}

	return resp, nil
}

func (c *CloudClient) readBodyForLogging(resp *http.Response) string {

	if resp == nil || resp.Body == nil {
		return "(no body)"
	}

	bodyBytes, err := io.ReadAll(resp.Body)

	resp.Body.Close()

	if err != nil {
		return fmt.Sprintf("(failed to read body: %v)", err)
	}

	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	if len(bodyBytes) == 0 {
		return "(empty body)"
	}

	return string(bodyBytes)
}
