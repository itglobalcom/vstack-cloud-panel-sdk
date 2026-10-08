package entities

import "testing"

// The catalog sends the values of a closed set as PascalCase strings, and the
// spellings are fixed by the contract, not by the SDK.
func TestApplicationCatalogClosedSetsOnTheWire(t *testing.T) {
	fixed := map[string]struct {
		got  string
		want string
	}{
		"ApplicationCredentialsModeServicePassword":      {string(ApplicationCredentialsModeServicePassword), "ServicePassword"},
		"ApplicationCredentialsModeNoPasswordInTemplate": {string(ApplicationCredentialsModeNoPasswordInTemplate), "NoPasswordInTemplate"},
		"ApplicationLLMKindKey":                          {string(ApplicationLLMKindKey), "Key"},
		"ApplicationLLMKindEndpoint":                     {string(ApplicationLLMKindEndpoint), "Endpoint"},
	}
	for name, c := range fixed {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", name, c.got, c.want)
		}
	}
}
