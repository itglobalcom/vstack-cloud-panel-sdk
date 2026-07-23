package sdk

import (
	"context"
	"fmt"
	"net/http"

	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

const (
	snapshotsPath = "snapshots"
	rollbackPath  = "rollback"
)

// Response types for Snapshot operations
type (
	// GetSnapshotResponse represents a single snapshot response
	GetSnapshotResponse struct {
		Snapshot *entities.Snapshot `json:"snapshot,omitempty"`
	}

	// ListSnapshotsResponse represents a list of snapshots response
	ListSnapshotsResponse struct {
		Snapshots []entities.Snapshot `json:"snapshots,omitempty"`
	}
)

// buildServerSnapshotPath constructs the path for server snapshot operations
func buildServerSnapshotPath(serverID string, snapshotID int, paths ...string) string {
	path := fmt.Sprintf("servers/%s/%s", serverID, snapshotsPath)
	if snapshotID > 0 {
		path = fmt.Sprintf("%s/%d", path, snapshotID)
	}
	for _, p := range paths {
		if p != "" {
			path = fmt.Sprintf("%s/%s", path, p)
		}
	}
	return path
}

// GetServerSnapshots retrieves all snapshots for a server
func (c *CloudClient) GetServerSnapshots(ctx context.Context, serverID string) ([]entities.Snapshot, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server ID is required")
	}

	path := buildServerSnapshotPath(serverID, 0)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list snapshots request for server %s: %w", serverID, err)
	}

	var resp ListSnapshotsResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list snapshots for server %s: %w", serverID, err)
	}

	return resp.Snapshots, nil
}

// GetServerSnapshot retrieves a specific snapshot by ID
func (c *CloudClient) GetServerSnapshot(ctx context.Context, serverID string, snapshotID int) (*entities.Snapshot, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server ID is required")
	}
	if snapshotID <= 0 {
		return nil, fmt.Errorf("snapshot ID must be greater than 0")
	}

	path := buildServerSnapshotPath(serverID, snapshotID)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for snapshot %d on server %s: %w", snapshotID, serverID, err)
	}

	var resp GetSnapshotResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get snapshot %d on server %s: %w", snapshotID, serverID, err)
	}

	if resp.Snapshot == nil {
		return nil, fmt.Errorf("snapshot %d not found on server %s: %w", snapshotID, serverID, ErrNotFound)
	}

	return resp.Snapshot, nil
}

// CreateServerSnapshot creates a new snapshot and returns a task ID
func (c *CloudClient) CreateServerSnapshot(ctx context.Context, serverID string, req *entities.CreateSnapshotRequest) (*TaskID, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server ID is required")
	}
	if req == nil {
		return nil, fmt.Errorf("create snapshot request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid create snapshot request: %w", err)
	}

	path := buildServerSnapshotPath(serverID, 0)
	httpReq, err := c.newRequest(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot request for server %s: %w", serverID, err)
	}

	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to create snapshot on server %s: %w", serverID, err)
	}

	return &task, nil
}

// CreateServerSnapshotAndWait creates a snapshot and waits for server to become Active
func (c *CloudClient) CreateServerSnapshotAndWait(ctx context.Context, serverID string, req *entities.CreateSnapshotRequest) (*entities.Snapshot, error) {
	// Get initial snapshot IDs before creation
	initialSnapshots, err := c.GetServerSnapshots(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to get initial snapshots: %w", err)
	}

	// Create map of existing snapshot IDs
	initialIDMap := make(map[int]bool)
	for _, snap := range initialSnapshots {
		initialIDMap[snap.ID] = true
	}

	// Create snapshot
	task, err := c.CreateServerSnapshot(ctx, serverID, req)
	if err != nil {
		return nil, err
	}

	// Wait for task completion
	completedTask, err := c.waitTaskCompletion(ctx, task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for snapshot creation: %w", err)
	}

	// Wait for server to become Active
	if _, err := c.WaitServerActive(ctx, serverID); err != nil {
		return nil, fmt.Errorf("failed to wait for server to become Active: %w", err)
	}

	// Try to get snapshot ID from task result first
	if completedTask.SnapshotID != 0 {
		return c.GetServerSnapshot(ctx, serverID, completedTask.SnapshotID)
	}

	// If snapshot ID not in task result, find the new snapshot by comparing lists
	currentSnapshots, err := c.GetServerSnapshots(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshots after task completion: %w", err)
	}

	// Find new snapshots that match the request criteria
	var matchedSnapshots []*entities.Snapshot
	for i := range currentSnapshots {
		snap := &currentSnapshots[i]

		// Skip if not a new snapshot
		if initialIDMap[snap.ID] {
			continue
		}

		// Verify server_id matches
		if snap.ServerID != serverID {
			continue
		}

		// Match by name
		if snap.Name == req.Name {
			matchedSnapshots = append(matchedSnapshots, snap)
		}
	}

	if len(matchedSnapshots) == 0 {
		return nil, fmt.Errorf("no new snapshot found after task completion")
	}

	if len(matchedSnapshots) > 1 && c.logger != nil {
		c.logger.Warn(fmt.Sprintf("Found %d matching snapshots, expected 1. Returning first one.", len(matchedSnapshots)))
	}

	return matchedSnapshots[0], nil
}

// RollbackServerSnapshot rolls back server to a snapshot and returns a task ID
func (c *CloudClient) RollbackServerSnapshot(ctx context.Context, serverID string, snapshotID int) (*TaskID, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server ID is required")
	}
	if snapshotID <= 0 {
		return nil, fmt.Errorf("snapshot ID must be greater than 0")
	}

	path := buildServerSnapshotPath(serverID, snapshotID, rollbackPath)
	httpReq, err := c.newRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create rollback request for snapshot %d on server %s: %w", snapshotID, serverID, err)
	}

	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to rollback to snapshot %d on server %s: %w", snapshotID, serverID, err)
	}

	return &task, nil
}

// RollbackServerSnapshotAndWait rolls back server to a snapshot and waits for completion
func (c *CloudClient) RollbackServerSnapshotAndWait(ctx context.Context, serverID string, snapshotID int) (*entities.Server, error) {
	task, err := c.RollbackServerSnapshot(ctx, serverID, snapshotID)
	if err != nil {
		return nil, err
	}

	// Wait for task completion
	if _, err := c.waitTaskCompletion(ctx, task.ID); err != nil {
		return nil, fmt.Errorf("failed to wait for snapshot rollback: %w", err)
	}

	// Get the server state after rollback
	return c.GetServer(ctx, serverID)
}

// DeleteServerSnapshot deletes a snapshot
func (c *CloudClient) DeleteServerSnapshot(ctx context.Context, serverID string, snapshotID int) error {
	if serverID == "" {
		return fmt.Errorf("server ID is required")
	}
	if snapshotID <= 0 {
		return fmt.Errorf("snapshot ID must be greater than 0")
	}

	path := buildServerSnapshotPath(serverID, snapshotID)
	req, err := c.newRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request for snapshot %d on server %s: %w", snapshotID, serverID, err)
	}

	if err := c.doJSON(req, nil); err != nil {
		return fmt.Errorf("failed to delete snapshot %d on server %s: %w", snapshotID, serverID, err)
	}

	return nil
}

// DeleteServerSnapshotAndWait deletes a snapshot and waits for server to become Active
func (c *CloudClient) DeleteServerSnapshotAndWait(ctx context.Context, serverID string, snapshotID int) error {
	err := c.DeleteServerSnapshot(ctx, serverID, snapshotID)
	if err != nil {
		return err
	}

	// Wait for task completion and server to become Active
	if _, err := c.WaitServerActive(ctx, serverID); err != nil {
		return fmt.Errorf("failed to wait for snapshot deletion: %w", err)
	}

	return nil
}
