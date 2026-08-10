package sdk

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

func TestNormalizeDomainName(t *testing.T) {
	cases := map[string]string{
		"example.com":      "example.com.",
		"example.com.":     "example.com.",
		"WWW.Example.COM":  "www.example.com.", // the API stores names in lowercase
		"WWW.Example.COM.": "www.example.com.",
		"":                 "",
	}
	for in, want := range cases {
		if got := normalizeDomainName(in); got != want {
			t.Errorf("normalizeDomainName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSplitSRVName(t *testing.T) {
	svc, proto, base, ok := splitSRVName("_sip._tcp.example.com.")
	if !ok || svc != "sip" || proto != "TCP" || base != "example.com." {
		t.Errorf("splitSRVName full: got (%q,%q,%q,%v)", svc, proto, base, ok)
	}
	if _, _, _, ok := splitSRVName("www.example.com."); ok {
		t.Error("splitSRVName must reject a non-SRV-shaped name")
	}
}

// The server adds the _service._proto. prefix to the supplied name (both on
// POST and PUT) — the SDK must send the base name, otherwise the name gets doubled.
func TestNormalizeSRVNameFields(t *testing.T) {
	t.Run("full name derives service/protocol", func(t *testing.T) {
		var service string
		var protocol *entities.Protocol
		base := normalizeSRVNameFields("_sip._tcp.example.com.", &service, &protocol)
		if base != "example.com." || service != "sip" || protocol == nil || *protocol != entities.Protocol("TCP") {
			t.Errorf("got base=%q service=%q protocol=%v", base, service, protocol)
		}
	})

	t.Run("explicit service/protocol preserved", func(t *testing.T) {
		service := "voip"
		p := entities.SRVProtocolUDP
		protocol := &p
		base := normalizeSRVNameFields("_sip._tcp.example.com.", &service, &protocol)
		if base != "example.com." || service != "voip" || *protocol != entities.SRVProtocolUDP {
			t.Errorf("explicit values must win: base=%q service=%q protocol=%v", base, service, *protocol)
		}
	})

	t.Run("base name passes through (backward compatible)", func(t *testing.T) {
		service := "sip"
		p := entities.SRVProtocolTCP
		protocol := &p
		base := normalizeSRVNameFields("example.com.", &service, &protocol)
		if base != "example.com." {
			t.Errorf("base name must pass through, got %q", base)
		}
	})
}

func TestUnquoteTXT(t *testing.T) {
	cases := map[string]string{
		`"v=spf1 -all"`: "v=spf1 -all",
		"v=spf1 -all":   "v=spf1 -all",
		`""`:            "",
	}
	for in, want := range cases {
		if got := unquoteTXT(in); got != want {
			t.Errorf("unquoteTXT(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseAPIError(t *testing.T) {
	codes, params, msg := parseAPIError([]byte(`{"errors":[{"code":-4000,"message":"conflict"}]}`))
	if len(codes) != 1 || codes[0] != APICodeConflict || msg != "conflict" || len(params) != 0 {
		t.Errorf("real format: codes=%v params=%v msg=%q", codes, params, msg)
	}

	// SDK-5: error_params must be parsed and flattened.
	codes, params, msg = parseAPIError([]byte(`{"errors":[{"code":-12030,"message":"not found","error_params":[{"name":"server_ids","value":"999999"}]}]}`))
	if len(codes) != 1 || codes[0] != -12030 || msg != "not found" ||
		len(params) != 1 || params[0].Name != "server_ids" || params[0].Value != "999999" {
		t.Errorf("error_params: codes=%v params=%v msg=%q", codes, params, msg)
	}

	codes, params, msg = parseAPIError([]byte(`{"message":"legacy"}`))
	if codes != nil || msg != "legacy" || params != nil {
		t.Errorf("legacy format: codes=%v params=%v msg=%q", codes, params, msg)
	}

	if _, _, msg := parseAPIError([]byte(`not-json`)); msg != "not-json" {
		t.Errorf("raw fallback: %q", msg)
	}
}

func TestTypedErrorHelpers(t *testing.T) {
	notFound := fmt.Errorf("failed to get domain x: %w", &RequestError{
		Status: "404 Not Found", StatusCode: http.StatusNotFound,
	})
	if !IsNotFound(notFound) {
		t.Error("IsNotFound must see a wrapped 404 RequestError")
	}

	exists := fmt.Errorf("wrap: %w", &RequestError{
		Status: "400 Bad Request", StatusCode: http.StatusBadRequest, Codes: []int{APICodeAlreadyExists},
	})
	if !IsAlreadyExists(exists) || IsConflict(exists) || IsNotFound(exists) {
		t.Error("IsAlreadyExists must match -5542 (and nothing else)")
	}

	conflict := &RequestError{Status: "400 Bad Request", StatusCode: http.StatusBadRequest, Codes: []int{APICodeConflict}}
	if !IsConflict(conflict) {
		t.Error("IsConflict must match -4000")
	}

	if IsNotFound(fmt.Errorf("plain error")) || IsAlreadyExists(nil) {
		t.Error("helpers must be false for plain/nil errors")
	}
}
