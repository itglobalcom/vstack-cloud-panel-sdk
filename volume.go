package sdk

import (
	"context"
	"fmt"
	"net/http"

	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

const (
	volumesPath = "volumes"
)

// Response types for Volume operations
type (
	// GetVolumeResponse represents a single volume response
	GetVolumeResponse struct {
		Volume *entities.Volume `json:"volume,omitempty"`
	}

	// ListVolumesResponse represents a list of volumes response
	ListVolumesResponse struct {
		Volumes []entities.Volume `json:"volumes,omitempty"`
	}
)

// buildServerVolumePath constructs the path for server volume operations
func buildServerVolumePath(serverID string, volumeID int) string {
	path := fmt.Sprintf("servers/%s/%s", serverID, volumesPath)
	if volumeID > 0 {
		path = fmt.Sprintf("%s/%d", path, volumeID)
	}
	return path
}

// GetServerVolumes retrieves all volumes for a server
func (c *CloudClient) GetServerVolumes(ctx context.Context, serverID string) ([]entities.Volume, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server ID is required")
	}

	path := buildServerVolumePath(serverID, 0)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list volumes request for server %s: %w", serverID, err)
	}

	var resp ListVolumesResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list volumes for server %s: %w", serverID, err)
	}

	return resp.Volumes, nil
}

// GetServerVolume retrieves a specific volume by ID
func (c *CloudClient) GetServerVolume(ctx context.Context, serverID string, volumeID int) (*entities.Volume, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server ID is required")
	}
	if volumeID <= 0 {
		return nil, fmt.Errorf("volume ID must be greater than 0")
	}

	path := buildServerVolumePath(serverID, volumeID)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for volume %d on server %s: %w", volumeID, serverID, err)
	}

	var resp GetVolumeResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get volume %d on server %s: %w", volumeID, serverID, err)
	}

	if resp.Volume == nil {
		return nil, fmt.Errorf("volume %d not found on server %s: %w", volumeID, serverID, ErrNotFound)
	}

	return resp.Volume, nil
}

// CreateServerVolume creates a new volume and returns a task ID
func (c *CloudClient) CreateServerVolume(ctx context.Context, serverID string, req *entities.CreateVolumeRequest) (*TaskID, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server ID is required")
	}
	if req == nil {
		return nil, fmt.Errorf("create volume request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid create volume request: %w", err)
	}

	path := buildServerVolumePath(serverID, 0)
	httpReq, err := c.newRequest(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create volume request for server %s: %w", serverID, err)
	}

	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to create volume on server %s: %w", serverID, err)
	}

	return &task, nil
}

// CreateServerVolumeAndWait creates a volume and waits for server to become Active
func (c *CloudClient) CreateServerVolumeAndWait(ctx context.Context, serverID string, req *entities.CreateVolumeRequest) (*entities.Volume, error) {
	task, err := c.CreateServerVolume(ctx, serverID, req)
	if err != nil {
		return nil, err
	}

	// Wait for task completion
	completedTask, err := c.waitTaskCompletion(ctx, task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for volume creation: %w", err)
	}

	// Get volume ID from task result
	if completedTask.VolumeID == 0 {
		return nil, fmt.Errorf("volume ID not found in task result")
	}

	// Wait for server to become Active
	if _, err := c.WaitServerActive(ctx, serverID); err != nil {
		return nil, fmt.Errorf("failed to wait for server to become Active: %w", err)
	}

	// Get the created volume
	return c.GetServerVolume(ctx, serverID, completedTask.VolumeID)
}

// UpdateServerVolume updates a volume and returns a task ID
func (c *CloudClient) UpdateServerVolume(ctx context.Context, serverID string, volumeID int, req *entities.UpdateVolumeRequest) (*TaskID, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server ID is required")
	}
	if volumeID <= 0 {
		return nil, fmt.Errorf("volume ID must be greater than 0")
	}
	if req == nil {
		return nil, fmt.Errorf("update volume request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid update volume request: %w", err)
	}

	path := buildServerVolumePath(serverID, volumeID)
	httpReq, err := c.newRequest(ctx, http.MethodPut, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create update request for volume %d on server %s: %w", volumeID, serverID, err)
	}

	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to update volume %d on server %s: %w", volumeID, serverID, err)
	}

	return &task, nil
}

// UpdateServerVolumeAndWait updates a volume and waits for server to become Active
func (c *CloudClient) UpdateServerVolumeAndWait(ctx context.Context, serverID string, volumeID int, req *entities.UpdateVolumeRequest) (*entities.Volume, error) {
	task, err := c.UpdateServerVolume(ctx, serverID, volumeID, req)
	if err != nil {
		return nil, err
	}

	// Wait for task completion and server to become Active
	if _, err := c.WaitServerTaskCompletion(ctx, serverID, task.ID); err != nil {
		return nil, fmt.Errorf("failed to wait for volume update: %w", err)
	}

	// Get the updated volume
	return c.GetServerVolume(ctx, serverID, volumeID)
}

// DeleteServerVolume deletes a volume
func (c *CloudClient) DeleteServerVolume(ctx context.Context, serverID string, volumeID int) error {
	if serverID == "" {
		return fmt.Errorf("server ID is required")
	}
	if volumeID <= 0 {
		return fmt.Errorf("volume ID must be greater than 0")
	}

	path := buildServerVolumePath(serverID, volumeID)
	req, err := c.newRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request for volume %d on server %s: %w", volumeID, serverID, err)
	}

	if err := c.doJSON(req, nil); err != nil {
		return fmt.Errorf("failed to delete volume %d on server %s: %w", volumeID, serverID, err)
	}

	return nil
}

// DeleteServerVolumeAndWait deletes a volume and waits for server to become Active
func (c *CloudClient) DeleteServerVolumeAndWait(ctx context.Context, serverID string, volumeID int) error {
	err := c.DeleteServerVolume(ctx, serverID, volumeID)
	if err != nil {
		return err
	}

	// Wait for task completion and server to become Active
	if _, err := c.WaitServerActive(ctx, serverID); err != nil {
		return fmt.Errorf("failed to wait for volume deletion: %w", err)
	}

	return nil
}
