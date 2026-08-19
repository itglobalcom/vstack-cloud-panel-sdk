package entities

import "fmt"

// Network represents an isolated network
type Network struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	LocationID    string   `json:"location_id"`
	Description   string   `json:"description"`
	NetworkPrefix string   `json:"network_prefix"`
	Mask          int      `json:"mask"`
	ServerIDs     []string `json:"server_ids"`
	GatewayIDs    []string `json:"gateway_ids"`
	State         string   `json:"state"`
	Created       string   `json:"created"`
	Tags          []string `json:"tags"`
}

// CreateNetworkRequest represents a request to create an isolated network.
type CreateNetworkRequest struct {
	Name          string `json:"name"`
	LocationID    string `json:"location_id"`
	Description   string `json:"description,omitempty"`
	NetworkPrefix string `json:"network_prefix,omitempty"`
	Mask          int    `json:"mask,omitempty"`
}

// UpdateNetworkRequest represents a request to rename an existing network.
type UpdateNetworkRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Validate checks the create network request.
func (r *CreateNetworkRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if r.LocationID == "" {
		return fmt.Errorf("location_id is required")
	}
	return nil
}

// Validate checks the update network request.
func (r *UpdateNetworkRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name must be provided")
	}
	return nil
}
