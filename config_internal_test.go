package sdk

import (
	"testing"
	"time"
)

// The wait a base task gets by default is the platform deadline for the
// synchronous install of one-click applications — 15 minutes. The number is not
// a tuning knob of the SDK, so it is asserted as a duration and not against the
// constant itself.
func TestDefaultPollingTimeoutIsTheInstallDeadline(t *testing.T) {
	if DefaultPollingTimeout != 15*time.Minute {
		t.Errorf("DefaultPollingTimeout = %s, want %s", DefaultPollingTimeout, 15*time.Minute)
	}
}

// The default used to be two different numbers — one in NewConfig, another in
// normalize for a non-positive value — so the same absent setting produced two
// different waits. Every road to the default must now end on the same value.
func TestPollingTimeoutDefault(t *testing.T) {
	cases := map[string]struct {
		opts []Option
		want time.Duration
	}{
		"not set at all":               {nil, DefaultPollingTimeout},
		"set to zero":                  {[]Option{WithPollingTimeout(0)}, DefaultPollingTimeout},
		"set to a negative duration":   {[]Option{WithPollingTimeout(-time.Second)}, DefaultPollingTimeout},
		"set below the default":        {[]Option{WithPollingTimeout(90 * time.Second)}, 90 * time.Second},
		"set above the default":        {[]Option{WithPollingTimeout(30 * time.Minute)}, 30 * time.Minute},
		"set to the default itself":    {[]Option{WithPollingTimeout(DefaultPollingTimeout)}, DefaultPollingTimeout},
		"another option set instead":   {[]Option{WithMaxRetries(0)}, DefaultPollingTimeout},
		"polling interval set instead": {[]Option{WithPollingInterval(time.Second)}, DefaultPollingTimeout},
	}

	for name, tc := range cases {
		cfg, err := NewConfig("test-key", "https://example.test", tc.opts...)
		if err != nil {
			t.Fatalf("%s: NewConfig: %v", name, err)
		}
		if cfg.PollingTimeout != tc.want {
			t.Errorf("%s: PollingTimeout = %s, want %s", name, cfg.PollingTimeout, tc.want)
		}
	}
}

// normalize is also reached by a Config assembled by hand, and it must not carry
// a default of its own: a value it fills in has to be the one NewConfig sets.
func TestNormalizePollingTimeout(t *testing.T) {
	cases := map[string]struct {
		set  time.Duration
		want time.Duration
	}{
		"zero becomes the default":     {0, DefaultPollingTimeout},
		"negative becomes the default": {-time.Second, DefaultPollingTimeout},
		"a set value is kept":          {90 * time.Second, 90 * time.Second},
	}

	for name, tc := range cases {
		cfg := &Config{PollingTimeout: tc.set}
		cfg.normalize()
		if cfg.PollingTimeout != tc.want {
			t.Errorf("%s: PollingTimeout = %s, want %s", name, cfg.PollingTimeout, tc.want)
		}
	}
}
