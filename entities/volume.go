package entities

import "fmt"

// Volume represents a server volume
type Volume struct {
	ID       int    `json:"id"`
	ServerID string `json:"server_id"`
	Name     string `json:"name"`
	SizeMB   int    `json:"size_mb"`
	Created  string `json:"created"`
}

// CreateVolumeRequest represents a request to create a volume
type CreateVolumeRequest struct {
	Name   string `json:"name"`
	SizeMB int    `json:"size_mb"`
}

// Validate validates the create volume request
func (r *CreateVolumeRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("volume name is required")
	}

	if r.SizeMB <= 0 {
		return fmt.Errorf("volume size must be greater than 0")
	}

	// Volume size must be a multiple of 10 GB (10240 MB)
	if r.SizeMB%(10*1024) != 0 {
		return fmt.Errorf("volume size must be a multiple of 10 GB (10240 MB)")
	}

	return nil
}

// UpdateVolumeRequest represents a request to update a volume
type UpdateVolumeRequest struct {
	Name   string `json:"name,omitempty"`
	SizeMB int    `json:"size_mb"`
}

// Validate validates the update volume request
func (r *UpdateVolumeRequest) Validate() error {
	if r.SizeMB <= 0 {
		return fmt.Errorf("volume size must be greater than 0")
	}

	// Volume size must be a multiple of 10 GB (10240 MB)
	if r.SizeMB%(10*1024) != 0 {
		return fmt.Errorf("volume size must be a multiple of 10 GB (10240 MB)")
	}

	return nil
}
