package entities

import "fmt"

// NIC represents a server network interface
type NIC struct {
	ID            int    `json:"id"`
	ServerID      string `json:"server_id"`
	NetworkType   string `json:"network_type,omitempty"`
	NetworkID     string `json:"network_id"`
	MAC           string `json:"mac"`
	IPAddress     string `json:"ip_address"`
	Mask          int    `json:"mask"`
	Gateway       string `json:"gateway"`
	BandwidthMbps int    `json:"bandwidth_mbps"`
}

// CreateNICRequest represents a request to create a network interface
type CreateNICRequest struct {
	// For isolated network connection
	NetworkID string `json:"network_id,omitempty"`
	IPAddress string `json:"ip_address,omitempty"`

	// For public network connection
	BandwidthMbps int `json:"bandwidth_mbps,omitempty"`
}

// Validate validates the create NIC request
func (r *CreateNICRequest) Validate() error {
	// Must specify either network_id (isolated) or bandwidth_mbps (public)
	if r.NetworkID == "" && r.BandwidthMbps == 0 {
		return fmt.Errorf("either network_id or bandwidth_mbps must be specified")
	}

	// Cannot specify both
	if r.NetworkID != "" && r.BandwidthMbps > 0 {
		return fmt.Errorf("cannot specify both network_id and bandwidth_mbps")
	}

	// Validate bandwidth if specified
	if r.BandwidthMbps > 0 && r.BandwidthMbps%10 != 0 {
		return fmt.Errorf("bandwidth must be a multiple of 10 Mbps")
	}

	return nil
}

// IsIsolatedNetwork returns true if this request is for an isolated network
func (r *CreateNICRequest) IsIsolatedNetwork() bool {
	return r.NetworkID != ""
}

// IsPublicNetwork returns true if this request is for a public network
func (r *CreateNICRequest) IsPublicNetwork() bool {
	return r.BandwidthMbps > 0
}

// UpdateNICRequest represents a request to update a network interface
type UpdateNICRequest struct {
	BandwidthMbps int `json:"bandwidth_mbps"`
}

// Validate validates the update NIC request
func (r *UpdateNICRequest) Validate() error {
	if r.BandwidthMbps <= 0 {
		return fmt.Errorf("bandwidth_mbps must be greater than 0")
	}

	if r.BandwidthMbps%10 != 0 {
		return fmt.Errorf("bandwidth must be a multiple of 10 Mbps")
	}

	return nil
}
