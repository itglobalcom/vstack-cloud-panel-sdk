package entities

import "fmt"

// Gateway represents an edge gateway
type Gateway struct {
	ID            string         `json:"id"`
	LocationID    string         `json:"location_id"`
	Name          string         `json:"name"`
	NICs          []GatewayNIC   `json:"nics"`
	NATRules      []NATRule      `json:"nat_rules,omitempty"`
	FirewallRules []FirewallRule `json:"firewall_rules,omitempty"`
	State         string         `json:"state"`
	PoweredOn     bool           `json:"powered_on"`
	Created       string         `json:"created"`
	Tags          []string       `json:"tags,omitempty"`
}

// GatewayNIC represents a gateway network interface
type GatewayNIC struct {
	ID            int    `json:"id"`
	NetworkID     string `json:"network_id"`
	IPAddress     string `json:"ip_address"`
	BandwidthMbps int    `json:"bandwidth_mbps"`
}

// NATRule represents a NAT rule
type NATRule struct {
	Type            string `json:"type"`     // SNAT, DNAT, BINAT
	Protocol        string `json:"protocol"` // TCP, ICMP, UDP, IP
	Source          string `json:"source"`
	Destination     string `json:"destination"`
	DestinationPort int    `json:"destination_port,omitempty"`
	Translated      string `json:"translated"`
	TranslatedPort  int    `json:"translated_port,omitempty"`
}

// Validate checks if the NAT rule is valid
func (r *NATRule) Validate() error {
	switch r.Type {
	case NATTypeSNAT, NATTypeDNAT, NATTypeBINAT:
		// valid
	default:
		if r.Type == "" {
			return fmt.Errorf("type is required")
		}
		return fmt.Errorf("invalid NAT type: %s", r.Type)
	}

	switch r.Protocol {
	case ProtocolTCP, ProtocolUDP, ProtocolICMP, ProtocolIP:
		// valid
	default:
		if r.Protocol == "" {
			return fmt.Errorf("protocol is required")
		}
		return fmt.Errorf("invalid protocol: %s", r.Protocol)
	}

	if r.Source == "" {
		return fmt.Errorf("source is required")
	}
	if r.Destination == "" {
		return fmt.Errorf("destination is required")
	}
	if r.Translated == "" {
		return fmt.Errorf("translated is required")
	}

	return nil
}

// FirewallRule represents a firewall rule
type FirewallRule struct {
	Action          string `json:"action"`    // Deny, Allow
	Direction       string `json:"direction"` // In, Out
	Protocol        string `json:"protocol"`  // TCP, ICMP, UDP, IP
	Source          string `json:"source"`
	SourcePort      int    `json:"source_port,omitempty"`
	Destination     string `json:"destination"`
	DestinationPort int    `json:"destination_port,omitempty"`
}

// Validate checks if the firewall rule is valid
func (r *FirewallRule) Validate() error {
	switch r.Action {
	case FirewallActionAllow, FirewallActionDeny:
		// valid
	default:
		if r.Action == "" {
			return fmt.Errorf("action is required")
		}
		return fmt.Errorf("invalid action: %s", r.Action)
	}

	switch r.Direction {
	case FirewallDirectionIn, FirewallDirectionOut:
		// valid
	default:
		if r.Direction == "" {
			return fmt.Errorf("direction is required")
		}
		return fmt.Errorf("invalid direction: %s", r.Direction)
	}

	switch r.Protocol {
	case ProtocolTCP, ProtocolUDP, ProtocolICMP, ProtocolIP:
		// valid
	default:
		if r.Protocol == "" {
			return fmt.Errorf("protocol is required")
		}
		return fmt.Errorf("invalid protocol: %s", r.Protocol)
	}

	if r.Source == "" {
		return fmt.Errorf("source is required")
	}
	if r.Destination == "" {
		return fmt.Errorf("destination is required")
	}

	return nil
}

// CreateGatewayRequest represents a request to create a gateway
type CreateGatewayRequest struct {
	LocationID    string   `json:"location_id"`
	Name          string   `json:"name"`
	BandwidthMbps int      `json:"bandwidth_mbps"`
	NetworkIDs    []string `json:"network_ids"`
}

// UpdateGatewayRequest represents a request to update gateway name
type UpdateGatewayRequest struct {
	Name string `json:"name"`
}

// UpdateGatewayBandwidthRequest represents a request to update gateway bandwidth
type UpdateGatewayBandwidthRequest struct {
	BandwidthMbps int `json:"bandwidth_mbps"`
}

// UpdateNATRulesRequest represents a request to update NAT rules
type UpdateNATRulesRequest struct {
	NATRules []NATRule `json:"nat_rules"`
}

// Validate validates each NAT rule in the request
func (r *UpdateNATRulesRequest) Validate() error {
	for i, rule := range r.NATRules {
		if err := rule.Validate(); err != nil {
			return fmt.Errorf("nat_rules[%d]: %w", i, err)
		}
	}
	return nil
}

// UpdateFirewallRulesRequest represents a request to update firewall rules
type UpdateFirewallRulesRequest struct {
	FirewallRules []FirewallRule `json:"firewall_rules"`
}

// Validate validates each firewall rule in the request
func (r *UpdateFirewallRulesRequest) Validate() error {
	for i, rule := range r.FirewallRules {
		if err := rule.Validate(); err != nil {
			return fmt.Errorf("firewall_rules[%d]: %w", i, err)
		}
	}
	return nil
}

// ConnectNetworkRequest represents a request to connect network to gateway
type ConnectNetworkRequest struct {
	NetworkID string `json:"network_id"`
}

// CreateGatewayTagRequest represents a request to create a gateway tag
type CreateGatewayTagRequest struct {
	Value string `json:"value"`
}

// Gateway state constants
const (
	GatewayStateNew     = "New"
	GatewayStateActive  = "Active"
	GatewayStateBusy    = "Busy"
	GatewayStateBlocked = "Blocked"
)

// NAT rule type constants
const (
	NATTypeSNAT  = "SNAT"
	NATTypeDNAT  = "DNAT"
	NATTypeBINAT = "BINAT"
)

// Protocol constants
const (
	ProtocolTCP  = "TCP"
	ProtocolUDP  = "UDP"
	ProtocolICMP = "ICMP"
	ProtocolIP   = "IP"
)

// Firewall action constants
const (
	FirewallActionAllow = "Allow"
	FirewallActionDeny  = "Deny"
)

// Firewall direction constants
const (
	FirewallDirectionIn  = "In"
	FirewallDirectionOut = "Out"
)

// Validate checks if the create gateway request is valid
func (r *CreateGatewayRequest) Validate() error {
	if r.LocationID == "" {
		return fmt.Errorf("location_id is required")
	}
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if r.BandwidthMbps <= 0 {
		return fmt.Errorf("bandwidth_mbps must be greater than 0")
	}
	if len(r.NetworkIDs) == 0 {
		return fmt.Errorf("at least one network_id is required")
	}
	return nil
}

// Validate checks if the update gateway request is valid
func (r *UpdateGatewayRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

// Validate checks if the update bandwidth request is valid
func (r *UpdateGatewayBandwidthRequest) Validate() error {
	if r.BandwidthMbps <= 0 {
		return fmt.Errorf("bandwidth_mbps must be greater than 0")
	}
	return nil
}

// Validate checks if the connect network request is valid
func (r *ConnectNetworkRequest) Validate() error {
	if r.NetworkID == "" {
		return fmt.Errorf("network_id is required")
	}
	return nil
}

// Validate checks if the create tag request is valid
func (r *CreateGatewayTagRequest) Validate() error {
	if r.Value == "" {
		return fmt.Errorf("tag value is required")
	}
	return nil
}

// IsActive checks if the gateway is in active state
func (g *Gateway) IsActive() bool {
	return g.State == GatewayStateActive
}

// IsPoweredOn checks if the gateway is powered on
func (g *Gateway) IsPoweredOn() bool {
	return g.PoweredOn
}
