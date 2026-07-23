package entities

import "fmt"

// Snapshot represents a server snapshot
type Snapshot struct {
	ID       int    `json:"id"`
	ServerID string `json:"server_id"`
	Name     string `json:"name"`
	SizeMB   int    `json:"size_mb"`
	Created  string `json:"created"`
}

// CreateSnapshotRequest represents a request to create a snapshot
type CreateSnapshotRequest struct {
	Name string `json:"name"`
}

// Validate validates the create snapshot request
func (r *CreateSnapshotRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("snapshot name is required")
	}
	return nil
}
