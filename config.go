package sdk

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultTimeout         = 30 * time.Second
	DefaultPollingInterval = 5 * time.Second
	DefaultUserAgent       = "vstack-cloud-panel-go-sdk/1.0.0"
	DefaultMaxRetries      = 7
	DefaultRetryWaitMin    = 3 * time.Second
	DefaultRetryWaitMax    = 30 * time.Second
)

// Config holds the configuration for the CloudClient
type Config struct {
	APIKey          string
	BaseURL         string
	Timeout         time.Duration
	PollingInterval time.Duration
	// PollingTimeout is the maximum time an "...AndWait" / Wait* call spends polling
	// a single task before giving up.
	//
	// VMware operations run far longer than this 2m default — server create
	// and copy take ~4 min, rebuild up to ~26 min — so VMware task waiting does NOT
	// use this value as-is: WaitVmwareTask raises it to VmwareTaskWaitDefaultTimeout
	// when it is smaller. Setting PollingTimeout above that floor still wins, which is
	// worth doing for rebuild; setting it below has no effect on VMware waits (use
	// WaitVmwareTaskWithTimeout to wait for less). Base resources use this value
	// directly.
	PollingTimeout time.Duration
	UserAgent      string
	HTTPClient     *http.Client
	Logger         Logger
	LogLevel       LogLevel
	Context        context.Context

	// Retry settings
	MaxRetries      int
	RetryWaitMin    time.Duration
	RetryWaitMax    time.Duration
	RetryableStatus []int // HTTP status codes for retry
	RetryableCodes  []int // API error codes for retry
}

// Option is a functional option for configuring Config
type Option func(*Config)

// WithTimeout sets a custom timeout
func WithTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.Timeout = timeout
	}
}

// WithPollingInterval sets a custom polling interval
func WithPollingInterval(interval time.Duration) Option {
	return func(c *Config) {
		c.PollingInterval = interval
	}
}

// WithPollingTimeout sets a maximum time for polling operations.
//
// For VMware tasks this raises the wait but cannot lower it: WaitVmwareTask never
// waits less than VmwareTaskWaitDefaultTimeout. Raising it above that floor is worth
// doing for rebuild, whose duration varies widely. See Config.PollingTimeout.
func WithPollingTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.PollingTimeout = timeout
	}
}

// WithUserAgent sets a custom User-Agent
func WithUserAgent(userAgent string) Option {
	return func(c *Config) {
		c.UserAgent = userAgent
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(client *http.Client) Option {
	return func(c *Config) {
		c.HTTPClient = client
	}
}

// WithLogger sets a custom logger
func WithLogger(logger Logger) Option {
	return func(c *Config) {
		c.Logger = logger
	}
}

// WithLogLevel sets the minimum log level
func WithLogLevel(level LogLevel) Option {
	return func(c *Config) {
		c.LogLevel = level
	}
}

// WithContext sets a context for the config
func WithContext(ctx context.Context) Option {
	return func(c *Config) {
		c.Context = ctx
	}
}

// WithMaxRetries sets the maximum number of retry attempts
func WithMaxRetries(maxRetries int) Option {
	return func(c *Config) {
		c.MaxRetries = maxRetries
	}
}

// WithRetryWaitMinMax sets the minimum and maximum wait time between retries
func WithRetryWaitMinMax(min, max time.Duration) Option {
	return func(c *Config) {
		c.RetryWaitMin = min
		c.RetryWaitMax = max
	}
}

// WithRetryableStatus sets custom HTTP status codes that trigger retry
func WithRetryableStatus(codes []int) Option {
	return func(c *Config) {
		c.RetryableStatus = codes
	}
}

// WithRetryableCodes sets custom API error codes that trigger retry
func WithRetryableCodes(codes []int) Option {
	return func(c *Config) {
		c.RetryableCodes = codes
	}
}

// NewConfig creates a new Config with the provided API key and base URL
func NewConfig(apiKey, baseURL string, opts ...Option) (*Config, error) {
	// Set defaults
	cfg := &Config{
		APIKey:          apiKey,
		BaseURL:         baseURL,
		Timeout:         DefaultTimeout,
		PollingInterval: DefaultPollingInterval,
		// Deliberately kept at 2m: it is a cross-cutting default, and base resources
		// settle well inside it. VMware tasks run far longer, so they do not use this
		// value as-is — WaitVmwareTask raises it to VmwareTaskWaitDefaultTimeout.
		PollingTimeout:  2 * time.Minute,
		UserAgent:       DefaultUserAgent,
		Logger:          NewNopLogger(),
		LogLevel:        Info,
		Context:         context.Background(),
		MaxRetries:      DefaultMaxRetries,
		RetryWaitMin:    DefaultRetryWaitMin,
		RetryWaitMax:    DefaultRetryWaitMax,
		RetryableStatus: []int{408, 429, 500, 502, 503, 504},
		// APICodeConflict (-4000): the API serializes changes to an object (e.g. a
		// DNS zone) — a concurrent operation gets a transient conflict and should
		// be retried.
		RetryableCodes: []int{APICodeConflict, -5592, -19059, -19511, -19605, -19803},
	}

	// Apply functional options
	for _, opt := range opts {
		opt(cfg)
	}

	// Normalize configuration
	cfg.normalize()

	// Validate configuration
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// normalize applies default values and normalizes configuration fields
func (c *Config) normalize() {
	c.BaseURL = strings.TrimRight(c.BaseURL, "/")

	if c.Timeout <= 0 {
		c.Timeout = DefaultTimeout
	}

	if c.PollingInterval <= 0 {
		c.PollingInterval = DefaultPollingInterval
	}

	if c.PollingTimeout <= 0 {
		c.PollingTimeout = 5 * time.Minute
	}

	if c.UserAgent == "" {
		c.UserAgent = DefaultUserAgent
	}

	if c.Logger == nil {
		c.Logger = NewNopLogger()
	}

	if c.LogLevel < Debug || c.LogLevel > Error {
		c.LogLevel = Info
	}

	if c.Context == nil {
		c.Context = context.Background()
	}

	if c.MaxRetries < 0 {
		c.MaxRetries = 0
	}

	if c.RetryWaitMin <= 0 {
		c.RetryWaitMin = DefaultRetryWaitMin
	}

	if c.RetryWaitMax <= 0 {
		c.RetryWaitMax = DefaultRetryWaitMax
	}

	if c.RetryableStatus == nil {
		c.RetryableStatus = []int{408, 429, 500, 502, 503, 504}
	}

	if c.RetryableCodes == nil {
		c.RetryableCodes = []int{APICodeConflict, -5592, -19059, -19511, -19605, -19803}
	}
}

// validate checks if the configuration is valid
func (c *Config) validate() error {
	if c.APIKey == "" {
		return NewValidationError("API key is required")
	}

	if c.BaseURL == "" {
		return NewValidationError("base URL is required")
	}

	if c.RetryWaitMin > c.RetryWaitMax {
		return NewValidationError("retry wait min cannot be greater than retry wait max")
	}

	return nil
}

// IsRetryableCode checks if the given API error code should trigger a retry
func (c *Config) IsRetryableCode(code int) bool {
	for _, retryableCode := range c.RetryableCodes {
		if retryableCode == code {
			return true
		}
	}
	return false
}

// String returns a string representation of the Config
func (c *Config) String() string {
	return fmt.Sprintf("Config{BaseURL: %s, Timeout: %s, PollingInterval: %s, MaxRetries: %d, UserAgent: %s, LogLevel: %v, RetryableCodes: %v, APIKey: [REDACTED]}",
		c.BaseURL, c.Timeout, c.PollingInterval, c.MaxRetries, c.UserAgent, c.LogLevel, c.RetryableCodes)
}

// Copy creates a deep copy of the Config
func (c *Config) Copy() *Config {
	retryableStatus := make([]int, len(c.RetryableStatus))
	copy(retryableStatus, c.RetryableStatus)

	retryableCodes := make([]int, len(c.RetryableCodes))
	copy(retryableCodes, c.RetryableCodes)

	return &Config{
		APIKey:          c.APIKey,
		BaseURL:         c.BaseURL,
		Timeout:         c.Timeout,
		PollingInterval: c.PollingInterval,
		PollingTimeout:  c.PollingTimeout,
		UserAgent:       c.UserAgent,
		HTTPClient:      c.HTTPClient,
		Logger:          c.Logger,
		LogLevel:        c.LogLevel,
		Context:         c.Context,
		MaxRetries:      c.MaxRetries,
		RetryWaitMin:    c.RetryWaitMin,
		RetryWaitMax:    c.RetryWaitMax,
		RetryableStatus: retryableStatus,
		RetryableCodes:  retryableCodes,
	}
}
