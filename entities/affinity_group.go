package entities

import "fmt"

// AffinityGroup represents an affinity or anti-affinity group
type AffinityGroup struct {
	ID         string   `json:"id"`
	LocationID string   `json:"location_id"`
	Name       string   `json:"name"`
	Affinity   bool     `json:"affinity"`
	ServerIDs  []string `json:"server_ids"`
}

// CreateAffinityGroupRequest represents a request to create an affinity group
type CreateAffinityGroupRequest struct {
	Name       string `json:"name"`
	LocationID string `json:"location_id"`
	Affinity   bool   `json:"affinity"`
}

// Validate checks if the create affinity group request is valid
func (r *CreateAffinityGroupRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if r.LocationID == "" {
		return fmt.Errorf("location_id is required")
	}
	return nil
}
