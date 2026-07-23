package sdk

import (
	"context"
	"fmt"
	"net/http"

	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

const (
	sshKeyBaseURL = "ssh-keys"
)

// Response types
type (
	GetSSHKeyResponse struct {
		SSHKey *entities.SSHKey `json:"ssh_key,omitempty"`
	}

	ListSSHKeysResponse struct {
		SSHKeys []*entities.SSHKey `json:"ssh_keys,omitempty"`
	}
)

// buildSSHKeyPath constructs the path for SSH key operations
func buildSSHKeyPath(keyID int) string {
	if keyID == 0 {
		return sshKeyBaseURL
	}
	return fmt.Sprintf("%s/%d", sshKeyBaseURL, keyID)
}

// GetSSHKey retrieves a specific SSH key by ID
func (c *CloudClient) GetSSHKey(ctx context.Context, keyID int) (*entities.SSHKey, error) {
	if keyID == 0 {
		return nil, fmt.Errorf("SSH key ID is required")
	}

	path := buildSSHKeyPath(keyID)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for SSH key %d: %w", keyID, err)
	}

	var resp GetSSHKeyResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get SSH key %d: %w", keyID, err)
	}

	if resp.SSHKey == nil {
		return nil, fmt.Errorf("SSH key %d not found in response: %w", keyID, ErrNotFound)
	}

	return resp.SSHKey, nil
}

// GetSSHKeyList retrieves all SSH keys
func (c *CloudClient) GetSSHKeyList(ctx context.Context) ([]*entities.SSHKey, error) {
	req, err := c.newRequest(ctx, http.MethodGet, sshKeyBaseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list SSH keys request: %w", err)
	}

	var resp ListSSHKeysResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list SSH keys: %w", err)
	}

	return resp.SSHKeys, nil
}

// CreateSSHKey creates a new SSH key
func (c *CloudClient) CreateSSHKey(ctx context.Context, req *entities.CreateSSHKeyRequest) (*entities.SSHKey, error) {
	if req == nil {
		return nil, fmt.Errorf("create SSH key request is required")
	}

	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid create SSH key request: %w", err)
	}

	httpReq, err := c.newRequest(ctx, http.MethodPost, sshKeyBaseURL, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH key request: %w", err)
	}

	var sshKey entities.SSHKey
	if err := c.doJSON(httpReq, &sshKey); err != nil {
		return nil, fmt.Errorf("failed to create SSH key: %w", err)
	}

	return &sshKey, nil
}

// DeleteSSHKey deletes an SSH key
func (c *CloudClient) DeleteSSHKey(ctx context.Context, keyID int) error {
	if keyID == 0 {
		return fmt.Errorf("SSH key ID is required")
	}

	path := buildSSHKeyPath(keyID)
	req, err := c.newRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request for SSH key %d: %w", keyID, err)
	}

	if err := c.doJSON(req, nil); err != nil {
		return fmt.Errorf("failed to delete SSH key %d: %w", keyID, err)
	}

	return nil
}
