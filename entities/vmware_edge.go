package entities

import "fmt"

// ===================== Edge firewall =====================

// VmwareEdgeFirewallRule represents a single edge firewall rule.
//
// NET-7: flat and symmetric with the write model — the backend expands the vCloud
// applications[] into one flat rule per protocol+ports, so a read rule round-trips
// to a write rule. The vCloud-only id and the nested applications are gone.
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
type VmwareUpdateEdgeFirewallRule struct {
	Name            *string `json:"name,omitempty"`
	Action          string  `json:"action"`
	Protocol        *string `json:"protocol,omitempty"`
	Source          *string `json:"source,omitempty"`
	SourcePort      *string `json:"source_port,omitempty"`
	Destination     *string `json:"destination,omitempty"`
	DestinationPort *string `json:"destination_port,omitempty"`
}

// VmwareUpdateEdgeFirewallRequest replaces the full edge firewall configuration.
//
// SDK-N5: WARNING - omitting Enabled disables the network firewall. The pointer with
// omitempty reads as "leave this field unchanged", but on the backend a missing
// enabled means false, so a naive read-modify-write ("get firewall, append a
// rule, write it back") silently turns the firewall off (networks-sdk.md, SDK-N5,
// networks-api.md, NET-13). To guard against this, Validate requires Enabled and
// DefaultAction to be set explicitly; supply them from a preceding
// GetVmwareEdgeFirewall.
type VmwareUpdateEdgeFirewallRequest struct {
	Enabled       *bool                          `json:"enabled,omitempty"`
	DefaultAction *string                        `json:"default_action,omitempty"`
	Rules         []VmwareUpdateEdgeFirewallRule `json:"rules"`
}

// Validate checks the update edge firewall request (C-6).
func (r *VmwareUpdateEdgeFirewallRequest) Validate() error {
	// SDK-N5: require Enabled explicitly - omitting it disables the firewall on the
	// backend, so we refuse to send a request that could do so unintentionally.
	if r.Enabled == nil {
		return fmt.Errorf("enabled is required: omitting it disables the network firewall")
	}
	// SDK-N5: default_action must be set explicitly for the same reason.
	if r.DefaultAction == nil || *r.DefaultAction == "" {
		return fmt.Errorf("default_action is required")
	}
	for i, rule := range r.Rules {
		// C-6: action is a mandatory field of every firewall rule.
		if rule.Action == "" {
			return fmt.Errorf("rules[%d]: action is required", i)
		}
	}
	return nil
}

// ===================== Edge NAT =====================

// VmwareEdgeNATRule represents a single NAT rule as returned by the API.
//
// C-12: Nat -> NAT.
//
// SDK-N1: ID cannot be used to delete this rule. GET returns the vCloud object id,
// but DELETE expects the internal database id, so the value here is not accepted
// by DeleteVmwareEdgeNATRule - deleting a NAT rule via the SDK is currently
// impossible until the API is fixed (networks-sdk.md, SDK-N1, networks-api.md, NET-1).
type VmwareEdgeNATRule struct {
	// SDK-N1: ID is the internal DB id that DeleteVmwareEdgeNATRule accepts (NET-1, cloudmng);
	// VcloudID is the vCloud object id, exposed for reference only.
	ID             *int    `json:"id,omitempty"`
	VcloudID       *string `json:"vcloud_id,omitempty"`
	Description    *string `json:"description,omitempty"`
	Type           *string `json:"type,omitempty"`
	OriginalIP     *string `json:"original_ip,omitempty"`
	TranslatedIP   *string `json:"translated_ip,omitempty"`
	Protocol       *string `json:"protocol,omitempty"`
	OriginalPort   *string `json:"original_port,omitempty"`
	TranslatedPort *string `json:"translated_port,omitempty"`
	Enabled        *bool   `json:"enabled,omitempty"`
}

// VmwareEdgeNAT represents the edge NAT configuration.
//
// C-12: Nat -> NAT.
type VmwareEdgeNAT struct {
	Rules []VmwareEdgeNATRule `json:"rules"`
}

// VmwareUpsertNATRuleRequest represents a request to create or update a NAT rule.
//
// C-12: Nat -> NAT.
type VmwareUpsertNATRuleRequest struct {
	RuleID         *int   `json:"rule_id,omitempty"`
	Type           string `json:"type"`
	Description    string `json:"description,omitempty"`
	Protocol       string `json:"protocol"`
	OriginalIP     string `json:"original_ip"`
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
	return nil
}

// ===================== Edge VPN =====================

// VmwareEdgeVPNTunnel represents a single IPsec VPN tunnel as returned by the API.
//
// C-12: Vpn -> VPN.
//
// SDK-N1: ID cannot be used to delete this tunnel. GET returns the vCloud object id,
// but DELETE expects the internal database id, so the value here is not accepted
// by DeleteVmwareEdgeVPNTunnel - deleting a VPN tunnel via the SDK is currently
// impossible until the API is fixed (networks-sdk.md, SDK-N1, networks-api.md, NET-1).
type VmwareEdgeVPNTunnel struct {
	// SDK-N1: ID is the internal DB id that DeleteVmwareEdgeVPNTunnel accepts (NET-1); VcloudID is
	// the vCloud site id, for reference only.
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
	Mtu                   *int     `json:"mtu,omitempty"`
	PerfectForwardSecrecy *bool    `json:"perfect_forward_secrecy,omitempty"`
	EncryptionType        *string  `json:"encryption_type,omitempty"`
	DigestAlgorithm       *string  `json:"digest_algorithm,omitempty"`
	DiffieHellmanGroup    *string  `json:"diffie_hellman_group,omitempty"`
}

// VmwareEdgeVPN represents the edge VPN configuration.
//
// C-12: Vpn -> VPN.
type VmwareEdgeVPN struct {
	Enabled *bool                 `json:"enabled,omitempty"`
	Tunnels []VmwareEdgeVPNTunnel `json:"tunnels"`
}

// VmwareUpsertVPNTunnelRequest represents a request to create or update a VPN tunnel.
//
// C-12: Vpn -> VPN.
//
// SDK-N2: Mtu, EncryptionType and DiffieHellmanGroup are marked optional (pointer /
// omitempty) but are in fact mandatory - a request without them is rejected
// (-12011, -12090) because the backend receives zero values that fail the range
// and enum checks (networks-sdk.md, SDK-N2, networks-api.md, NET-6). Validate
// enforces their presence.
type VmwareUpsertVPNTunnelRequest struct {
	TunnelID              *int   `json:"tunnel_id,omitempty"`
	Name                  string `json:"name"`
	Enabled               *bool  `json:"enabled,omitempty"`
	Mtu                   *int   `json:"mtu,omitempty"`
	EncryptionType        string `json:"encryption_type,omitempty"`
	SharedKey             string `json:"shared_key"`
	PeerNetwork           string `json:"peer_network"`
	PeerEndpoint          string `json:"peer_endpoint"`
	PeerIdentificator     string `json:"peer_identificator"`
	PerfectForwardSecrecy *bool  `json:"perfect_forward_secrecy,omitempty"`
	DiffieHellmanGroup    string `json:"diffie_hellman_group,omitempty"`
}

// Validate checks the upsert VPN tunnel request.
func (r *VmwareUpsertVPNTunnelRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if r.SharedKey == "" {
		return fmt.Errorf("shared_key is required")
	}
	if r.PeerNetwork == "" || r.PeerEndpoint == "" || r.PeerIdentificator == "" {
		return fmt.Errorf("peer_network, peer_endpoint and peer_identificator are required")
	}
	// SDK-N2: mtu, diffie_hellman_group and encryption_type are mandatory despite the
	// optional-looking tags - the backend rejects the request without them.
	if r.Mtu == nil {
		return fmt.Errorf("mtu is required")
	}
	if r.DiffieHellmanGroup == "" {
		return fmt.Errorf("diffie_hellman_group is required")
	}
	if r.EncryptionType == "" {
		return fmt.Errorf("encryption_type is required")
	}
	return nil
}

// ===================== Edge bandwidth =====================

// VmwareEdgeBandwidthRequest represents a request to set the edge uplink bandwidth.
type VmwareEdgeBandwidthRequest struct {
	BandwidthMbps int `json:"bandwidth_mbps"`
}

// Validate checks the edge bandwidth request.
func (r *VmwareEdgeBandwidthRequest) Validate() error {
	if r.BandwidthMbps <= 0 {
		return fmt.Errorf("bandwidth_mbps must be greater than 0")
	}
	return nil
}
