package sdk

import (
	"context"
	"fmt"
	"net/http"

	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

const (
	serverBaseURL = "servers"
)

// Response types
type (
	GetServerResponse struct {
		Server *entities.Server `json:"server,omitempty"`
	}

	ListServersResponse struct {
		Servers []*entities.Server `json:"servers,omitempty"`
	}
)

// buildServerPath constructs the path for server operations
func buildServerPath(serverID string, parts ...string) string {
	path := fmt.Sprintf("%s/%s", serverBaseURL, serverID)
	for _, part := range parts {
		path = fmt.Sprintf("%s/%s", path, part)
	}
	return path
}

// GetServer retrieves a specific server by ID
func (c *CloudClient) GetServer(ctx context.Context, serverID string) (*entities.Server, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server ID is required")
	}

	path := buildServerPath(serverID)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for server %s: %w", serverID, err)
	}

	var resp GetServerResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get server %s: %w", serverID, err)
	}

	if resp.Server == nil {
		return nil, fmt.Errorf("server %s not found in response: %w", serverID, ErrNotFound)
	}

	return resp.Server, nil
}

// GetServerList retrieves all servers
func (c *CloudClient) GetServerList(ctx context.Context) ([]*entities.Server, error) {
	req, err := c.newRequest(ctx, http.MethodGet, serverBaseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list servers request: %w", err)
	}

	var resp ListServersResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list servers: %w", err)
	}

	return resp.Servers, nil
}

// CreateServer creates a new server and returns a task ID
func (c *CloudClient) CreateServer(ctx context.Context, req *entities.CreateServerRequest) (*TaskID, error) {
	if req == nil {
		return nil, fmt.Errorf("create server request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid create server request: %w", err)
	}

	httpReq, err := c.newRequest(ctx, http.MethodPost, serverBaseURL, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create server request: %w", err)
	}

	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	return &task, nil
}

// CreateServerAndWait creates a server and waits for it to become Active
func (c *CloudClient) CreateServerAndWait(ctx context.Context, req *entities.CreateServerRequest) (*entities.Server, error) {
	task, err := c.CreateServer(ctx, req)
	if err != nil {
		return nil, err
	}

	// Wait for task completion
	completedTask, err := c.waitTaskCompletion(ctx, task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for server creation: %w", err)
	}

	// Get server ID from task result
	if completedTask.ServerID == "" {
		return nil, fmt.Errorf("server ID not found in task result")
	}

	// Wait for server to become Active
	return c.WaitServerActive(ctx, completedTask.ServerID)
}

// UpdateServer updates server resources (PUT - both CPU and RAM required) and returns a task ID
func (c *CloudClient) UpdateServer(ctx context.Context, serverID string, req *entities.UpdateServerRequest) (*TaskID, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server ID is required")
	}
	if req == nil {
		return nil, fmt.Errorf("update server request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid update server request: %w", err)
	}

	path := buildServerPath(serverID)
	httpReq, err := c.newRequest(ctx, http.MethodPut, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create update request for server %s: %w", serverID, err)
	}

	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to update server %s: %w", serverID, err)
	}

	return &task, nil
}

// UpdateServerAndWait updates server and waits for it to become Active
func (c *CloudClient) UpdateServerAndWait(ctx context.Context, serverID string, req *entities.UpdateServerRequest) (*entities.Server, error) {
	task, err := c.UpdateServer(ctx, serverID, req)
	if err != nil {
		return nil, err
	}

	// Wait for task completion and server to become Active
	return c.WaitServerTaskCompletion(ctx, serverID, task.ID)
}

// PatchServer updates server resources (PATCH - CPU or RAM or both) and returns a task ID
func (c *CloudClient) PatchServer(ctx context.Context, serverID string, req *entities.PatchServerRequest) (*TaskID, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server ID is required")
	}
	if req == nil {
		return nil, fmt.Errorf("patch server request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid patch server request: %w", err)
	}

	path := buildServerPath(serverID)
	httpReq, err := c.newRequest(ctx, http.MethodPatch, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create patch request for server %s: %w", serverID, err)
	}

	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to patch server %s: %w", serverID, err)
	}

	return &task, nil
}

// PatchServerAndWait patches server and waits for it to become Active
func (c *CloudClient) PatchServerAndWait(ctx context.Context, serverID string, req *entities.PatchServerRequest) (*entities.Server, error) {
	task, err := c.PatchServer(ctx, serverID, req)
	if err != nil {
		return nil, err
	}

	// Wait for task completion and server to become Active
	return c.WaitServerTaskCompletion(ctx, serverID, task.ID)
}

// RenameServer changes the server name and returns a task ID
func (c *CloudClient) RenameServer(ctx context.Context, serverID string, req *entities.RenameServerRequest) (*TaskID, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server ID is required")
	}
	if req == nil {
		return nil, fmt.Errorf("rename server request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid rename server request: %w", err)
	}

	path := buildServerPath(serverID, "name")
	httpReq, err := c.newRequest(ctx, http.MethodPut, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create rename request for server %s: %w", serverID, err)
	}

	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to rename server %s: %w", serverID, err)
	}

	return &task, nil
}

// RenameServerAndWait renames server and waits for it to become Active
func (c *CloudClient) RenameServerAndWait(ctx context.Context, serverID string, req *entities.RenameServerRequest) (*entities.Server, error) {
	task, err := c.RenameServer(ctx, serverID, req)
	if err != nil {
		return nil, err
	}

	// Wait for task completion and server to become Active
	return c.WaitServerTaskCompletion(ctx, serverID, task.ID)
}

// DeleteServer deletes a server
func (c *CloudClient) DeleteServer(ctx context.Context, serverID string) error {
	if serverID == "" {
		return fmt.Errorf("server ID is required")
	}

	path := buildServerPath(serverID)
	req, err := c.newRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request for server %s: %w", serverID, err)
	}

	if err := c.doJSON(req, nil); err != nil {
		return fmt.Errorf("failed to delete server %s: %w", serverID, err)
	}

	return nil
}

// CreateServerTag creates tag for server
func (c *CloudClient) CreateServerTag(ctx context.Context, serverID string, req *entities.CreateServerTagRequest) error {
	if serverID == "" {
		return fmt.Errorf("server ID is required")
	}

	path := buildServerPath(serverID, "tags")
	httpReq, err := c.newRequest(ctx, http.MethodPost, path, req)
	if err != nil {
		return fmt.Errorf("failed to create new tag request for server %s: %w", serverID, err)
	}

	if err := c.doJSON(httpReq, nil); err != nil {
		return fmt.Errorf("failed to create tag for server %s: %w", serverID, err)
	}

	return nil
}

// DeleteServerTag delete tag for server
func (c *CloudClient) DeleteServerTag(ctx context.Context, serverID string, tag string) error {
	if serverID == "" {
		return fmt.Errorf("server ID is required")
	}

	path := buildServerPath(serverID, "tags", tag)
	httpReq, err := c.newRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("failed to delete tag request for server %s: %w", serverID, err)
	}

	if err := c.doJSON(httpReq, nil); err != nil {
		return fmt.Errorf("failed to delete tag for server %s: %w", serverID, err)
	}

	return nil
}

// GetServerPrice retrieves the monthly price for a server configuration
func (c *CloudClient) GetServerPrice(ctx context.Context, req *entities.GetServerPriceRequest) (float64, error) {
	if req == nil {
		return 0, fmt.Errorf("get server price request is required")
	}

	if err := req.Validate(); err != nil {
		return 0, fmt.Errorf("invalid get server price request: %w", err)
	}

	httpReq, err := c.newRequest(ctx, http.MethodPost, "servers/price", req)
	if err != nil {
		return 0, fmt.Errorf("failed to create get server price request: %w", err)
	}

	var resp entities.GetServerPriceResponse
	if err := c.doJSON(httpReq, &resp); err != nil {
		return 0, fmt.Errorf("failed to get server price: %w", err)
	}

	return resp.Price, nil
}

// Power management operations

// PowerOnServer turns on the server power and returns a task ID
func (c *CloudClient) PowerOnServer(ctx context.Context, serverID string) (*TaskID, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server ID is required")
	}

	path := buildServerPath(serverID, "power", "on")
	req, err := c.newRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create power on request for server %s: %w", serverID, err)
	}

	var task TaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to power on server %s: %w", serverID, err)
	}

	return &task, nil
}

// PowerOnServerAndWait powers on server and waits for it to become Active
func (c *CloudClient) PowerOnServerAndWait(ctx context.Context, serverID string) (*entities.Server, error) {
	task, err := c.PowerOnServer(ctx, serverID)
	if err != nil {
		return nil, err
	}

	// Wait for task completion and server to become Active
	return c.WaitServerTaskCompletion(ctx, serverID, task.ID)
}

// PowerOffServer shuts down a server via operating system (graceful shutdown) and returns a task ID
func (c *CloudClient) PowerOffServer(ctx context.Context, serverID string) (*TaskID, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server ID is required")
	}

	path := buildServerPath(serverID, "power", "off")
	req, err := c.newRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create power off request for server %s: %w", serverID, err)
	}

	var task TaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to power off server %s: %w", serverID, err)
	}

	return &task, nil
}

// PowerOffServerAndWait powers off server and waits for it to become Active
func (c *CloudClient) PowerOffServerAndWait(ctx context.Context, serverID string) (*entities.Server, error) {
	task, err := c.PowerOffServer(ctx, serverID)
	if err != nil {
		return nil, err
	}

	// Wait for task completion and server to become Active
	return c.WaitServerTaskCompletion(ctx, serverID, task.ID)
}

// ShutdownServer shuts down a server via power off (hard shutdown) and returns a task ID
func (c *CloudClient) ShutdownServer(ctx context.Context, serverID string) (*TaskID, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server ID is required")
	}

	path := buildServerPath(serverID, "power", "shutdown")
	req, err := c.newRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create shutdown request for server %s: %w", serverID, err)
	}

	var task TaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to shutdown server %s: %w", serverID, err)
	}

	return &task, nil
}

// ShutdownServerAndWait shuts down server and waits for it to become Active
func (c *CloudClient) ShutdownServerAndWait(ctx context.Context, serverID string) (*entities.Server, error) {
	task, err := c.ShutdownServer(ctx, serverID)
	if err != nil {
		return nil, err
	}

	// Wait for task completion and server to become Active
	return c.WaitServerTaskCompletion(ctx, serverID, task.ID)
}

// RebootServer soft reboots a server (via OS) and returns a task ID
func (c *CloudClient) RebootServer(ctx context.Context, serverID string) (*TaskID, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server ID is required")
	}

	path := buildServerPath(serverID, "power", "reboot")
	req, err := c.newRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create reboot request for server %s: %w", serverID, err)
	}

	var task TaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to reboot server %s: %w", serverID, err)
	}

	return &task, nil
}

// RebootServerAndWait reboots server and waits for it to become Active
func (c *CloudClient) RebootServerAndWait(ctx context.Context, serverID string) (*entities.Server, error) {
	task, err := c.RebootServer(ctx, serverID)
	if err != nil {
		return nil, err
	}

	// Wait for task completion and server to become Active
	return c.WaitServerTaskCompletion(ctx, serverID, task.ID)
}

// ResetServer hard reboots a server (power cycle) and returns a task ID
func (c *CloudClient) ResetServer(ctx context.Context, serverID string) (*TaskID, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server ID is required")
	}

	path := buildServerPath(serverID, "power", "reset")
	req, err := c.newRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create reset request for server %s: %w", serverID, err)
	}

	var task TaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to reset server %s: %w", serverID, err)
	}

	return &task, nil
}

// ResetServerAndWait resets server and waits for it to become Active
func (c *CloudClient) ResetServerAndWait(ctx context.Context, serverID string) (*entities.Server, error) {
	task, err := c.ResetServer(ctx, serverID)
	if err != nil {
		return nil, err
	}

	// Wait for task completion and server to become Active
	return c.WaitServerTaskCompletion(ctx, serverID, task.ID)
}
