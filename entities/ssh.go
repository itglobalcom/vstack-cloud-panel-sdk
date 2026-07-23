package entities

import "fmt"

// SSHKey represents an SSH key
type SSHKey struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
}

// Request types
type CreateSSHKeyRequest struct {
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
}

// Validate checks if the create SSH key request is valid
func (r *CreateSSHKeyRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if r.PublicKey == "" {
		return fmt.Errorf("public_key is required")
	}
	return nil
}
