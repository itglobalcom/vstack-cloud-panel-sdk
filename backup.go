package sdk

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

const (
	backupPath              = "backup"
	backupStoragesPath      = "storages"
	backupRestorePointsPath = "restore-points"
	backupRestorePath       = "restore"
	backupRestoreNearbyPath = "restore-nearby"
)

// Response types for Backup operations
type (
	// ListBackupRestorePointsResponse represents a list of backup restore points response
	ListBackupRestorePointsResponse struct {
		RestorePoints []entities.BackupRestorePoint `json:"restore_points,omitempty"`
	}
)

// buildServerBackupPath constructs the path for server backup operations
func buildServerBackupPath(serverID string, paths ...string) string {
	path := fmt.Sprintf("servers/%s/%s", serverID, backupPath)
	for _, p := range paths {
		if p != "" {
			path = fmt.Sprintf("%s/%s", path, p)
		}
	}
	return path
}

// buildServerBackupRestorePointPath constructs the path for operations on a
// single backup restore point
func buildServerBackupRestorePointPath(serverID string, restorePointID int, paths ...string) string {
	return buildServerBackupPath(serverID, append([]string{backupRestorePointsPath, fmt.Sprintf("%d", restorePointID)}, paths...)...)
}

// GetBackupStorageList retrieves the backup storages available to a new server
// ordered in the location (techTitle) from the image. An empty imageID returns
// the storages regardless of the image.
func (c *CloudClient) GetBackupStorageList(ctx context.Context, locationID, imageID string) (*entities.BackupStorageCatalog, error) {
	if locationID == "" {
		return nil, fmt.Errorf("location ID is required")
	}

	params := url.Values{}
	params.Set("location_id", locationID)
	if imageID != "" {
		params.Set("image_id", imageID)
	}
	path := withQuery(fmt.Sprintf("%s/%s", backupPath, backupStoragesPath), params)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list backup storages request for location %s: %w", locationID, err)
	}

	var resp entities.BackupStorageCatalog
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list backup storages for location %s: %w", locationID, err)
	}

	return &resp, nil
}

// GetServerBackupStorages retrieves the backup storages available to a server
func (c *CloudClient) GetServerBackupStorages(ctx context.Context, serverID string) (*entities.BackupStorageCatalog, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server ID is required")
	}

	path := buildServerBackupPath(serverID, backupStoragesPath)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list backup storages request for server %s: %w", serverID, err)
	}

	var resp entities.BackupStorageCatalog
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list backup storages for server %s: %w", serverID, err)
	}

	return &resp, nil
}

// GetServerBackup retrieves the state of the backup service of a server. A
// server without the service answers Enabled false, not an error.
func (c *CloudClient) GetServerBackup(ctx context.Context, serverID string) (*entities.ServerBackup, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server ID is required")
	}

	path := buildServerBackupPath(serverID)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get backup request for server %s: %w", serverID, err)
	}

	var resp entities.ServerBackup
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get backup of server %s: %w", serverID, err)
	}

	return &resp, nil
}

// EnableServerBackup enables the backup service of a server with the schedule
// and returns a task ID
func (c *CloudClient) EnableServerBackup(ctx context.Context, serverID string, req *entities.BackupSchedule) (*TaskID, error) {
	return c.sendServerBackupSchedule(ctx, http.MethodPost, "enable", serverID, req)
}

// EnableServerBackupAndWait enables the backup service of a server and waits for
// completion
func (c *CloudClient) EnableServerBackupAndWait(ctx context.Context, serverID string, req *entities.BackupSchedule) (*entities.ServerBackup, error) {
	task, err := c.EnableServerBackup(ctx, serverID, req)
	if err != nil {
		return nil, err
	}

	if _, err := c.waitTaskCompletion(ctx, task.ID); err != nil {
		return nil, fmt.Errorf("failed to wait for backup enabling on server %s: %w", serverID, err)
	}

	return c.GetServerBackup(ctx, serverID)
}

// UpdateServerBackup replaces the backup schedule of a server and returns a task
// ID. A change that keeps the storages of the rules completes synchronously and
// returns AlreadyCompletedTaskID.
func (c *CloudClient) UpdateServerBackup(ctx context.Context, serverID string, req *entities.BackupSchedule) (*TaskID, error) {
	return c.sendServerBackupSchedule(ctx, http.MethodPut, "update", serverID, req)
}

// UpdateServerBackupAndWait replaces the backup schedule of a server and waits
// for completion
func (c *CloudClient) UpdateServerBackupAndWait(ctx context.Context, serverID string, req *entities.BackupSchedule) (*entities.ServerBackup, error) {
	task, err := c.UpdateServerBackup(ctx, serverID, req)
	if err != nil {
		return nil, err
	}

	if _, err := c.waitTaskCompletion(ctx, task.ID); err != nil {
		return nil, fmt.Errorf("failed to wait for backup update on server %s: %w", serverID, err)
	}

	return c.GetServerBackup(ctx, serverID)
}

func (c *CloudClient) sendServerBackupSchedule(ctx context.Context, method, action, serverID string, req *entities.BackupSchedule) (*TaskID, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server ID is required")
	}
	if req == nil {
		return nil, fmt.Errorf("backup schedule is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid backup schedule: %w", err)
	}

	path := buildServerBackupPath(serverID)
	httpReq, err := c.newRequest(ctx, method, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s backup request for server %s: %w", action, serverID, err)
	}

	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to %s backup of server %s: %w", action, serverID, err)
	}

	return &task, nil
}

// DisableServerBackup disables the backup service of a server and returns a task
// ID. The copies of the server are deleted together with the service.
func (c *CloudClient) DisableServerBackup(ctx context.Context, serverID string) (*TaskID, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server ID is required")
	}

	path := buildServerBackupPath(serverID)
	req, err := c.newRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create disable backup request for server %s: %w", serverID, err)
	}

	var task TaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to disable backup of server %s: %w", serverID, err)
	}

	return &task, nil
}

// DisableServerBackupAndWait disables the backup service of a server and waits
// for completion
func (c *CloudClient) DisableServerBackupAndWait(ctx context.Context, serverID string) error {
	task, err := c.DisableServerBackup(ctx, serverID)
	if err != nil {
		return err
	}

	if _, err := c.waitTaskCompletion(ctx, task.ID); err != nil {
		return fmt.Errorf("failed to wait for backup disabling on server %s: %w", serverID, err)
	}

	return nil
}

// GetServerBackupRestorePoints retrieves the backup restore points of a server
func (c *CloudClient) GetServerBackupRestorePoints(ctx context.Context, serverID string) ([]entities.BackupRestorePoint, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server ID is required")
	}

	path := buildServerBackupPath(serverID, backupRestorePointsPath)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list backup restore points request for server %s: %w", serverID, err)
	}

	var resp ListBackupRestorePointsResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list backup restore points for server %s: %w", serverID, err)
	}

	return resp.RestorePoints, nil
}

// CreateServerBackupRestorePoint starts a manual backup copy of a server and
// returns a task ID. The task completes together with the copy; the outcome of
// the copy is the state of its restore point. A nil request lets the API name
// the copy.
func (c *CloudClient) CreateServerBackupRestorePoint(ctx context.Context, serverID string, req *entities.CreateBackupRestorePointRequest) (*TaskID, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server ID is required")
	}
	if req == nil {
		req = &entities.CreateBackupRestorePointRequest{}
	}

	path := buildServerBackupPath(serverID, backupRestorePointsPath)
	httpReq, err := c.newRequest(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup restore point request for server %s: %w", serverID, err)
	}

	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to create backup restore point on server %s: %w", serverID, err)
	}

	return &task, nil
}

// CreateServerBackupRestorePointAndWait starts a manual backup copy of a server,
// waits for its task and returns the new restore point, found as the manual
// point missing from the list read before the copy. A point the platform failed
// to take is returned as an error matching IsBackupRestorePointFailed.
func (c *CloudClient) CreateServerBackupRestorePointAndWait(ctx context.Context, serverID string, req *entities.CreateBackupRestorePointRequest) (*entities.BackupRestorePoint, error) {
	initialPoints, err := c.GetServerBackupRestorePoints(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to get initial backup restore points: %w", err)
	}

	initialIDs := make(map[int]bool, len(initialPoints))
	for _, point := range initialPoints {
		initialIDs[point.ID] = true
	}

	task, err := c.CreateServerBackupRestorePoint(ctx, serverID, req)
	if err != nil {
		return nil, err
	}

	if _, err := c.waitTaskCompletion(ctx, task.ID); err != nil {
		return nil, fmt.Errorf("failed to wait for backup restore point creation on server %s: %w", serverID, err)
	}

	currentPoints, err := c.GetServerBackupRestorePoints(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to get backup restore points after task completion: %w", err)
	}

	var matched []*entities.BackupRestorePoint
	for i := range currentPoints {
		point := &currentPoints[i]
		if initialIDs[point.ID] || !point.IsManual {
			continue
		}
		if req != nil && req.Name != "" && point.Name != req.Name {
			continue
		}
		matched = append(matched, point)
	}

	if len(matched) == 0 {
		return nil, fmt.Errorf("no new backup restore point found on server %s after task completion", serverID)
	}

	if len(matched) > 1 && c.logger != nil {
		c.logger.Warn(fmt.Sprintf("Found %d new backup restore points, expected 1. Returning first one.", len(matched)))
	}

	point := matched[0]
	switch point.State {
	case entities.BackupRestorePointStateActive:
		return point, nil
	case entities.BackupRestorePointStateFailed:
		return nil, fmt.Errorf("backup restore point %d on server %s: %w", point.ID, serverID, ErrBackupRestorePointFailed)
	default:
		return nil, fmt.Errorf("backup restore point %d on server %s is %s after task completion", point.ID, serverID, point.State)
	}
}

// RestoreServerBackupRestorePoint restores a server over itself from a backup
// restore point and returns a task ID
func (c *CloudClient) RestoreServerBackupRestorePoint(ctx context.Context, serverID string, restorePointID int) (*TaskID, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server ID is required")
	}
	if restorePointID <= 0 {
		return nil, fmt.Errorf("restore point ID must be greater than 0")
	}

	path := buildServerBackupRestorePointPath(serverID, restorePointID, backupRestorePath)
	httpReq, err := c.newRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create restore request for backup restore point %d on server %s: %w", restorePointID, serverID, err)
	}

	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to restore server %s from backup restore point %d: %w", serverID, restorePointID, err)
	}

	return &task, nil
}

// RestoreServerBackupRestorePointAndWait restores a server over itself from a
// backup restore point, waits for completion and returns the server
func (c *CloudClient) RestoreServerBackupRestorePointAndWait(ctx context.Context, serverID string, restorePointID int) (*entities.Server, error) {
	task, err := c.RestoreServerBackupRestorePoint(ctx, serverID, restorePointID)
	if err != nil {
		return nil, err
	}

	if _, err := c.waitTaskCompletion(ctx, task.ID); err != nil {
		return nil, fmt.Errorf("failed to wait for server %s restore from backup restore point %d: %w", serverID, restorePointID, err)
	}

	return c.GetServer(ctx, serverID)
}

// RestoreServerBackupRestorePointNearby restores a backup restore point of a
// server into a new server and returns a task ID. A nil request restores into
// the location of the server. The task does not name the new server: it appears
// in the server list of the target location.
func (c *CloudClient) RestoreServerBackupRestorePointNearby(ctx context.Context, serverID string, restorePointID int, req *entities.RestoreBackupRestorePointNearbyRequest) (*TaskID, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server ID is required")
	}
	if restorePointID <= 0 {
		return nil, fmt.Errorf("restore point ID must be greater than 0")
	}
	if req == nil {
		req = &entities.RestoreBackupRestorePointNearbyRequest{}
	}

	path := buildServerBackupRestorePointPath(serverID, restorePointID, backupRestoreNearbyPath)
	httpReq, err := c.newRequest(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create restore nearby request for backup restore point %d on server %s: %w", restorePointID, serverID, err)
	}

	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to restore backup restore point %d of server %s nearby: %w", restorePointID, serverID, err)
	}

	return &task, nil
}

// RestoreServerBackupRestorePointNearbyAndWait restores a backup restore point
// of a server into a new server and waits for completion
func (c *CloudClient) RestoreServerBackupRestorePointNearbyAndWait(ctx context.Context, serverID string, restorePointID int, req *entities.RestoreBackupRestorePointNearbyRequest) error {
	task, err := c.RestoreServerBackupRestorePointNearby(ctx, serverID, restorePointID, req)
	if err != nil {
		return err
	}

	if _, err := c.waitTaskCompletion(ctx, task.ID); err != nil {
		return fmt.Errorf("failed to wait for backup restore point %d of server %s restore nearby: %w", restorePointID, serverID, err)
	}

	return nil
}

// DeleteServerBackupRestorePoint deletes a backup restore point and returns a
// task ID
func (c *CloudClient) DeleteServerBackupRestorePoint(ctx context.Context, serverID string, restorePointID int) (*TaskID, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server ID is required")
	}
	if restorePointID <= 0 {
		return nil, fmt.Errorf("restore point ID must be greater than 0")
	}

	params := url.Values{}
	params.Set("return_task", "true")
	path := withQuery(buildServerBackupRestorePointPath(serverID, restorePointID), params)
	req, err := c.newRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create delete request for backup restore point %d on server %s: %w", restorePointID, serverID, err)
	}

	var task TaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to delete backup restore point %d on server %s: %w", restorePointID, serverID, err)
	}

	return &task, nil
}

// DeleteServerBackupRestorePointAndWait deletes a backup restore point and waits
// for completion
func (c *CloudClient) DeleteServerBackupRestorePointAndWait(ctx context.Context, serverID string, restorePointID int) error {
	task, err := c.DeleteServerBackupRestorePoint(ctx, serverID, restorePointID)
	if err != nil {
		return err
	}

	if _, err := c.waitTaskCompletion(ctx, task.ID); err != nil {
		return fmt.Errorf("failed to wait for backup restore point %d deletion on server %s: %w", restorePointID, serverID, err)
	}

	return nil
}
