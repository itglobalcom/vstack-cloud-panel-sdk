package sdk

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

const (
	gatewaysBaseURL = "gateways"
)

// Response types
type (
	GetGatewayResponse struct {
		Gateway *entities.Gateway `json:"gateway,omitempty"`
	}

	ListGatewaysResponse struct {
		Gateways []*entities.Gateway `json:"gateways,omitempty"`
	}

	GetNATRulesResponse struct {
		NATRules []entities.NATRule `json:"nat_rules,omitempty"`
	}

	GetFirewallRulesResponse struct {
		FirewallRules []entities.FirewallRule `json:"firewall_rules,omitempty"`
	}
)

// buildGatewayPath constructs the path for gateway operations
func buildGatewayPath(gatewayID string, parts ...string) string {
	path := fmt.Sprintf("%s/%s", gatewaysBaseURL, gatewayID)
	for _, part := range parts {
		path = fmt.Sprintf("%s/%s", path, part)
	}
	return path
}

// GetGateway retrieves a specific gateway by ID
func (c *CloudClient) GetGateway(ctx context.Context, gatewayID string) (*entities.Gateway, error) {
	if gatewayID == "" {
		return nil, fmt.Errorf("gateway ID is required")
	}

	path := buildGatewayPath(gatewayID)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for gateway %s: %w", gatewayID, err)
	}

	var resp GetGatewayResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get gateway %s: %w", gatewayID, err)
	}

	if resp.Gateway == nil {
		return nil, fmt.Errorf("gateway %s not found in response: %w", gatewayID, ErrNotFound)
	}

	return resp.Gateway, nil
}

// GetGatewayList retrieves all gateways
func (c *CloudClient) GetGatewayList(ctx context.Context) ([]*entities.Gateway, error) {
	req, err := c.newRequest(ctx, http.MethodGet, gatewaysBaseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list gateways request: %w", err)
	}

	var resp ListGatewaysResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list gateways: %w", err)
	}

	return resp.Gateways, nil
}

// CreateGateway creates a new gateway and returns a task ID
func (c *CloudClient) CreateGateway(ctx context.Context, req *entities.CreateGatewayRequest) (*TaskID, error) {
	if req == nil {
		return nil, fmt.Errorf("create gateway request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid create gateway request: %w", err)
	}

	httpReq, err := c.newRequest(ctx, http.MethodPost, gatewaysBaseURL, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create gateway request: %w", err)
	}

	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to create gateway: %w", err)
	}

	return &task, nil
}

// CreateGatewayAndWait creates a gateway and waits for completion
func (c *CloudClient) CreateGatewayAndWait(ctx context.Context, req *entities.CreateGatewayRequest) (*entities.Gateway, error) {
	task, err := c.CreateGateway(ctx, req)
	if err != nil {
		return nil, err
	}

	// Wait for task completion
	completedTask, err := c.waitTaskCompletion(ctx, task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for gateway creation: %w", err)
	}

	// Get gateway ID from task result
	if completedTask.GatewayID == "" {
		return nil, fmt.Errorf("gateway ID not found in task result")
	}

	// Get the created gateway
	return c.GetGateway(ctx, completedTask.GatewayID)
}

// UpdateGateway updates gateway name
func (c *CloudClient) UpdateGateway(ctx context.Context, gatewayID string, req *entities.UpdateGatewayRequest) (*entities.Gateway, error) {
	if gatewayID == "" {
		return nil, fmt.Errorf("gateway ID is required")
	}
	if req == nil {
		return nil, fmt.Errorf("update gateway request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid update gateway request: %w", err)
	}

	path := buildGatewayPath(gatewayID)
	httpReq, err := c.newRequest(ctx, http.MethodPut, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create update request for gateway %s: %w", gatewayID, err)
	}

	var resp GetGatewayResponse
	if err := c.doJSON(httpReq, &resp); err != nil {
		return nil, fmt.Errorf("failed to update gateway %s: %w", gatewayID, err)
	}

	return resp.Gateway, nil
}

// DeleteGateway deletes a gateway
func (c *CloudClient) DeleteGateway(ctx context.Context, gatewayID string) error {
	if gatewayID == "" {
		return fmt.Errorf("gateway ID is required")
	}

	path := buildGatewayPath(gatewayID)
	req, err := c.newRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request for gateway %s: %w", gatewayID, err)
	}

	if err := c.doJSON(req, nil); err != nil {
		return fmt.Errorf("failed to delete gateway %s: %w", gatewayID, err)
	}

	return nil
}

// UpdateGatewayBandwidth updates gateway bandwidth and returns a task ID
func (c *CloudClient) UpdateGatewayBandwidth(ctx context.Context, gatewayID string, req *entities.UpdateGatewayBandwidthRequest) (*TaskID, error) {
	if gatewayID == "" {
		return nil, fmt.Errorf("gateway ID is required")
	}
	if req == nil {
		return nil, fmt.Errorf("update bandwidth request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid update bandwidth request: %w", err)
	}

	path := buildGatewayPath(gatewayID, "bandwidth")
	httpReq, err := c.newRequest(ctx, http.MethodPut, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create bandwidth update request for gateway %s: %w", gatewayID, err)
	}

	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to update gateway %s bandwidth: %w", gatewayID, err)
	}

	return &task, nil
}

// UpdateGatewayBandwidthAndWait updates gateway bandwidth and waits for completion
func (c *CloudClient) UpdateGatewayBandwidthAndWait(ctx context.Context, gatewayID string, req *entities.UpdateGatewayBandwidthRequest) (*entities.Gateway, error) {
	task, err := c.UpdateGatewayBandwidth(ctx, gatewayID, req)
	if err != nil {
		return nil, err
	}

	// Wait for task completion
	if _, err := c.waitTaskCompletion(ctx, task.ID); err != nil {
		return nil, fmt.Errorf("failed to wait for bandwidth update: %w", err)
	}

	// Get the updated gateway
	return c.GetGateway(ctx, gatewayID)
}

// Power management operations

// StopGateway stops the gateway and returns a task ID
func (c *CloudClient) StopGateway(ctx context.Context, gatewayID string) (*TaskID, error) {
	if gatewayID == "" {
		return nil, fmt.Errorf("gateway ID is required")
	}

	path := buildGatewayPath(gatewayID, "stop")
	req, err := c.newRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create stop request for gateway %s: %w", gatewayID, err)
	}

	var task TaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to stop gateway %s: %w", gatewayID, err)
	}

	return &task, nil
}

// StopGatewayAndWait stops gateway and waits for completion
func (c *CloudClient) StopGatewayAndWait(ctx context.Context, gatewayID string) (*entities.Gateway, error) {
	task, err := c.StopGateway(ctx, gatewayID)
	if err != nil {
		return nil, err
	}

	// Wait for task completion
	if _, err := c.waitTaskCompletion(ctx, task.ID); err != nil {
		return nil, fmt.Errorf("failed to wait for gateway stop: %w", err)
	}

	return c.GetGateway(ctx, gatewayID)
}

// StartGateway starts the gateway and returns a task ID
func (c *CloudClient) StartGateway(ctx context.Context, gatewayID string) (*TaskID, error) {
	if gatewayID == "" {
		return nil, fmt.Errorf("gateway ID is required")
	}

	path := buildGatewayPath(gatewayID, "start")
	req, err := c.newRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create start request for gateway %s: %w", gatewayID, err)
	}

	var task TaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to start gateway %s: %w", gatewayID, err)
	}

	return &task, nil
}

// StartGatewayAndWait starts gateway and waits for completion
func (c *CloudClient) StartGatewayAndWait(ctx context.Context, gatewayID string) (*entities.Gateway, error) {
	task, err := c.StartGateway(ctx, gatewayID)
	if err != nil {
		return nil, err
	}

	// Wait for task completion
	if _, err := c.waitTaskCompletion(ctx, task.ID); err != nil {
		return nil, fmt.Errorf("failed to wait for gateway start: %w", err)
	}

	return c.GetGateway(ctx, gatewayID)
}

// RestartGateway restarts the gateway and returns a task ID
func (c *CloudClient) RestartGateway(ctx context.Context, gatewayID string) (*TaskID, error) {
	if gatewayID == "" {
		return nil, fmt.Errorf("gateway ID is required")
	}

	path := buildGatewayPath(gatewayID, "restart")
	req, err := c.newRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create restart request for gateway %s: %w", gatewayID, err)
	}

	var task TaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to restart gateway %s: %w", gatewayID, err)
	}

	return &task, nil
}

// RestartGatewayAndWait restarts gateway and waits for completion
func (c *CloudClient) RestartGatewayAndWait(ctx context.Context, gatewayID string) (*entities.Gateway, error) {
	task, err := c.RestartGateway(ctx, gatewayID)
	if err != nil {
		return nil, err
	}

	// Wait for task completion
	if _, err := c.waitTaskCompletion(ctx, task.ID); err != nil {
		return nil, fmt.Errorf("failed to wait for gateway restart: %w", err)
	}

	return c.GetGateway(ctx, gatewayID)
}

// NAT rules operations

// GetNATRules retrieves NAT rules for a gateway
func (c *CloudClient) GetNATRules(ctx context.Context, gatewayID string) ([]entities.NATRule, error) {
	if gatewayID == "" {
		return nil, fmt.Errorf("gateway ID is required")
	}

	path := buildGatewayPath(gatewayID, "nat")
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get NAT rules request for gateway %s: %w", gatewayID, err)
	}

	var resp GetNATRulesResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get NAT rules for gateway %s: %w", gatewayID, err)
	}

	return resp.NATRules, nil
}

// UpdateNATRules updates NAT rules for a gateway and returns a task ID
func (c *CloudClient) UpdateNATRules(ctx context.Context, gatewayID string, req *entities.UpdateNATRulesRequest) (*TaskID, error) {
	if gatewayID == "" {
		return nil, fmt.Errorf("gateway ID is required")
	}
	if req == nil {
		return nil, fmt.Errorf("update NAT rules request is required")
	}

	path := buildGatewayPath(gatewayID, "nat")
	httpReq, err := c.newRequest(ctx, http.MethodPut, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create update NAT rules request for gateway %s: %w", gatewayID, err)
	}

	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to update NAT rules for gateway %s: %w", gatewayID, err)
	}

	return &task, nil
}

// UpdateNATRulesAndWait updates NAT rules and waits for completion
func (c *CloudClient) UpdateNATRulesAndWait(ctx context.Context, gatewayID string, req *entities.UpdateNATRulesRequest) error {
	task, err := c.UpdateNATRules(ctx, gatewayID, req)
	if err != nil {
		return err
	}

	// Wait for task completion
	_, err = c.waitTaskCompletion(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("failed to wait for NAT rules update: %w", err)
	}

	return nil
}

// Firewall rules operations

// GetFirewallRules retrieves firewall rules for a gateway
func (c *CloudClient) GetFirewallRules(ctx context.Context, gatewayID string) ([]entities.FirewallRule, error) {
	if gatewayID == "" {
		return nil, fmt.Errorf("gateway ID is required")
	}

	path := buildGatewayPath(gatewayID, "firewall")
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get firewall rules request for gateway %s: %w", gatewayID, err)
	}

	var resp GetFirewallRulesResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get firewall rules for gateway %s: %w", gatewayID, err)
	}

	return resp.FirewallRules, nil
}

// UpdateFirewallRules updates firewall rules for a gateway and returns a task ID
func (c *CloudClient) UpdateFirewallRules(ctx context.Context, gatewayID string, req *entities.UpdateFirewallRulesRequest) (*TaskID, error) {
	if gatewayID == "" {
		return nil, fmt.Errorf("gateway ID is required")
	}
	if req == nil {
		return nil, fmt.Errorf("update firewall rules request is required")
	}

	path := buildGatewayPath(gatewayID, "firewall")
	httpReq, err := c.newRequest(ctx, http.MethodPut, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create update firewall rules request for gateway %s: %w", gatewayID, err)
	}

	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to update firewall rules for gateway %s: %w", gatewayID, err)
	}

	return &task, nil
}

// UpdateFirewallRulesAndWait updates firewall rules and waits for completion
func (c *CloudClient) UpdateFirewallRulesAndWait(ctx context.Context, gatewayID string, req *entities.UpdateFirewallRulesRequest) error {
	task, err := c.UpdateFirewallRules(ctx, gatewayID, req)
	if err != nil {
		return err
	}

	// Wait for task completion
	_, err = c.waitTaskCompletion(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("failed to wait for firewall rules update: %w", err)
	}

	return nil
}

// Network operations

// ConnectNetwork connects an isolated network to gateway and returns a task ID
func (c *CloudClient) ConnectNetwork(ctx context.Context, gatewayID string, req *entities.ConnectNetworkRequest) (*TaskID, error) {
	if gatewayID == "" {
		return nil, fmt.Errorf("gateway ID is required")
	}
	if req == nil {
		return nil, fmt.Errorf("connect network request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid connect network request: %w", err)
	}

	path := buildGatewayPath(gatewayID, "nics")
	httpReq, err := c.newRequest(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create connect network request for gateway %s: %w", gatewayID, err)
	}

	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to connect network to gateway %s: %w", gatewayID, err)
	}

	return &task, nil
}

// ConnectNetworkAndWait connects network and waits for completion
func (c *CloudClient) ConnectNetworkAndWait(ctx context.Context, gatewayID string, req *entities.ConnectNetworkRequest) error {
	task, err := c.ConnectNetwork(ctx, gatewayID, req)
	if err != nil {
		return err
	}

	// Wait for task completion
	_, err = c.waitTaskCompletion(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("failed to wait for network connection: %w", err)
	}

	return nil
}

// DisconnectNetwork disconnects an isolated network from gateway
func (c *CloudClient) DisconnectNetwork(ctx context.Context, gatewayID string, nicID int) error {
	if gatewayID == "" {
		return fmt.Errorf("gateway ID is required")
	}
	if nicID <= 0 {
		return fmt.Errorf("NIC ID must be greater than 0")
	}

	path := buildGatewayPath(gatewayID, "nics", fmt.Sprintf("%d", nicID))
	req, err := c.newRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("failed to create disconnect network request for gateway %s: %w", gatewayID, err)
	}

	if err := c.doJSON(req, nil); err != nil {
		return fmt.Errorf("failed to disconnect network from gateway %s: %w", gatewayID, err)
	}

	return nil
}

// DisconnectNetworkAndWait disconnects an isolated network from gateway and waits until the NIC is removed
func (c *CloudClient) DisconnectNetworkAndWait(ctx context.Context, gatewayID string, nicID int) error {
	if gatewayID == "" {
		return fmt.Errorf("gateway ID is required")
	}
	if nicID <= 0 {
		return fmt.Errorf("NIC ID must be greater than 0")
	}

	path := buildGatewayPath(gatewayID, "nics", fmt.Sprintf("%d", nicID))
	req, err := c.newRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("failed to create disconnect network request for gateway %s: %w", gatewayID, err)
	}

	if err := c.doJSON(req, nil); err != nil {
		return fmt.Errorf("failed to disconnect network from gateway %s: %w", gatewayID, err)
	}

	// Wait until the NIC disappears from the list
	return c.waitForNICDeletion(ctx, gatewayID, nicID)
}

// waitForNICDeletion polls the gateway until the specified NIC is no longer present
func (c *CloudClient) waitForNICDeletion(ctx context.Context, gatewayID string, nicID int) error {
	// Polling settings
	initialInterval := 2 * time.Second
	maxInterval := 10 * time.Second
	maxWaitTime := 2 * time.Minute

	ctx, cancel := context.WithTimeout(ctx, maxWaitTime)
	defer cancel()

	interval := initialInterval

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for NIC %d deletion from gateway %s: %w", nicID, gatewayID, ctx.Err())
		case <-time.After(interval):
			// Get current gateway state
			gateway, err := c.GetGateway(ctx, gatewayID)
			if err != nil {
				return fmt.Errorf("failed to check gateway state while waiting for NIC deletion: %w", err)
			}

			// Check if NIC is still in the list
			nicExists := false
			for _, nic := range gateway.NICs {
				if nic.ID == nicID {
					nicExists = true
					break
				}
			}

			// If NIC is no longer present - operation completed
			if !nicExists {
				return nil
			}

			// Exponential backoff with maximum interval
			interval = time.Duration(float64(interval) * 1.5)
			if interval > maxInterval {
				interval = maxInterval
			}
		}
	}
}

// Tag operations

// CreateGatewayTag creates a tag for gateway
func (c *CloudClient) CreateGatewayTag(ctx context.Context, gatewayID string, req *entities.CreateGatewayTagRequest) error {
	if gatewayID == "" {
		return fmt.Errorf("gateway ID is required")
	}
	if req == nil {
		return fmt.Errorf("create tag request is required")
	}
	if err := req.Validate(); err != nil {
		return fmt.Errorf("invalid create tag request: %w", err)
	}

	path := buildGatewayPath(gatewayID, "tags")
	httpReq, err := c.newRequest(ctx, http.MethodPost, path, req)
	if err != nil {
		return fmt.Errorf("failed to create tag request for gateway %s: %w", gatewayID, err)
	}

	if err := c.doJSON(httpReq, nil); err != nil {
		return fmt.Errorf("failed to create tag for gateway %s: %w", gatewayID, err)
	}

	return nil
}

// DeleteGatewayTag deletes a tag from gateway
func (c *CloudClient) DeleteGatewayTag(ctx context.Context, gatewayID, tag string) error {
	if gatewayID == "" {
		return fmt.Errorf("gateway ID is required")
	}
	if tag == "" {
		return fmt.Errorf("tag is required")
	}

	path := buildGatewayPath(gatewayID, "tags", tag)
	req, err := c.newRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete tag request for gateway %s: %w", gatewayID, err)
	}

	if err := c.doJSON(req, nil); err != nil {
		return fmt.Errorf("failed to delete tag from gateway %s: %w", gatewayID, err)
	}

	return nil
}
