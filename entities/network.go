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

type CreateNetworkRequest struct {
	Name          string `json:"name"`
	LocationID    string `json:"location_id"`
	Description   string `json:"description,omitempty"`
	NetworkPrefix string `json:"network_prefix,omitempty"`
	Mask          int    `json:"mask,omitempty"`
}

type UpdateNetworkRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

func (r *CreateNetworkRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if r.LocationID == "" {
		return fmt.Errorf("location_id is required")
	}
	return nil
}

func (r *UpdateNetworkRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name must be provided")
	}
	return nil
}
