package entities

import (
	"encoding/json"
	"strings"
	"testing"
)

func intPtr(v int) *int       { return &v }
func boolPtr(v bool) *bool    { return &v }
func strPtr(v string) *string { return &v }

// validKey satisfies the backend rule: 32-128 alphanumeric characters with at
// least one upper-case letter, one lower-case letter and one digit.
const validKey = "SdkExampleVpnSharedKey1234567890"

func TestVmwareGPURequestValidate(t *testing.T) {
	// All three fields are required even though two are optional-looking pointers:
	// the backend looks up a slicing policy by the exact triple.
	cases := []struct {
		name    string
		req     VmwareGPURequest
		wantErr bool
	}{
		{"full triple", VmwareGPURequest{GPUModelID: 1, VramMB: intPtr(4096), CardCount: intPtr(1)}, false},
		{"no model", VmwareGPURequest{VramMB: intPtr(4096), CardCount: intPtr(1)}, true},
		{"no vram", VmwareGPURequest{GPUModelID: 1, CardCount: intPtr(1)}, true},
		{"no card count", VmwareGPURequest{GPUModelID: 1, VramMB: intPtr(4096)}, true},
	}
	for _, tc := range cases {
		err := tc.req.Validate()
		if (err != nil) != tc.wantErr {
			t.Errorf("%s: Validate() = %v, wantErr %v", tc.name, err, tc.wantErr)
		}
	}
}

func TestVmwareCreateServerRequestValidate(t *testing.T) {
	base := func() VmwareCreateServerRequest {
		return VmwareCreateServerRequest{
			LocationID: 5, Name: "srv", ImageID: 1010,
			CPUCount: 1, RamMB: 1024, SystemDiskSizeMB: 10240,
		}
	}
	complete := base()
	if err := complete.Validate(); err != nil {
		t.Fatalf("a complete request must validate, got %v", err)
	}

	mutate := map[string]func(*VmwareCreateServerRequest){
		"no location":  func(r *VmwareCreateServerRequest) { r.LocationID = 0 },
		"no name":      func(r *VmwareCreateServerRequest) { r.Name = "" },
		"no image":     func(r *VmwareCreateServerRequest) { r.ImageID = 0 },
		"no cpu":       func(r *VmwareCreateServerRequest) { r.CPUCount = 0 },
		"no ram":       func(r *VmwareCreateServerRequest) { r.RamMB = 0 },
		"no disk":      func(r *VmwareCreateServerRequest) { r.SystemDiskSizeMB = 0 },
		"partial gpu":  func(r *VmwareCreateServerRequest) { r.GPU = &VmwareGPURequest{GPUModelID: 1} },
		"negative cpu": func(r *VmwareCreateServerRequest) { r.CPUCount = -1 },
	}
	for name, apply := range mutate {
		r := base()
		apply(&r)
		if err := r.Validate(); err == nil {
			t.Errorf("%s: Validate() = nil, want an error", name)
		}
	}

	// A complete GPU triple must pass through.
	r := base()
	r.GPU = &VmwareGPURequest{GPUModelID: 1, VramMB: intPtr(4096), CardCount: intPtr(1)}
	if err := r.Validate(); err != nil {
		t.Errorf("complete GPU triple rejected: %v", err)
	}

	// nested_hypervisor has no local precondition: both values must pass.
	for _, want := range []bool{true, false} {
		r := base()
		r.NestedHypervisor = boolPtr(want)
		if err := r.Validate(); err != nil {
			t.Errorf("nested_hypervisor=%v rejected: %v", want, err)
		}
	}

	// GPU + nested_hypervisor is a domain rule enforced by the backend
	// (APICodeVmwareOperationNotSupportedForGpuServer); Validate must not pre-empt it.
	both := base()
	both.GPU = &VmwareGPURequest{GPUModelID: 1, VramMB: intPtr(4096), CardCount: intPtr(1)}
	both.NestedHypervisor = boolPtr(true)
	if err := both.Validate(); err != nil {
		t.Errorf("gpu + nested_hypervisor must reach the API and be refused there, got %v", err)
	}
}

func TestVmwareUpsertVPNTunnelRequestValidate(t *testing.T) {
	base := func() VmwareUpsertVPNTunnelRequest {
		return VmwareUpsertVPNTunnelRequest{
			Name:               "vpn",
			MTU:                intPtr(1500),
			EncryptionType:     VmwareEdgeVPNEncryptionAES256,
			SharedKey:          validKey,
			PeerNetwork:        "10.20.30.0/24",
			PeerEndpoint:       "203.0.113.10",
			PeerIdentificator:  "203.0.113.10",
			DiffieHellmanGroup: VmwareEdgeVPNDiffieHellmanGroup14,
		}
	}
	complete := base()
	if err := complete.Validate(); err != nil {
		t.Fatalf("a complete request must validate, got %v", err)
	}

	// mtu, encryption_type and diffie_hellman_group look optional but are required.
	mutate := map[string]func(*VmwareUpsertVPNTunnelRequest){
		"no name":       func(r *VmwareUpsertVPNTunnelRequest) { r.Name = "" },
		"no mtu":        func(r *VmwareUpsertVPNTunnelRequest) { r.MTU = nil },
		"no encryption": func(r *VmwareUpsertVPNTunnelRequest) { r.EncryptionType = "" },
		"no dh group":   func(r *VmwareUpsertVPNTunnelRequest) { r.DiffieHellmanGroup = "" },
		"no peer net":   func(r *VmwareUpsertVPNTunnelRequest) { r.PeerNetwork = "" },
		"no endpoint":   func(r *VmwareUpsertVPNTunnelRequest) { r.PeerEndpoint = "" },
		"no peer id":    func(r *VmwareUpsertVPNTunnelRequest) { r.PeerIdentificator = "" },
		"zero tunnel":   func(r *VmwareUpsertVPNTunnelRequest) { r.TunnelID = intPtr(0) },
	}
	for name, apply := range mutate {
		r := base()
		apply(&r)
		if err := r.Validate(); err == nil {
			t.Errorf("%s: Validate() = nil, want an error", name)
		}
	}
}

// The pre-shared key rules mirror the backend check, so a bad key is reported
// locally instead of after a round trip.
func TestVmwareVPNSharedKeyRules(t *testing.T) {
	cases := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"valid", validKey, false},
		{"exactly min length", "Aa1" + strings.Repeat("b", VmwareEdgeVPNSharedKeyMinLength-3), false},
		{"exactly max length", "Aa1" + strings.Repeat("b", VmwareEdgeVPNSharedKeyMaxLength-3), false},
		{"empty", "", true},
		{"too short", "Aa1" + strings.Repeat("b", VmwareEdgeVPNSharedKeyMinLength-4), true},
		{"too long", "Aa1" + strings.Repeat("b", VmwareEdgeVPNSharedKeyMaxLength-2), true},
		{"no upper", strings.Repeat("a", 30) + "12", true},
		{"no lower", strings.Repeat("A", 30) + "12", true},
		{"no digit", strings.Repeat("a", 16) + strings.Repeat("B", 16), true},
		{"non alphanumeric", "Aa1-" + strings.Repeat("b", 30), true},
		{"space", "Aa1 " + strings.Repeat("b", 30), true},
	}
	for _, tc := range cases {
		r := VmwareUpsertVPNTunnelRequest{
			Name: "vpn", MTU: intPtr(1500),
			EncryptionType: VmwareEdgeVPNEncryptionAES256, SharedKey: tc.key,
			PeerNetwork: "10.0.0.0/24", PeerEndpoint: "203.0.113.1",
			PeerIdentificator: "203.0.113.1", DiffieHellmanGroup: VmwareEdgeVPNDiffieHellmanGroup14,
		}
		err := r.Validate()
		if (err != nil) != tc.wantErr {
			t.Errorf("shared key %q (%s): Validate() = %v, wantErr %v", tc.name, tc.name, err, tc.wantErr)
		}
	}
}

// Omitting enabled / default_action leaves them unchanged on the backend, so
// Validate must not demand them.
func TestVmwareUpdateEdgeFirewallRequestValidate(t *testing.T) {
	rulesOnly := VmwareUpdateEdgeFirewallRequest{
		Rules: []VmwareUpdateEdgeFirewallRule{{
			Name:   strPtr("ssh"),
			Action: VmwareEdgeFirewallActionAllow,
		}},
	}
	if err := rulesOnly.Validate(); err != nil {
		t.Errorf("a rules-only update must validate (enabled/default_action are optional), got %v", err)
	}

	if err := (&VmwareUpdateEdgeFirewallRequest{}).Validate(); err != nil {
		t.Errorf("an empty update clears the rules and must validate, got %v", err)
	}

	full := VmwareUpdateEdgeFirewallRequest{
		Enabled:       boolPtr(true),
		DefaultAction: strPtr(VmwareEdgeFirewallActionAllow),
		Rules:         rulesOnly.Rules,
	}
	if err := full.Validate(); err != nil {
		t.Errorf("a full update must validate, got %v", err)
	}

	noAction := VmwareUpdateEdgeFirewallRequest{
		Rules: []VmwareUpdateEdgeFirewallRule{{Name: strPtr("ssh")}},
	}
	if err := noAction.Validate(); err == nil {
		t.Error("a rule without an action must be rejected")
	}

	emptyDefault := VmwareUpdateEdgeFirewallRequest{DefaultAction: strPtr("")}
	if err := emptyDefault.Validate(); err == nil {
		t.Error("an explicitly empty default_action must be rejected")
	}
}

func TestVmwareUpdateServerFirewallRequestValidate(t *testing.T) {
	complete := VmwareServerFirewallRule{
		Name:             "ssh-in",
		TrafficDirection: VmwareTrafficDirectionIncoming,
		Action:           VmwareFirewallActionAllow,
		Protocol:         "tcp",
	}
	if err := (&VmwareUpdateServerFirewallRequest{Rules: []VmwareServerFirewallRule{complete}}).Validate(); err != nil {
		t.Errorf("a complete rule must validate, got %v", err)
	}
	// An empty set clears the firewall and is allowed.
	if err := (&VmwareUpdateServerFirewallRequest{}).Validate(); err != nil {
		t.Errorf("an empty rule set must validate, got %v", err)
	}

	missing := map[string]func(*VmwareServerFirewallRule){
		"name":              func(r *VmwareServerFirewallRule) { r.Name = "" },
		"traffic_direction": func(r *VmwareServerFirewallRule) { r.TrafficDirection = "" },
		"action":            func(r *VmwareServerFirewallRule) { r.Action = "" },
		"protocol":          func(r *VmwareServerFirewallRule) { r.Protocol = "" },
	}
	for field, apply := range missing {
		rule := complete
		apply(&rule)
		req := VmwareUpdateServerFirewallRequest{Rules: []VmwareServerFirewallRule{rule}}
		err := req.Validate()
		if err == nil {
			t.Errorf("a rule without %s must be rejected", field)
			continue
		}
		if !strings.Contains(err.Error(), field) {
			t.Errorf("error for missing %s should name the field, got: %v", field, err)
		}
		if !strings.Contains(err.Error(), "rules[0]") {
			t.Errorf("error for missing %s should name the index, got: %v", field, err)
		}
	}
}

// capacity is declared as a string but parsed as a number; a non-numeric value
// otherwise comes back as a bare -2002 that names neither field nor value.
func TestVmwareCreatePublicNetworkRequestValidate(t *testing.T) {
	cases := []struct {
		capacity string
		wantErr  bool
	}{
		{"4", false},
		{"1", false},
		{"", true},
		{"/29", true},
		{"small", true},
		{"0", true},
		{"-4", true},
	}
	for _, tc := range cases {
		req := VmwareCreatePublicNetworkRequest{LocationID: 5, Name: "pub", Capacity: tc.capacity}
		if err := req.Validate(); (err != nil) != tc.wantErr {
			t.Errorf("capacity %q: Validate() = %v, wantErr %v", tc.capacity, err, tc.wantErr)
		}
	}
}

func TestVmwareEditNetworkRequestValidate(t *testing.T) {
	if err := (&VmwareEditNetworkRequest{}).Validate(); err == nil {
		t.Error("an edit that changes nothing must be rejected")
	}
	if err := (&VmwareEditNetworkRequest{Name: "n"}).Validate(); err != nil {
		t.Errorf("a name-only edit must validate, got %v", err)
	}
	if err := (&VmwareEditNetworkRequest{BandwidthMbps: intPtr(20)}).Validate(); err != nil {
		t.Errorf("a bandwidth-only edit must validate, got %v", err)
	}
	if err := (&VmwareEditNetworkRequest{BandwidthMbps: intPtr(0)}).Validate(); err == nil {
		t.Error("a zero bandwidth must be rejected")
	}
}

func TestVmwareUpdateNICRequestValidate(t *testing.T) {
	if err := (&VmwareUpdateNICRequest{NetworkID: 5}).Validate(); err != nil {
		t.Errorf("network_id alone must validate, got %v", err)
	}
	if err := (&VmwareUpdateNICRequest{}).Validate(); err == nil {
		t.Error("a missing network_id must be rejected")
	}
	if err := (&VmwareUpdateNICRequest{NetworkID: 5, BandwidthMbps: intPtr(0)}).Validate(); err == nil {
		t.Error("a zero bandwidth must be rejected when set")
	}
}

func TestVmwareVolumeRequestsValidate(t *testing.T) {
	ok := VmwareCreateVolumeRequest{Name: "vol", DiskType: "SSD", SizeMB: 10240}
	if err := ok.Validate(); err != nil {
		t.Errorf("a complete create must validate, got %v", err)
	}
	for name, req := range map[string]VmwareCreateVolumeRequest{
		"no name": {DiskType: "SSD", SizeMB: 10240},
		"no type": {Name: "vol", SizeMB: 10240},
		"no size": {Name: "vol", DiskType: "SSD"},
	} {
		if err := req.Validate(); err == nil {
			t.Errorf("%s: Validate() = nil, want an error", name)
		}
	}
	if err := (&VmwareEditVolumeRequest{SizeMB: 0}).Validate(); err == nil {
		t.Error("a zero size must be rejected")
	}
}

func TestVmwareUpsertNATRuleRequestValidate(t *testing.T) {
	ok := VmwareUpsertNATRuleRequest{
		Type: VmwareEdgeNATTypeDNAT, Protocol: "tcp",
		OriginalIP: "any", TranslatedIP: "192.168.0.10",
	}
	if err := ok.Validate(); err != nil {
		t.Errorf("a complete rule must validate, got %v", err)
	}
	for name, apply := range map[string]func(*VmwareUpsertNATRuleRequest){
		"no type":          func(r *VmwareUpsertNATRuleRequest) { r.Type = "" },
		"no protocol":      func(r *VmwareUpsertNATRuleRequest) { r.Protocol = "" },
		"no original ip":   func(r *VmwareUpsertNATRuleRequest) { r.OriginalIP = "" },
		"no translated ip": func(r *VmwareUpsertNATRuleRequest) { r.TranslatedIP = "" },
		"zero rule id":     func(r *VmwareUpsertNATRuleRequest) { r.RuleID = intPtr(0) },
	} {
		r := ok
		apply(&r)
		if err := r.Validate(); err == nil {
			t.Errorf("%s: Validate() = nil, want an error", name)
		}
	}
}

func TestVmwareTaskTerminalStates(t *testing.T) {
	cases := []struct {
		state                             string
		terminal, completed, failedResult bool
	}{
		{VmwareTaskStateNew, false, false, false},
		{VmwareTaskStateInProgress, false, false, false},
		{VmwareTaskStateCompleted, true, true, false},
		{VmwareTaskStateFailed, true, false, true},
		{VmwareTaskStateCanceled, true, false, false},
		{"something-new", false, false, false},
	}
	for _, tc := range cases {
		task := VmwareTask{State: tc.state}
		if got := task.IsTerminal(); got != tc.terminal {
			t.Errorf("state %q: IsTerminal() = %v, want %v", tc.state, got, tc.terminal)
		}
		if got := task.IsCompleted(); got != tc.completed {
			t.Errorf("state %q: IsCompleted() = %v, want %v", tc.state, got, tc.completed)
		}
		if got := task.IsFailed(); got != tc.failedResult {
			t.Errorf("state %q: IsFailed() = %v, want %v", tc.state, got, tc.failedResult)
		}
	}
}

// The JSON names are fixed by the contract: nested_hypervisor on the server and
// the order, nested_hypervisor_supported in both location models. They are
// asserted against literal contract JSON so a rename cannot pass silently.
func TestVmwareNestedHypervisorContractNames(t *testing.T) {
	// Server state: present in both list and get responses, so it is a plain
	// bool rather than a live-only pointer.
	var enabled VmwareServer
	if err := json.Unmarshal([]byte(`{"id":42,"state":"active","nested_hypervisor":true}`), &enabled); err != nil {
		t.Fatalf("unmarshal server: %v", err)
	}
	if !enabled.NestedHypervisor {
		t.Errorf("nested_hypervisor:true must decode into VmwareServer.NestedHypervisor, got %+v", enabled)
	}
	// An absent field must read as "off": the API omits null fields.
	var absent VmwareServer
	if err := json.Unmarshal([]byte(`{"id":42,"state":"active"}`), &absent); err != nil {
		t.Fatalf("unmarshal server without the field: %v", err)
	}
	if absent.NestedHypervisor {
		t.Error("an absent nested_hypervisor must decode as false")
	}
	if body, err := json.Marshal(&enabled); err != nil {
		t.Fatalf("marshal server: %v", err)
	} else if !strings.Contains(string(body), `"nested_hypervisor":true`) {
		t.Errorf("VmwareServer must serialize the field as nested_hypervisor, got %s", body)
	}

	// Order field: a pointer, so "omitted" stays distinguishable from an
	// explicit false.
	order := VmwareCreateServerRequest{
		LocationID: 5, Name: "srv", ImageID: 1010,
		CPUCount: 1, RamMB: 1024, SystemDiskSizeMB: 10240,
	}
	body, err := json.Marshal(&order)
	if err != nil {
		t.Fatalf("marshal order: %v", err)
	}
	if strings.Contains(string(body), "nested_hypervisor") {
		t.Errorf("an unset NestedHypervisor must be omitted from the order, got %s", body)
	}
	order.NestedHypervisor = boolPtr(true)
	if body, err = json.Marshal(&order); err != nil {
		t.Fatalf("marshal order: %v", err)
	} else if !strings.Contains(string(body), `"nested_hypervisor":true`) {
		t.Errorf("order must serialize the field as nested_hypervisor, got %s", body)
	}
	order.NestedHypervisor = boolPtr(false)
	if body, err = json.Marshal(&order); err != nil {
		t.Fatalf("marshal order: %v", err)
	} else if !strings.Contains(string(body), `"nested_hypervisor":false`) {
		t.Errorf("an explicit false must stay in the order body, got %s", body)
	}

	// Location capability: two models describe the same catalog
	// (GetVMwareLocations and GetVmwareLocationList) and must not diverge.
	catalogJSON := []byte(`{"id":2,"tech_title":"ds-msk","gpu_supported":false,"nested_hypervisor_supported":true}`)
	var vmwareLocation VMwareLocation
	if err := json.Unmarshal(catalogJSON, &vmwareLocation); err != nil {
		t.Fatalf("unmarshal VMwareLocation: %v", err)
	}
	if !vmwareLocation.NestedHypervisorSupported {
		t.Errorf("VMwareLocation must read nested_hypervisor_supported, got %+v", vmwareLocation)
	}
	var metainfoLocation VmwareLocation
	if err := json.Unmarshal(catalogJSON, &metainfoLocation); err != nil {
		t.Fatalf("unmarshal VmwareLocation: %v", err)
	}
	if !metainfoLocation.NestedHypervisorSupported {
		t.Errorf("VmwareLocation must read nested_hypervisor_supported, got %+v", metainfoLocation)
	}
}
