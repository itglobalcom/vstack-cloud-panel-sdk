package sdk

import (
	"context"
	"fmt"
	"net/http"

	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

const (
	networkBaseURL = "networks/isolated"
	tagsPath       = "tags"
)

// Response types
type (
	GetNetworkResponse struct {
		Network *entities.Network `json:"isolated_network,omitempty"`
	}

	ListNetworksResponse struct {
		Networks []*entities.Network `json:"isolated_networks,omitempty"`
	}
)

// Request types
type AddNetworkTagRequest struct {
	Tag string `json:"value" binding:"required"`
}

// buildNetworkPath constructs the path for network operations
func buildNetworkPath(networkID string, parts ...string) string {
	path := fmt.Sprintf("%s/%s", networkBaseURL, networkID)
	for _, part := range parts {
		path = fmt.Sprintf("%s/%s", path, part)
	}
	return path
}

// GetNetwork retrieves a specific network by ID
func (c *CloudClient) GetNetwork(ctx context.Context, networkID string) (*entities.Network, error) {
	if networkID == "" {
		return nil, fmt.Errorf("network ID is required")
	}

	path := buildNetworkPath(networkID)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for network %s: %w", networkID, err)
	}

	var resp GetNetworkResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get network %s: %w", networkID, err)
	}

	if resp.Network == nil {
		return nil, fmt.Errorf("network %s not found in response: %w", networkID, ErrNotFound)
	}

	return resp.Network, nil
}

// GetNetworkList retrieves all networks
func (c *CloudClient) GetNetworkList(ctx context.Context) ([]*entities.Network, error) {
	req, err := c.newRequest(ctx, http.MethodGet, networkBaseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list networks request: %w", err)
	}

	var resp ListNetworksResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}

	return resp.Networks, nil
}

// CreateNetwork creates a new isolated network
func (c *CloudClient) CreateNetwork(ctx context.Context, req *entities.CreateNetworkRequest) (*TaskID, error) {
	if req == nil {
		return nil, fmt.Errorf("create network request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid create network request: %w", err)
	}

	httpReq, err := c.newRequest(ctx, http.MethodPost, networkBaseURL, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create network request: %w", err)
	}

	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to create network: %w", err)
	}

	return &task, nil
}

// CreateNetworkAndWait creates a network and waits for completion
func (c *CloudClient) CreateNetworkAndWait(ctx context.Context, req *entities.CreateNetworkRequest) (*entities.Network, error) {
	task, err := c.CreateNetwork(ctx, req)
	if err != nil {
		return nil, err
	}

	completedTask, err := c.waitTaskCompletion(ctx, task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for network task %s: %w", task.ID, err)
	}

	network, err := c.GetNetwork(ctx, completedTask.NetworkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get network %s after task completion: %w", completedTask.NetworkID, err)
	}

	return network, nil
}

// UpdateNetwork updates the name and description of a network
func (c *CloudClient) UpdateNetwork(ctx context.Context, networkID string, req *entities.UpdateNetworkRequest) (*entities.Network, error) {
	if networkID == "" {
		return nil, fmt.Errorf("network ID is required")
	}
	if req == nil {
		return nil, fmt.Errorf("update network request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid update network request: %w", err)
	}

	path := buildNetworkPath(networkID)
	httpReq, err := c.newRequest(ctx, http.MethodPut, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create update request for network %s: %w", networkID, err)
	}

	var resp GetNetworkResponse
	if err := c.doJSON(httpReq, &resp); err != nil {
		return nil, fmt.Errorf("failed to update network %s: %w", networkID, err)
	}

	return resp.Network, nil
}

// DeleteNetwork deletes a network and returns a task ID
func (c *CloudClient) DeleteNetwork(ctx context.Context, networkID string) error {
	if networkID == "" {
		return fmt.Errorf("network ID is required")
	}

	path := buildNetworkPath(networkID)
	req, err := c.newRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request for network %s: %w", networkID, err)
	}

	if err := c.doJSON(req, nil); err != nil {
		return fmt.Errorf("failed to delete network %s: %w", networkID, err)
	}

	return nil
}

// AddNetworkTag adds a tag to a network
func (c *CloudClient) AddNetworkTag(ctx context.Context, networkID string, req *AddNetworkTagRequest) error {
	if networkID == "" {
		return fmt.Errorf("network ID is required")
	}
	if req == nil || req.Tag == "" {
		return fmt.Errorf("tag is required")
	}

	path := buildNetworkPath(networkID, tagsPath)
	httpReq, err := c.newRequest(ctx, http.MethodPost, path, req)
	if err != nil {
		return fmt.Errorf("failed to create add tag request for network %s: %w", networkID, err)
	}

	if err := c.doJSON(httpReq, nil); err != nil {
		return fmt.Errorf("failed to add tag to network %s: %w", networkID, err)
	}

	return nil
}

// DeleteNetworkTag removes a tag from an isolated network
func (c *CloudClient) DeleteNetworkTag(ctx context.Context, networkID, tag string) error {
	if networkID == "" {
		return fmt.Errorf("network ID is required")
	}
	if tag == "" {
		return fmt.Errorf("tag is required")
	}

	path := buildNetworkPath(networkID, fmt.Sprintf("%s/%s", tagsPath, tag))
	httpReq, err := c.newRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete tag request for network %s: %w", networkID, err)
	}

	if err := c.doJSON(httpReq, nil); err != nil {
		return fmt.Errorf("failed to delete tag from network %s: %w", networkID, err)
	}

	return nil
}
