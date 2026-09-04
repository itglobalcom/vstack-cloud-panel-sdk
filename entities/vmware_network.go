package entities

import (
	"fmt"
	"strconv"
)

// VmwareNetwork type constants. These are the values returned in
// VmwareNetwork.Type and form the public type enum.
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

// VmwareNetwork state constants. Unlike the type enum these are NOT published in
// the API contract - they are the values observed on live responses, collected
// here so callers (and WaitVmwareNetworkState) do not spell them inline. Treat an
// unrecognized state as "still settling" rather than as an error.
const (
	// VmwareNetworkStateCreating - the network is being provisioned.
	VmwareNetworkStateCreating = "creating"
	// VmwareNetworkStateActive - the network is ready for use.
	VmwareNetworkStateActive = "active"
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
	NICsCount     int     `json:"nics_count"`
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
	LocationID int    `json:"location_id"`
	Name       string `json:"name"`
	// Capacity is a subnet prefix length, not a number of addresses: the contract
	// declares NetworkCapacityEnum, whose members are Network24…Network29 and which
	// travels as a string. A decimal string such as "1" or "4" is accepted only
	// because the API's StringEnumConverter also parses a member's numeric value,
	// and the numbering does not follow the prefix — 1 is Network24, 2 is Network29,
	// 3 is Network25, 4 is Network26, 5 is Network27, 6 is Network28. Anything
	// outside the enum (8 and above) is refused with -12042 "The capacity of public
	// network is invalid"; a valid value can still fail with -12043 "There is no
	// free network at the moment" when the location has no free block left.
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
	// The field is a string, but the backend parses it as a number and answers a
	// bare -2002 ("the request body is not formatted") for anything else, which
	// names neither the field nor the value. Catch it here instead.
	if n, err := strconv.Atoi(r.Capacity); err != nil || n <= 0 {
		return fmt.Errorf("capacity must be a positive number of addresses, got %q", r.Capacity)
	}
	return nil
}

// VmwareEditNetworkRequest represents a request to edit a network. Both fields
// are optional; at least one must be provided.
//
// BandwidthMbps here is the ONLY way to set the bandwidth of a routed
// network's edge - edge bandwidth and network bandwidth are one field. The former
// PUT /edge/bandwidth endpoint is not exposed by the SDK because it never
// persisted the value (see the note at the bottom of entities/vmware_edge.go).
//
// Bandwidth does not apply to an isolated
// (private_client) network, and such a network no longer reports one -
// VmwareNetwork.BandwidthMbps comes back nil. Sending one anyway is still refused
// with the generic -12041 "Network bandwidth outside allowable limits" rather than
// a "not applicable" error, so send only Name when editing an isolated network.
type VmwareEditNetworkRequest struct {
	Name          string `json:"name,omitempty"`
	BandwidthMbps *int   `json:"bandwidth_mbps,omitempty"`
}

// Validate performs a soft check of the edit network request: at least one
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
type VmwareConnectServerNIC struct {
	ServerID int    `json:"server_id"`
	IP       string `json:"ip,omitempty"`
}

// VmwareConnectServersRequest represents a request to attach servers to a network.
type VmwareConnectServersRequest struct {
	NICs               []VmwareConnectServerNIC `json:"nics"`
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
