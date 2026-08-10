package entities

import "fmt"

// VmwareNetwork type constants (C-9). These are the values returned in
// VmwareNetwork.Type and form the public type enum. There is no published enum
// for VmwareNetwork.State.
const (
	// VmwareNetworkTypePrivateClient - an isolated client network.
	VmwareNetworkTypePrivateClient = "private_client"
	// VmwareNetworkTypeRoutedClient - a routed client network (carries an edge).
	VmwareNetworkTypeRoutedClient = "routed_client"
	// VmwareNetworkTypePublicClient - a public client network.
	VmwareNetworkTypePublicClient = "public_client"
	// VmwareNetworkTypePublicShared - a shared public network (IPv4).
	VmwareNetworkTypePublicShared = "public_shared"
	// VmwareNetworkTypePublicSharedIPv6 - a shared public network (IPv6).
	VmwareNetworkTypePublicSharedIPv6 = "public_shared_ipv6"
)

// VmwareNetwork represents a VMware network.
type VmwareNetwork struct {
	ID            int     `json:"id"`
	LocationID    int     `json:"location_id"`
	Type          string  `json:"type"`
	Name          string  `json:"name"`
	Address       *string `json:"address,omitempty"`
	Mask          *int    `json:"mask,omitempty"`
	Gateway       *string `json:"gateway,omitempty"`
	BandwidthMbps *int    `json:"bandwidth_mbps,omitempty"`
	IsDhcp        *bool   `json:"is_dhcp,omitempty"`
	Shared        *bool   `json:"shared,omitempty"`
	State         string  `json:"state"`
	NicsCount     int     `json:"nics_count"`
}

// VmwareCreateIsolatedNetworkRequest represents a request to create an isolated
// (private) network.
type VmwareCreateIsolatedNetworkRequest struct {
	LocationID int    `json:"location_id"`
	Name       string `json:"name"`
	Address    string `json:"address"`
	Mask       *int   `json:"mask,omitempty"`
	EnableDhcp *bool  `json:"enable_dhcp,omitempty"`
}

// Validate checks the create isolated network request.
func (r *VmwareCreateIsolatedNetworkRequest) Validate() error {
	if r.LocationID <= 0 {
		return fmt.Errorf("location_id is required")
	}
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if r.Address == "" {
		return fmt.Errorf("address is required")
	}
	return nil
}

// VmwareCreateRoutedNetworkRequest represents a request to create a routed network.
type VmwareCreateRoutedNetworkRequest struct {
	LocationID    int    `json:"location_id"`
	Name          string `json:"name"`
	Address       string `json:"address"`
	Mask          *int   `json:"mask,omitempty"`
	EnableDhcp    *bool  `json:"enable_dhcp,omitempty"`
	BandwidthMbps *int   `json:"bandwidth_mbps,omitempty"`
}

// Validate checks the create routed network request.
func (r *VmwareCreateRoutedNetworkRequest) Validate() error {
	if r.LocationID <= 0 {
		return fmt.Errorf("location_id is required")
	}
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if r.Address == "" {
		return fmt.Errorf("address is required")
	}
	return nil
}

// VmwareCreatePublicNetworkRequest represents a request to create a public network.
type VmwareCreatePublicNetworkRequest struct {
	LocationID    int    `json:"location_id"`
	Name          string `json:"name"`
	Capacity      string `json:"capacity"`
	BandwidthMbps *int   `json:"bandwidth_mbps,omitempty"`
}

// Validate checks the create public network request.
func (r *VmwareCreatePublicNetworkRequest) Validate() error {
	if r.LocationID <= 0 {
		return fmt.Errorf("location_id is required")
	}
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if r.Capacity == "" {
		return fmt.Errorf("capacity is required")
	}
	return nil
}

// VmwareEditNetworkRequest represents a request to edit a network. Both fields
// are optional; at least one must be provided.
type VmwareEditNetworkRequest struct {
	Name          string `json:"name,omitempty"`
	BandwidthMbps *int   `json:"bandwidth_mbps,omitempty"`
}

// Validate performs a soft check of the edit network request (C-6): at least one
// field must be set, and bandwidth, if present, must be positive.
func (r *VmwareEditNetworkRequest) Validate() error {
	if r.Name == "" && r.BandwidthMbps == nil {
		return fmt.Errorf("at least one of name or bandwidth_mbps must be provided")
	}
	if r.BandwidthMbps != nil && *r.BandwidthMbps <= 0 {
		return fmt.Errorf("bandwidth_mbps must be greater than 0")
	}
	return nil
}

// VmwareConnectServerNIC identifies a server to attach to a network, with an
// optional IP.
//
// C-12: Nic -> NIC.
type VmwareConnectServerNIC struct {
	ServerID int    `json:"server_id"`
	IP       string `json:"ip,omitempty"`
}

// VmwareConnectServersRequest represents a request to attach servers to a network.
type VmwareConnectServersRequest struct {
	NICs               []VmwareConnectServerNIC `json:"nics"` // C-12: Nics -> NICs
	ForceCustomization *bool                    `json:"force_customization,omitempty"`
}

// Validate checks the connect servers request.
func (r *VmwareConnectServersRequest) Validate() error {
	if len(r.NICs) == 0 {
		return fmt.Errorf("at least one nic is required")
	}
	for i, n := range r.NICs {
		if n.ServerID <= 0 {
			return fmt.Errorf("nics[%d]: server_id is required", i)
		}
	}
	return nil
}
