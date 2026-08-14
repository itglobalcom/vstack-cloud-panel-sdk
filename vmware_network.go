package sdk

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

const (
	vmwareNetworksBaseURL = "vmware/networks"
)

// VMware network and edge response wrappers.
type (
	vmwareNetworkResponse struct {
		Network *entities.VmwareNetwork `json:"network,omitempty"`
	}
	vmwareNetworksResponse struct {
		Networks []*entities.VmwareNetwork `json:"networks,omitempty"`
	}
	vmwareTaskRefsResponse struct {
		TaskIDs []string `json:"task_ids,omitempty"`
	}
)

// buildVmwareNetworkPath constructs the path for VMware network operations.
func buildVmwareNetworkPath(networkID int, parts ...string) string {
	path := fmt.Sprintf("%s/%d", vmwareNetworksBaseURL, networkID)
	for _, p := range parts {
		path = fmt.Sprintf("%s/%s", path, p)
	}
	return path
}

// buildVmwareEdgePath constructs the path for VMware edge gateway operations.
func buildVmwareEdgePath(networkID int, parts ...string) string {
	return buildVmwareNetworkPath(networkID, append([]string{"edge"}, parts...)...)
}

// ===================== Networks =====================

// GetVmwareNetworkList retrieves all VMware networks, optionally filtered by
// location.
//
// locationID is omitted from the query string when nil rather than sent empty —
// an empty value of a declared query parameter makes the API answer HTTP 500.
func (c *CloudClient) GetVmwareNetworkList(ctx context.Context, locationID *int) ([]*entities.VmwareNetwork, error) {
	if locationID != nil && *locationID <= 0 {
		return nil, fmt.Errorf("location ID must be positive")
	}
	params := url.Values{}
	if locationID != nil {
		params.Set("location_id", strconv.Itoa(*locationID))
	}
	req, err := c.newRequest(ctx, http.MethodGet, withQuery(vmwareNetworksBaseURL, params), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list vmware networks request: %w", err)
	}
	var resp vmwareNetworksResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list vmware networks: %w", err)
	}
	return resp.Networks, nil
}

// GetVmwareNetwork retrieves a specific VMware network by ID.
func (c *CloudClient) GetVmwareNetwork(ctx context.Context, networkID int) (*entities.VmwareNetwork, error) {
	if networkID <= 0 {
		return nil, fmt.Errorf("network ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodGet, buildVmwareNetworkPath(networkID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get vmware network %d request: %w", networkID, err)
	}
	var resp vmwareNetworkResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get vmware network %d: %w", networkID, err)
	}
	if resp.Network == nil {
		return nil, fmt.Errorf("vmware network %d not found in response: %w", networkID, ErrNotFound)
	}
	return resp.Network, nil
}

// CreateVmwareIsolatedNetwork creates an isolated VMware network and returns the
// background task to await.
func (c *CloudClient) CreateVmwareIsolatedNetwork(ctx context.Context, req *entities.VmwareCreateIsolatedNetworkRequest) (*VmwareTaskID, error) {
	if req == nil {
		return nil, fmt.Errorf("create isolated network request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return c.createVmwareNetwork(ctx, "isolated", req)
}

// CreateVmwareRoutedNetwork creates a routed VMware network and returns the
// background task to await. A routed network carries an edge gateway, so the
// firewall / NAT / VPN methods below apply to it.
func (c *CloudClient) CreateVmwareRoutedNetwork(ctx context.Context, req *entities.VmwareCreateRoutedNetworkRequest) (*VmwareTaskID, error) {
	if req == nil {
		return nil, fmt.Errorf("create routed network request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return c.createVmwareNetwork(ctx, "routed", req)
}

// CreateVmwarePublicNetwork creates a public VMware network and returns the
// background task to await.
func (c *CloudClient) CreateVmwarePublicNetwork(ctx context.Context, req *entities.VmwareCreatePublicNetworkRequest) (*VmwareTaskID, error) {
	if req == nil {
		return nil, fmt.Errorf("create public network request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return c.createVmwareNetwork(ctx, "public", req)
}

// createVmwareNetwork posts a create-network request of the given kind and
// returns the background task.
func (c *CloudClient) createVmwareNetwork(ctx context.Context, kind string, body interface{}) (*VmwareTaskID, error) {
	req, err := c.newRequest(ctx, http.MethodPost, fmt.Sprintf("%s/%s", vmwareNetworksBaseURL, kind), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s vmware network request: %w", kind, err)
	}
	var task VmwareTaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to create %s vmware network: %w", kind, err)
	}
	return &task, nil
}

// CreateVmwareIsolatedNetworkAndWait creates an isolated VMware network, waits for
// the background task to finish and returns the created network.
func (c *CloudClient) CreateVmwareIsolatedNetworkAndWait(ctx context.Context, req *entities.VmwareCreateIsolatedNetworkRequest) (*entities.VmwareNetwork, error) {
	task, err := c.CreateVmwareIsolatedNetwork(ctx, req)
	if err != nil {
		return nil, err
	}
	return c.waitVmwareNetworkCreated(ctx, task)
}

// CreateVmwareRoutedNetworkAndWait creates a routed VMware network, waits for the
// background task to finish and returns the created network.
func (c *CloudClient) CreateVmwareRoutedNetworkAndWait(ctx context.Context, req *entities.VmwareCreateRoutedNetworkRequest) (*entities.VmwareNetwork, error) {
	task, err := c.CreateVmwareRoutedNetwork(ctx, req)
	if err != nil {
		return nil, err
	}
	return c.waitVmwareNetworkCreated(ctx, task)
}

// CreateVmwarePublicNetworkAndWait creates a public VMware network, waits for the
// background task to finish and returns the created network.
func (c *CloudClient) CreateVmwarePublicNetworkAndWait(ctx context.Context, req *entities.VmwareCreatePublicNetworkRequest) (*entities.VmwareNetwork, error) {
	task, err := c.CreateVmwarePublicNetwork(ctx, req)
	if err != nil {
		return nil, err
	}
	return c.waitVmwareNetworkCreated(ctx, task)
}

// waitVmwareNetworkCreated waits for a create-network task to complete and
// fetches the resulting network by the network_id carried in the completed task.
func (c *CloudClient) waitVmwareNetworkCreated(ctx context.Context, task *VmwareTaskID) (*entities.VmwareNetwork, error) {
	if task.IsZero() {
		return nil, fmt.Errorf("create vmware network returned no task to wait for")
	}
	completed, err := c.WaitVmwareTaskRef(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for vmware network creation: %w", err)
	}
	if completed == nil || completed.NetworkID == nil {
		return nil, fmt.Errorf("network ID not found in vmware task %s result", task.String())
	}
	return c.GetVmwareNetwork(ctx, *completed.NetworkID)
}

// EditVmwareNetwork updates the name and/or bandwidth of a VMware network.
//
// This is also the only way to set the bandwidth of a routed network's
// edge — edge bandwidth and network bandwidth are one field.
//
// Bandwidth does not apply to an isolated network — send only Name.
//
// An isolated-network edit is answered synchronously, with HTTP 204 and no task,
// and this method returns (nil, nil) in that case. A routed or public network
// answers 200 with a task to await. The two cases are distinguishable by the
// result: a nil task means the change is already applied.
func (c *CloudClient) EditVmwareNetwork(ctx context.Context, networkID int, req *entities.VmwareEditNetworkRequest) (*VmwareTaskID, error) {
	if networkID <= 0 {
		return nil, fmt.Errorf("network ID must be greater than 0")
	}
	if req == nil {
		return nil, fmt.Errorf("edit network request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid edit network request: %w", err)
	}
	httpReq, err := c.newRequest(ctx, http.MethodPut, buildVmwareNetworkPath(networkID), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create edit vmware network %d request: %w", networkID, err)
	}
	var task VmwareTaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to edit vmware network %d: %w", networkID, err)
	}
	// An empty task_id means the edit completed synchronously;
	// return (nil, nil) so callers do not await a meaningless empty task.
	if task.ID == "" {
		return nil, nil
	}
	return &task, nil
}

// EditVmwareNetworkAndWait edits a network, waits for the task (if any) and
// returns the refreshed network.
func (c *CloudClient) EditVmwareNetworkAndWait(ctx context.Context, networkID int, req *entities.VmwareEditNetworkRequest) (*entities.VmwareNetwork, error) {
	task, err := c.EditVmwareNetwork(ctx, networkID, req)
	if err != nil {
		return nil, err
	}
	// A nil task is the documented synchronous case (isolated network).
	if err := c.awaitVmwareTask(ctx, task); err != nil {
		return nil, err
	}
	return c.GetVmwareNetwork(ctx, networkID)
}

// DeleteVmwareNetwork deletes a VMware network and returns the background task to
// await.
//
// A network with servers or gateways attached cannot be deleted; that is reported
// as APICodeNetworkInUse (see IsNetworkInUse).
func (c *CloudClient) DeleteVmwareNetwork(ctx context.Context, networkID int) (*VmwareTaskID, error) {
	if networkID <= 0 {
		return nil, fmt.Errorf("network ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodDelete, buildVmwareNetworkPath(networkID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create delete vmware network %d request: %w", networkID, err)
	}
	var task VmwareTaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to delete vmware network %d: %w", networkID, err)
	}
	return &task, nil
}

// DeleteVmwareNetworkAndWait deletes a network and waits until it is really gone
// — the task can complete while the object is still readable.
func (c *CloudClient) DeleteVmwareNetworkAndWait(ctx context.Context, networkID int) error {
	task, err := c.DeleteVmwareNetwork(ctx, networkID)
	if err != nil {
		return err
	}
	if err := c.awaitVmwareTask(ctx, task); err != nil {
		return err
	}
	return c.WaitVmwareNetworkGone(ctx, networkID)
}

// ConnectVmwareServers connects a set of servers to a network and returns one
// task per server, in the order the request listed them.
//
// The tasks come back as typed references rather than bare strings.
// A partial failure is reported through the error's ErrorParams, which name the
// offending element (see RequestError.ErrorParams).
func (c *CloudClient) ConnectVmwareServers(ctx context.Context, networkID int, req *entities.VmwareConnectServersRequest) ([]*VmwareTaskID, error) {
	if networkID <= 0 {
		return nil, fmt.Errorf("network ID must be greater than 0")
	}
	if req == nil {
		return nil, fmt.Errorf("connect servers request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	httpReq, err := c.newRequest(ctx, http.MethodPost, buildVmwareNetworkPath(networkID, "servers"), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create connect servers to vmware network %d request: %w", networkID, err)
	}
	var resp vmwareTaskRefsResponse
	if err := c.doJSON(httpReq, &resp); err != nil {
		return nil, fmt.Errorf("failed to connect servers to vmware network %d: %w", networkID, err)
	}
	tasks := make([]*VmwareTaskID, 0, len(resp.TaskIDs))
	for _, id := range resp.TaskIDs {
		tasks = append(tasks, &VmwareTaskID{ID: id})
	}
	return tasks, nil
}

// ConnectVmwareServersAndWait connects servers to a network and waits for every
// task it started. The tasks are awaited in order.
func (c *CloudClient) ConnectVmwareServersAndWait(ctx context.Context, networkID int, req *entities.VmwareConnectServersRequest) error {
	tasks, err := c.ConnectVmwareServers(ctx, networkID, req)
	if err != nil {
		return err
	}
	for _, task := range tasks {
		if err := c.awaitVmwareTask(ctx, task); err != nil {
			return err
		}
	}
	return nil
}

// ===================== Edge: Firewall =====================

// GetVmwareEdgeFirewall retrieves the edge firewall configuration of a VMware
// network. Only a routed network has an edge.
func (c *CloudClient) GetVmwareEdgeFirewall(ctx context.Context, networkID int) (*entities.VmwareEdgeFirewall, error) {
	if networkID <= 0 {
		return nil, fmt.Errorf("network ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodGet, buildVmwareEdgePath(networkID, "firewall"), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get edge firewall of vmware network %d request: %w", networkID, err)
	}
	// The edge firewall response is wrapped as {"firewall": {...}},
	// aligning with the single-key envelope used across the API.
	var resp struct {
		Firewall *entities.VmwareEdgeFirewall `json:"firewall,omitempty"`
	}
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get edge firewall of vmware network %d: %w", networkID, err)
	}
	if resp.Firewall == nil {
		return nil, fmt.Errorf("edge firewall of vmware network %d not found in response: %w", networkID, ErrNotFound)
	}
	return resp.Firewall, nil
}

// UpdateVmwareEdgeFirewall atomically replaces the edge firewall rule set.
//
// Omitting Enabled or DefaultAction now leaves the current value
// alone, so a read-modify-write no longer risks switching the firewall off. Rules
// keeps set semantics — resend the rules you want to keep.
func (c *CloudClient) UpdateVmwareEdgeFirewall(ctx context.Context, networkID int, req *entities.VmwareUpdateEdgeFirewallRequest) (*VmwareTaskID, error) {
	if networkID <= 0 {
		return nil, fmt.Errorf("network ID must be greater than 0")
	}
	if req == nil {
		return nil, fmt.Errorf("update edge firewall request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	httpReq, err := c.newRequest(ctx, http.MethodPut, buildVmwareEdgePath(networkID, "firewall"), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create update edge firewall of vmware network %d request: %w", networkID, err)
	}
	var task VmwareTaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to update edge firewall of vmware network %d: %w", networkID, err)
	}
	if task.ID == "" {
		return nil, nil
	}
	return &task, nil
}

// UpdateVmwareEdgeFirewallAndWait replaces the edge firewall rule set, waits for
// the task and returns the resulting configuration.
func (c *CloudClient) UpdateVmwareEdgeFirewallAndWait(ctx context.Context, networkID int, req *entities.VmwareUpdateEdgeFirewallRequest) (*entities.VmwareEdgeFirewall, error) {
	task, err := c.UpdateVmwareEdgeFirewall(ctx, networkID, req)
	if err != nil {
		return nil, err
	}
	if err := c.awaitVmwareTask(ctx, task); err != nil {
		return nil, err
	}
	return c.GetVmwareEdgeFirewall(ctx, networkID)
}

// ===================== Edge: NAT =====================

// GetVmwareEdgeNAT retrieves the edge NAT configuration of a VMware network.
func (c *CloudClient) GetVmwareEdgeNAT(ctx context.Context, networkID int) (*entities.VmwareEdgeNAT, error) {
	if networkID <= 0 {
		return nil, fmt.Errorf("network ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodGet, buildVmwareEdgePath(networkID, "nat"), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get edge nat of vmware network %d request: %w", networkID, err)
	}
	// Decoded WITHOUT an envelope on purpose - this endpoint returns a flat
	// {"rules": [...]}, unlike the firewall and VPN ones.
	var nat entities.VmwareEdgeNAT
	if err := c.doJSON(req, &nat); err != nil {
		return nil, fmt.Errorf("failed to get edge nat of vmware network %d: %w", networkID, err)
	}
	return &nat, nil
}

// UpsertVmwareEdgeNATRule creates or updates a NAT rule — update when
// req.RuleID is set, create otherwise.
func (c *CloudClient) UpsertVmwareEdgeNATRule(ctx context.Context, networkID int, req *entities.VmwareUpsertNATRuleRequest) (*VmwareTaskID, error) {
	if networkID <= 0 {
		return nil, fmt.Errorf("network ID must be greater than 0")
	}
	if req == nil {
		return nil, fmt.Errorf("upsert nat rule request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	httpReq, err := c.newRequest(ctx, http.MethodPost, buildVmwareEdgePath(networkID, "nat"), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create upsert edge nat rule of vmware network %d request: %w", networkID, err)
	}
	var task VmwareTaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to upsert edge nat rule of vmware network %d: %w", networkID, err)
	}
	return &task, nil
}

// UpsertVmwareEdgeNATRuleAndWait upserts a NAT rule, waits for the task and
// returns the resulting NAT configuration.
func (c *CloudClient) UpsertVmwareEdgeNATRuleAndWait(ctx context.Context, networkID int, req *entities.VmwareUpsertNATRuleRequest) (*entities.VmwareEdgeNAT, error) {
	task, err := c.UpsertVmwareEdgeNATRule(ctx, networkID, req)
	if err != nil {
		return nil, err
	}
	if err := c.awaitVmwareTask(ctx, task); err != nil {
		return nil, err
	}
	return c.GetVmwareEdgeNAT(ctx, networkID)
}

// DeleteVmwareEdgeNATRule deletes a NAT rule and returns the background task to
// await.
//
// RuleID is VmwareEdgeNATRule.ID as returned by GetVmwareEdgeNAT —
// the internal DB id this endpoint accepts.
func (c *CloudClient) DeleteVmwareEdgeNATRule(ctx context.Context, networkID, ruleID int) (*VmwareTaskID, error) {
	if networkID <= 0 {
		return nil, fmt.Errorf("network ID must be greater than 0")
	}
	if ruleID <= 0 {
		return nil, fmt.Errorf("rule ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodDelete, buildVmwareEdgePath(networkID, "nat", strconv.Itoa(ruleID)), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create delete edge nat rule %d of vmware network %d request: %w", ruleID, networkID, err)
	}
	var task VmwareTaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to delete edge nat rule %d of vmware network %d: %w", ruleID, networkID, err)
	}
	return &task, nil
}

// DeleteVmwareEdgeNATRuleAndWait deletes a NAT rule and waits for its task.
func (c *CloudClient) DeleteVmwareEdgeNATRuleAndWait(ctx context.Context, networkID, ruleID int) error {
	task, err := c.DeleteVmwareEdgeNATRule(ctx, networkID, ruleID)
	if err != nil {
		return err
	}
	return c.awaitVmwareTask(ctx, task)
}

// ===================== Edge: VPN =====================

// GetVmwareEdgeVPN retrieves the edge VPN configuration of a VMware network.
func (c *CloudClient) GetVmwareEdgeVPN(ctx context.Context, networkID int) (*entities.VmwareEdgeVPN, error) {
	if networkID <= 0 {
		return nil, fmt.Errorf("network ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodGet, buildVmwareEdgePath(networkID, "vpn"), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get edge vpn of vmware network %d request: %w", networkID, err)
	}
	// The edge VPN response is wrapped as {"vpn": {...}}.
	var resp struct {
		VPN *entities.VmwareEdgeVPN `json:"vpn,omitempty"`
	}
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get edge vpn of vmware network %d: %w", networkID, err)
	}
	if resp.VPN == nil {
		return nil, fmt.Errorf("edge vpn of vmware network %d not found in response: %w", networkID, ErrNotFound)
	}
	return resp.VPN, nil
}

// UpsertVmwareEdgeVPNTunnel creates or updates a VPN tunnel — update when
// req.TunnelID is set, create otherwise.
//
// MTU, DiffieHellmanGroup and EncryptionType are mandatory despite their
// optional-looking tags; Validate enforces them, along with the pre-shared key
// rules, before the request is sent.
func (c *CloudClient) UpsertVmwareEdgeVPNTunnel(ctx context.Context, networkID int, req *entities.VmwareUpsertVPNTunnelRequest) (*VmwareTaskID, error) {
	if networkID <= 0 {
		return nil, fmt.Errorf("network ID must be greater than 0")
	}
	if req == nil {
		return nil, fmt.Errorf("upsert vpn tunnel request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	httpReq, err := c.newRequest(ctx, http.MethodPost, buildVmwareEdgePath(networkID, "vpn"), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create upsert edge vpn tunnel of vmware network %d request: %w", networkID, err)
	}
	var task VmwareTaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to upsert edge vpn tunnel of vmware network %d: %w", networkID, err)
	}
	return &task, nil
}

// UpsertVmwareEdgeVPNTunnelAndWait upserts a VPN tunnel, waits for the task and
// returns the resulting VPN configuration.
func (c *CloudClient) UpsertVmwareEdgeVPNTunnelAndWait(ctx context.Context, networkID int, req *entities.VmwareUpsertVPNTunnelRequest) (*entities.VmwareEdgeVPN, error) {
	task, err := c.UpsertVmwareEdgeVPNTunnel(ctx, networkID, req)
	if err != nil {
		return nil, err
	}
	if err := c.awaitVmwareTask(ctx, task); err != nil {
		return nil, err
	}
	return c.GetVmwareEdgeVPN(ctx, networkID)
}

// DeleteVmwareEdgeVPNTunnel deletes a VPN tunnel and returns the background task
// to await.
//
// TunnelID is VmwareEdgeVPNTunnel.ID as returned by
// GetVmwareEdgeVPN — the internal DB id this endpoint accepts.
func (c *CloudClient) DeleteVmwareEdgeVPNTunnel(ctx context.Context, networkID, tunnelID int) (*VmwareTaskID, error) {
	if networkID <= 0 {
		return nil, fmt.Errorf("network ID must be greater than 0")
	}
	if tunnelID <= 0 {
		return nil, fmt.Errorf("tunnel ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodDelete, buildVmwareEdgePath(networkID, "vpn", strconv.Itoa(tunnelID)), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create delete edge vpn tunnel %d of vmware network %d request: %w", tunnelID, networkID, err)
	}
	var task VmwareTaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to delete edge vpn tunnel %d of vmware network %d: %w", tunnelID, networkID, err)
	}
	return &task, nil
}

// DeleteVmwareEdgeVPNTunnelAndWait deletes a VPN tunnel and waits for its task.
func (c *CloudClient) DeleteVmwareEdgeVPNTunnelAndWait(ctx context.Context, networkID, tunnelID int) error {
	task, err := c.DeleteVmwareEdgeVPNTunnel(ctx, networkID, tunnelID)
	if err != nil {
		return err
	}
	return c.awaitVmwareTask(ctx, task)
}

// ===================== Edge: Bandwidth =====================
//
// UpdateVmwareEdgeBandwidth was removed. PUT /edge/bandwidth reported
// success and ran its task to completion without ever persisting the value, so the
// API kept serving the old bandwidth and the next network edit silently reverted
// the real setting; it also bypassed the bandwidth policy check and the NSX-T path,
// and the value could not be read back (GET answered 405).
//
// Bandwidth is one field shared by the network and its edge: set it with
// EditVmwareNetwork and read it from VmwareNetwork.BandwidthMbps.
