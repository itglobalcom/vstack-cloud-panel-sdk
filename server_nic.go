package sdk

import (
	"context"
	"fmt"
	"net/http"

	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

const (
	nicsPath = "nics"
)

// Response types for NIC operations
type (
	// GetNICResponse represents a single NIC response
	GetNICResponse struct {
		NIC *entities.NIC `json:"nic,omitempty"`
	}

	// ListNICsResponse represents a list of NICs response
	ListNICsResponse struct {
		NICs []entities.NIC `json:"nics,omitempty"`
	}
)

// buildServerNICPath constructs the path for server NIC operations
func buildServerNICPath(serverID string, nicID int) string {
	path := fmt.Sprintf("servers/%s/%s", serverID, nicsPath)
	if nicID > 0 {
		path = fmt.Sprintf("%s/%d", path, nicID)
	}
	return path
}

// GetServerNICs retrieves all network interfaces for a server
func (c *CloudClient) GetServerNICs(ctx context.Context, serverID string) ([]entities.NIC, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server ID is required")
	}

	path := buildServerNICPath(serverID, 0)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list NICs request for server %s: %w", serverID, err)
	}

	var resp ListNICsResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list NICs for server %s: %w", serverID, err)
	}

	return resp.NICs, nil
}

// GetServerNIC retrieves a specific network interface by ID
func (c *CloudClient) GetServerNIC(ctx context.Context, serverID string, nicID int) (*entities.NIC, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server ID is required")
	}
	if nicID <= 0 {
		return nil, fmt.Errorf("NIC ID must be greater than 0")
	}

	path := buildServerNICPath(serverID, nicID)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for NIC %d on server %s: %w", nicID, serverID, err)
	}

	var resp GetNICResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get NIC %d on server %s: %w", nicID, serverID, err)
	}

	if resp.NIC == nil {
		return nil, fmt.Errorf("NIC %d not found on server %s: %w", nicID, serverID, ErrNotFound)
	}

	return resp.NIC, nil
}

// CreateServerNIC creates a new network interface and returns a task ID
func (c *CloudClient) CreateServerNIC(ctx context.Context, serverID string, req *entities.CreateNICRequest) (*TaskID, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server ID is required")
	}
	if req == nil {
		return nil, fmt.Errorf("create NIC request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid create NIC request: %w", err)
	}

	path := buildServerNICPath(serverID, 0)
	httpReq, err := c.newRequest(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create NIC request for server %s: %w", serverID, err)
	}

	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to create NIC on server %s: %w", serverID, err)
	}

	return &task, nil
}

// CreateServerNICAndWait creates a network interface and waits for server to become Active
func (c *CloudClient) CreateServerNICAndWait(ctx context.Context, serverID string, req *entities.CreateNICRequest) (*entities.NIC, error) {
	// Get initial NIC IDs before creation
	initialNICs, err := c.GetServerNICs(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to get initial NICs: %w", err)
	}

	// Create map of existing NIC IDs
	initialIDMap := make(map[int]bool)
	for _, nic := range initialNICs {
		initialIDMap[nic.ID] = true
	}

	// Create NIC
	task, err := c.CreateServerNIC(ctx, serverID, req)
	if err != nil {
		return nil, err
	}

	// Wait for task completion
	completedTask, err := c.waitTaskCompletion(ctx, task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for NIC creation: %w", err)
	}

	// Wait for server to become Active
	if _, err := c.WaitServerActive(ctx, serverID); err != nil {
		return nil, fmt.Errorf("failed to wait for server to become Active: %w", err)
	}

	// Try to get NIC ID from task result first
	if completedTask.NicID != 0 {
		return c.GetServerNIC(ctx, serverID, completedTask.NicID)
	}

	// If NIC ID not in task result, find the new NIC by comparing lists
	currentNICs, err := c.GetServerNICs(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to get NICs after task completion: %w", err)
	}

	// Find new NICs that match the request criteria
	var matchedNICs []*entities.NIC
	for i := range currentNICs {
		nic := &currentNICs[i]

		// Skip if not a new NIC
		if initialIDMap[nic.ID] {
			continue
		}

		// Verify server_id matches
		if nic.ServerID != serverID {
			continue
		}

		// Match by request type
		if req.NetworkID != "" {
			// For isolated network: match by network_id
			if nic.NetworkID == req.NetworkID {
				// Additional check: if IP was specified, verify it matches
				if req.IPAddress == "" || nic.IPAddress == req.IPAddress {
					matchedNICs = append(matchedNICs, nic)
				}
			}
		} else if req.BandwidthMbps > 0 {
			// For public network: match by bandwidth
			if nic.BandwidthMbps == req.BandwidthMbps {
				matchedNICs = append(matchedNICs, nic)
			}
		} else {
			// No specific criteria, any new NIC matches
			matchedNICs = append(matchedNICs, nic)
		}
	}

	if len(matchedNICs) == 0 {
		return nil, fmt.Errorf("no new NIC found after task completion")
	}

	if len(matchedNICs) > 1 && c.logger != nil {
		c.logger.Warn(fmt.Sprintf("Found %d matching NICs, expected 1. Returning first one.", len(matchedNICs)))
	}

	return matchedNICs[0], nil
}

// UpdateServerNIC updates a network interface bandwidth and returns a task ID
func (c *CloudClient) UpdateServerNIC(ctx context.Context, serverID string, nicID int, req *entities.UpdateNICRequest) (*TaskID, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server ID is required")
	}
	if nicID <= 0 {
		return nil, fmt.Errorf("NIC ID must be greater than 0")
	}
	if req == nil {
		return nil, fmt.Errorf("update NIC request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid update NIC request: %w", err)
	}

	path := buildServerNICPath(serverID, nicID)
	httpReq, err := c.newRequest(ctx, http.MethodPut, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create update request for NIC %d on server %s: %w", nicID, serverID, err)
	}

	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to update NIC %d on server %s: %w", nicID, serverID, err)
	}

	return &task, nil
}

// UpdateServerNICAndWait updates a network interface and waits for completion
func (c *CloudClient) UpdateServerNICAndWait(ctx context.Context, serverID string, nicID int, req *entities.UpdateNICRequest) (*entities.NIC, error) {
	task, err := c.UpdateServerNIC(ctx, serverID, nicID, req)
	if err != nil {
		return nil, err
	}

	// Wait for task completion
	if _, err := c.waitTaskCompletion(ctx, task.ID); err != nil {
		return nil, fmt.Errorf("failed to wait for NIC update: %w", err)
	}

	// Get the updated NIC
	return c.GetServerNIC(ctx, serverID, nicID)
}

// DeleteServerNIC deletes a network interface
func (c *CloudClient) DeleteServerNIC(ctx context.Context, serverID string, nicID int) error {
	if serverID == "" {
		return fmt.Errorf("server ID is required")
	}
	if nicID <= 0 {
		return fmt.Errorf("NIC ID must be greater than 0")
	}

	path := buildServerNICPath(serverID, nicID)
	req, err := c.newRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request for NIC %d on server %s: %w", nicID, serverID, err)
	}

	if err := c.doJSON(req, nil); err != nil {
		return fmt.Errorf("failed to delete NIC %d on server %s: %w", nicID, serverID, err)
	}

	return nil
}

// DeleteServerNICAndWait deletes a network interface and waits for server to become Active
func (c *CloudClient) DeleteServerNICAndWait(ctx context.Context, serverID string, nicID int) error {
	err := c.DeleteServerNIC(ctx, serverID, nicID)
	if err != nil {
		return err
	}

	// Wait for task completion and server to become Active
	if _, err := c.WaitServerActive(ctx, serverID); err != nil {
		return fmt.Errorf("failed to wait for NIC deletion: %w", err)
	}

	return nil
}
