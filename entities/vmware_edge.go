package entities

import "fmt"

// ===================== Edge firewall =====================

// Edge firewall action / default-action values.
//
// The API normalizes enum values to lower case on read and accepts them
// case-insensitively on write, so writing these constants round-trips exactly.
const (
	// VmwareEdgeFirewallActionAllow lets matching traffic through.
	VmwareEdgeFirewallActionAllow = "allow"
	// VmwareEdgeFirewallActionDeny drops matching traffic.
	VmwareEdgeFirewallActionDeny = "deny"
)

// VmwareEdgeFirewallProtocolAny matches every protocol in an edge firewall rule.
// Alongside it the usual named protocols ("tcp", "udp", "icmp") are accepted; the
// API contract is the authority on the full set.
const VmwareEdgeFirewallProtocolAny = "any"

// VmwareEdgeFirewallAny is the wildcard accepted by the source/destination and
// port fields of an edge firewall rule.
const VmwareEdgeFirewallAny = "any"

// VmwareEdgeFirewallRule represents a single edge firewall rule as returned by
// the API.
//
// Flat and symmetric with the write model — the backend expands the vCloud
// applications[] into one flat rule per protocol+ports, so a read rule maps
// one-to-one onto a write rule. The vCloud-only id and the nested applications
// are gone.
//
// Enabled and Description are read-only in practice: the backend derives
// Description from the rule name and forces Enabled to true, ignoring whatever
// the update request carried. That is why
// VmwareUpdateEdgeFirewallRule deliberately has no such fields — offering them
// would promise a round trip the API does not perform.
type VmwareEdgeFirewallRule struct {
	Enabled         *bool   `json:"enabled,omitempty"`
	Name            *string `json:"name,omitempty"`
	Description     *string `json:"description,omitempty"`
	Action          *string `json:"action,omitempty"`
	Protocol        *string `json:"protocol,omitempty"`
	Source          *string `json:"source,omitempty"`
	SourcePort      *string `json:"source_port,omitempty"`
	Destination     *string `json:"destination,omitempty"`
	DestinationPort *string `json:"destination_port,omitempty"`
}

// VmwareEdgeFirewall represents the edge firewall configuration.
type VmwareEdgeFirewall struct {
	Enabled       *bool                    `json:"enabled,omitempty"`
	DefaultAction *string                  `json:"default_action,omitempty"`
	Rules         []VmwareEdgeFirewallRule `json:"rules"`
}

// VmwareUpdateEdgeFirewallRule represents a single rule in an edge firewall update.
//
// This mirrors the read rule minus the two fields the backend does not
// accept from the client (enabled, description) — see VmwareEdgeFirewallRule.
type VmwareUpdateEdgeFirewallRule struct {
	Name            *string `json:"name,omitempty"`
	Action          string  `json:"action"`
	Protocol        *string `json:"protocol,omitempty"`
	Source          *string `json:"source,omitempty"`
	SourcePort      *string `json:"source_port,omitempty"`
	Destination     *string `json:"destination,omitempty"`
	DestinationPort *string `json:"destination_port,omitempty"`
}

// VmwareUpdateEdgeFirewallRequest replaces the full edge firewall rule set.
//
// Enabled and DefaultAction mean exactly what their pointer+omitempty tags
// promise: omitting a field leaves the current value alone. An update that carries
// only Rules keeps enabled and default_action as they were.
//
// Rules keeps set semantics: the submitted slice replaces the whole rule set, so
// a read-modify-write must resend the rules it wants to keep. An empty (or nil)
// Rules clears them.
type VmwareUpdateEdgeFirewallRequest struct {
	Enabled       *bool                          `json:"enabled,omitempty"`
	DefaultAction *string                        `json:"default_action,omitempty"`
	Rules         []VmwareUpdateEdgeFirewallRule `json:"rules"`
}

// Validate checks the update edge firewall request. Enabled and
// DefaultAction are genuinely optional; every provided rule must carry
// an action, which the backend requires.
func (r *VmwareUpdateEdgeFirewallRequest) Validate() error {
	if r.DefaultAction != nil && *r.DefaultAction == "" {
		return fmt.Errorf("default_action must not be empty when set")
	}
	for i, rule := range r.Rules {
		if rule.Action == "" {
			return fmt.Errorf("rules[%d]: action is required", i)
		}
	}
	return nil
}

// ===================== Edge NAT =====================

// Edge NAT rule types. The API accepts them case-insensitively and returns them
// lower case.
const (
	// VmwareEdgeNATTypeDNAT translates an inbound connection to an internal
	// address (destination NAT / port forwarding).
	VmwareEdgeNATTypeDNAT = "dnat"
	// VmwareEdgeNATTypeSNAT translates outbound traffic from an internal subnet
	// to the edge external address (source NAT).
	VmwareEdgeNATTypeSNAT = "snat"
)

// VmwareEdgeNATRule represents a single NAT rule as returned by the API.
type VmwareEdgeNATRule struct {
	// ID is the internal database id, and the value DeleteVmwareEdgeNATRule expects.
	// VcloudID is the id of the same rule as a vCloud object, for reference only —
	// it is not accepted by any endpoint.
	ID       *int    `json:"id,omitempty"`
	VcloudID *string `json:"vcloud_id,omitempty"`
	// OriginalIP is normalized by the backend: for a DNAT rule it is replaced with
	// the external address of the edge gateway, whatever the request sent. A caller
	// that compares a written value against the read one — a Terraform provider
	// diffing state — must expect this substitution.
	OriginalIP     *string `json:"original_ip,omitempty"`
	Description    *string `json:"description,omitempty"`
	Type           *string `json:"type,omitempty"`
	TranslatedIP   *string `json:"translated_ip,omitempty"`
	Protocol       *string `json:"protocol,omitempty"`
	OriginalPort   *string `json:"original_port,omitempty"`
	TranslatedPort *string `json:"translated_port,omitempty"`
	Enabled        *bool   `json:"enabled,omitempty"`
}

// VmwareEdgeNAT represents the edge NAT configuration.
//
// The API returns this one flat, as {"rules": [...]}, without the
// single-key envelope the firewall and VPN responses use.
type VmwareEdgeNAT struct {
	Rules []VmwareEdgeNATRule `json:"rules"`
}

// VmwareUpsertNATRuleRequest represents a request to create or update a NAT rule.
//
// A rule is created enabled unless Enabled says otherwise, and on
// update an omitted Enabled leaves the current value alone.
type VmwareUpsertNATRuleRequest struct {
	// RuleID selects an existing rule to update (VmwareEdgeNATRule.ID). Leave it
	// nil to create a new rule.
	RuleID *int   `json:"rule_id,omitempty"`
	Type   string `json:"type"`
	// OriginalIP is ignored for DNAT rules and replaced with the edge external
	// address - see VmwareEdgeNATRule.OriginalIP.
	OriginalIP     string `json:"original_ip"`
	Description    string `json:"description,omitempty"`
	Protocol       string `json:"protocol"`
	OriginalPort   string `json:"original_port,omitempty"`
	TranslatedIP   string `json:"translated_ip"`
	TranslatedPort string `json:"translated_port,omitempty"`
	Enabled        *bool  `json:"enabled,omitempty"`
}

// Validate checks the upsert NAT rule request.
func (r *VmwareUpsertNATRuleRequest) Validate() error {
	if r.Type == "" {
		return fmt.Errorf("type is required")
	}
	if r.Protocol == "" {
		return fmt.Errorf("protocol is required")
	}
	if r.OriginalIP == "" || r.TranslatedIP == "" {
		return fmt.Errorf("original_ip and translated_ip are required")
	}
	if r.RuleID != nil && *r.RuleID <= 0 {
		return fmt.Errorf("rule_id must be greater than 0 when set")
	}
	return nil
}

// ===================== Edge VPN =====================

// Edge VPN encryption types. Observed on the live API; the contract is the
// authority on the full set.
const (
	VmwareEdgeVPNEncryptionAES     = "aes"
	VmwareEdgeVPNEncryptionAES256  = "aes256"
	VmwareEdgeVPNEncryptionAESGCM  = "aesgcm"
	VmwareEdgeVPNEncryptionTripDES = "tripledes"
)

// Edge VPN Diffie-Hellman groups.
//
// The API accepts these case-insensitively on write but
// returns them UPPER case on read - "dh14" written comes back as "DH14". A caller
// that diffs a written value against the read one must compare case-insensitively.
const (
	VmwareEdgeVPNDiffieHellmanGroup2  = "dh2"
	VmwareEdgeVPNDiffieHellmanGroup5  = "dh5"
	VmwareEdgeVPNDiffieHellmanGroup14 = "dh14"
	VmwareEdgeVPNDiffieHellmanGroup15 = "dh15"
	VmwareEdgeVPNDiffieHellmanGroup16 = "dh16"
)

// VmwareEdgeVPNSharedKeyMinLength and VmwareEdgeVPNSharedKeyMaxLength bound the
// IPsec pre-shared key. The backend also requires at least one upper-case letter,
// one lower-case letter and one digit (error -12013).
const (
	VmwareEdgeVPNSharedKeyMinLength = 32
	VmwareEdgeVPNSharedKeyMaxLength = 128
)

// VmwareEdgeVPNTunnel represents a single IPsec VPN tunnel as returned by the API.
//
// The read model is not symmetric with the write model.
// PeerSubnets is a list here while VmwareUpsertVPNTunnelRequest.PeerNetwork is a
// single string; LocalID, LocalIP, LocalSubnets, Description and DigestAlgorithm
// are derived by the backend and cannot be set; DiffieHellmanGroup comes back
// upper case. Compare read against written values accordingly.
type VmwareEdgeVPNTunnel struct {
	// ID is the internal database id, and the value DeleteVmwareEdgeVPNTunnel
	// expects. VcloudID is the vCloud site id, for reference only.
	ID                    *int     `json:"id,omitempty"`
	VcloudID              *string  `json:"vcloud_id,omitempty"`
	Enabled               *bool    `json:"enabled,omitempty"`
	Name                  *string  `json:"name,omitempty"`
	Description           *string  `json:"description,omitempty"`
	LocalID               *string  `json:"local_id,omitempty"`
	LocalIP               *string  `json:"local_ip,omitempty"`
	LocalSubnets          []string `json:"local_subnets,omitempty"`
	PeerIdentificator     *string  `json:"peer_identificator,omitempty"`
	PeerEndpoint          *string  `json:"peer_endpoint,omitempty"`
	PeerSubnets           []string `json:"peer_subnets,omitempty"`
	MTU                   *int     `json:"mtu,omitempty"`
	PerfectForwardSecrecy *bool    `json:"perfect_forward_secrecy,omitempty"`
	EncryptionType        *string  `json:"encryption_type,omitempty"`
	DigestAlgorithm       *string  `json:"digest_algorithm,omitempty"`
	DiffieHellmanGroup    *string  `json:"diffie_hellman_group,omitempty"`
}

// VmwareEdgeVPN represents the edge VPN configuration.
type VmwareEdgeVPN struct {
	Enabled *bool                 `json:"enabled,omitempty"`
	Tunnels []VmwareEdgeVPNTunnel `json:"tunnels"`
}

// VmwareUpsertVPNTunnelRequest represents a request to create or update a VPN tunnel.
//
// MTU, EncryptionType and DiffieHellmanGroup look optional (pointer / omitempty)
// but are mandatory: omitted, they reach the backend as zero values and fail its
// range and enum checks (-12011, -12090). Validate enforces their presence so the
// failure surfaces before the round trip.
type VmwareUpsertVPNTunnelRequest struct {
	// TunnelID selects an existing tunnel to update (VmwareEdgeVPNTunnel.ID).
	// Leave it nil to create a new tunnel.
	TunnelID *int   `json:"tunnel_id,omitempty"`
	Name     string `json:"name"`
	Enabled  *bool  `json:"enabled,omitempty"`
	MTU      *int   `json:"mtu,omitempty"`
	// EncryptionType is one of the VmwareEdgeVPNEncryption* constants.
	EncryptionType string `json:"encryption_type,omitempty"`
	// SharedKey is the IPsec pre-shared key: 32-128 alphanumeric characters with
	// at least one upper-case letter, one lower-case letter and one digit.
	SharedKey string `json:"shared_key"`
	// PeerNetwork is a single subnet in CIDR form. It is read back as the
	// PeerSubnets list.
	PeerNetwork           string `json:"peer_network"`
	PeerEndpoint          string `json:"peer_endpoint"`
	PeerIdentificator     string `json:"peer_identificator"`
	PerfectForwardSecrecy *bool  `json:"perfect_forward_secrecy,omitempty"`
	// DiffieHellmanGroup is one of the VmwareEdgeVPNDiffieHellmanGroup*
	// constants. It is read back upper case.
	DiffieHellmanGroup string `json:"diffie_hellman_group,omitempty"`
}

// Validate checks the upsert VPN tunnel request.
func (r *VmwareUpsertVPNTunnelRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if err := validateVmwareVPNSharedKey(r.SharedKey); err != nil {
		return err
	}
	if r.PeerNetwork == "" || r.PeerEndpoint == "" || r.PeerIdentificator == "" {
		return fmt.Errorf("peer_network, peer_endpoint and peer_identificator are required")
	}
	// mtu, diffie_hellman_group and encryption_type are mandatory despite the
	// optional-looking tags - the backend rejects the request without them.
	if r.MTU == nil {
		return fmt.Errorf("mtu is required")
	}
	if r.DiffieHellmanGroup == "" {
		return fmt.Errorf("diffie_hellman_group is required")
	}
	if r.EncryptionType == "" {
		return fmt.Errorf("encryption_type is required")
	}
	if r.TunnelID != nil && *r.TunnelID <= 0 {
		return fmt.Errorf("tunnel_id must be greater than 0 when set")
	}
	return nil
}

// validateVmwareVPNSharedKey mirrors the backend rule behind error -12013, so a
// bad key is reported locally instead of after a round trip.
func validateVmwareVPNSharedKey(key string) error {
	if key == "" {
		return fmt.Errorf("shared_key is required")
	}
	if len(key) < VmwareEdgeVPNSharedKeyMinLength || len(key) > VmwareEdgeVPNSharedKeyMaxLength {
		return fmt.Errorf("shared_key must be between %d and %d characters long",
			VmwareEdgeVPNSharedKeyMinLength, VmwareEdgeVPNSharedKeyMaxLength)
	}
	var hasUpper, hasLower, hasDigit bool
	for _, r := range key {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= '0' && r <= '9':
			hasDigit = true
		default:
			return fmt.Errorf("shared_key must be alphanumeric")
		}
	}
	if !hasUpper || !hasLower || !hasDigit {
		return fmt.Errorf("shared_key must contain at least one upper-case letter, one lower-case letter and one digit")
	}
	return nil
}

// ===================== Edge bandwidth =====================

// VmwareUpdateEdgeBandwidthRequest represents a request to set the uplink
// bandwidth (QoS) of a routed network's edge gateway.
//
// Edge bandwidth and network bandwidth are the same field, so the value is read
// back from VmwareNetwork.BandwidthMbps and the allowed range is the one the
// location's policy publishes; a value outside it is refused with -12041.
//
// On a platform deployment older than the one that fixed the endpoint, the
// request completes its task without persisting anything — the network keeps
// reporting the previous bandwidth. Use EditVmwareNetwork against such a
// deployment.
type VmwareUpdateEdgeBandwidthRequest struct {
	BandwidthMbps int `json:"bandwidth_mbps"`
}

// Validate checks the update edge bandwidth request.
func (r *VmwareUpdateEdgeBandwidthRequest) Validate() error {
	if r.BandwidthMbps <= 0 {
		return fmt.Errorf("bandwidth_mbps must be greater than 0")
	}
	return nil
}
