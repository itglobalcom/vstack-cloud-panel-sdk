package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// CloudClient represents the main client for interacting with the cloud API.
// Clients are safe for concurrent use by multiple goroutines.
type CloudClient struct {
	httpClient *http.Client
	config     *Config
	logger     *LeveledLogger

	// baseURL is cached for performance
	baseURL string
}

// NewClient creates a new CloudClient instance with the provided configuration.
// The client is safe for concurrent use and should be reused rather than created per-request.
func NewClient(config *Config) (*CloudClient, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Config is already validated in NewConfig, but double-check for safety
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Use provided HTTP client or create a new one with custom transport
	httpClient := config.HTTPClient
	if httpClient == nil {
		transport := &headerTransport{
			base: http.DefaultTransport,
			headers: map[string]string{
				"X-API-KEY":  config.APIKey,
				"User-Agent": config.UserAgent,
			},
		}

		httpClient = &http.Client{
			Timeout:   config.Timeout,
			Transport: transport,
		}
	} else {
		// Wrap existing client's transport to add headers
		existingTransport := httpClient.Transport
		if existingTransport == nil {
			existingTransport = http.DefaultTransport
		}

		httpClient.Transport = &headerTransport{
			base: existingTransport,
			headers: map[string]string{
				"X-API-KEY":  config.APIKey,
				"User-Agent": config.UserAgent,
			},
		}
	}

	// Cache base URL
	baseURL := strings.TrimRight(config.BaseURL, "/")

	return &CloudClient{
		httpClient: httpClient,
		config:     config,
		logger:     NewLeveledLogger(config.Logger, config.LogLevel),
		baseURL:    baseURL,
	}, nil
}

// headerTransport is a custom RoundTripper that adds headers to all requests.
// It's safe for concurrent use.
type headerTransport struct {
	base    http.RoundTripper
	headers map[string]string
	mu      sync.RWMutex // Protect headers map for concurrent access
}

// RoundTrip implements http.RoundTripper interface
func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clonedReq := req.Clone(req.Context())

	// Add custom headers
	t.mu.RLock()
	for key, value := range t.headers {
		// Only set if not already present to allow per-request overrides
		if clonedReq.Header.Get(key) == "" {
			clonedReq.Header.Set(key, value)
		}
	}
	t.mu.RUnlock()

	return t.base.RoundTrip(clonedReq)
}

// Config returns a copy of the client configuration.
// Note: This returns a shallow copy. Modifying nested objects (like HTTPClient)
// will affect the original config.
func (c *CloudClient) Config() Config {
	return *c.config
}

// buildURL constructs the full API URL with base URL and path.
// This method is safe for concurrent use.
func (c *CloudClient) buildURL(path string) string {
	path = strings.TrimPrefix(path, "/")
	return fmt.Sprintf("%s/api/v1/%s", c.baseURL, path)
}

// newRequest creates a new HTTP request with the given method, path, and body.
// It automatically adds the context and builds the full URL.
func (c *CloudClient) newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	url := c.buildURL(path)

	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}

		// Log request body in debug mode
		if c.logger != nil {
			c.logger.Debug(fmt.Sprintf("Request: %s %s\nBody: %s", method, url, string(bodyBytes)))
		}

		bodyReader = bytes.NewReader(bodyBytes)
	} else {
		// Log request without body
		if c.logger != nil {
			c.logger.Debug(fmt.Sprintf("Request: %s %s", method, url))
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set content type for requests with body
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

// // do executes the HTTP request and returns the raw response.
// // The caller is responsible for closing the response body.
// func (c *CloudClient) do(req *http.Request) (*http.Response, error) {
// 	c.logger.Info("HTTP Request: %s %s", req.Method, req.URL.String())

// 	resp, err := c.httpClient.Do(req)
// 	if err != nil {
// 		c.logger.Error("Request failed: %v", err)
// 		return nil, fmt.Errorf("request failed: %w", err)
// 	}

// 	c.logger.Info("HTTP Response: %s", resp.Status)

// 	return resp, nil
// }

// doJSON executes the HTTP request, handles errors, and unmarshals JSON response.
// This method automatically closes the response body.
func (c *CloudClient) doJSON(req *http.Request, result any) error {
	resp, err := c.doWithRetry(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("Failed to read response body: %v", err)
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode >= 400 {
		codes, message := parseAPIError(body)
		return &RequestError{
			Status:     resp.Status,
			StatusCode: resp.StatusCode,
			Message:    message,
			Body:       body,
			Codes:      codes,
		}
	}

	// If no result expected, return early
	if result == nil {
		return nil
	}

	// Unmarshal JSON response
	if len(body) > 0 {
		if err := json.Unmarshal(body, result); err != nil {
			c.logger.Error("Failed to unmarshal response: %v", err)
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}
