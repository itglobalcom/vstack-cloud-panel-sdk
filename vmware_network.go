package sdk

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

// C-12: path constants declared as a const block instead of a single line.
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
		TaskIds []string `json:"task_ids,omitempty"`
	}
)

// buildVmwareNetworkPath constructs the path for VMware network operations.
//
// C-12: vmwareNetworkPath -> buildVmwareNetworkPath.
func buildVmwareNetworkPath(networkID int, parts ...string) string {
	path := fmt.Sprintf("%s/%d", vmwareNetworksBaseURL, networkID)
	for _, p := range parts {
		path = fmt.Sprintf("%s/%s", path, p)
	}
	return path
}

// buildVmwareEdgePath constructs the path for VMware edge gateway operations.
//
// C-12: vmwareEdgePath -> buildVmwareEdgePath.
func buildVmwareEdgePath(networkID int, parts ...string) string {
	return buildVmwareNetworkPath(networkID, append([]string{"edge"}, parts...)...)
}

// ===================== Networks =====================

// GetVmwareNetworkList retrieves all VMware networks, optionally filtered by location.
func (c *CloudClient) GetVmwareNetworkList(ctx context.Context, locationID *int) ([]*entities.VmwareNetwork, error) {
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
	// N4/C-5: validate the id before issuing the request.
	if networkID <= 0 {
		return nil, fmt.Errorf("network ID must be greater than 0")
	}
	// C-7: wrap the newRequest error with context, consistent with the rest of the file.
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

// CreateVmwareIsolatedNetwork creates an isolated VMware network and returns a task ID.
func (c *CloudClient) CreateVmwareIsolatedNetwork(ctx context.Context, req *entities.VmwareCreateIsolatedNetworkRequest) (*TaskID, error) {
	if req == nil {
		return nil, fmt.Errorf("create isolated network request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return c.createVmwareNetwork(ctx, "isolated", req)
}

// CreateVmwareRoutedNetwork creates a routed VMware network and returns a task ID.
func (c *CloudClient) CreateVmwareRoutedNetwork(ctx context.Context, req *entities.VmwareCreateRoutedNetworkRequest) (*TaskID, error) {
	if req == nil {
		return nil, fmt.Errorf("create routed network request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return c.createVmwareNetwork(ctx, "routed", req)
}

// CreateVmwarePublicNetwork creates a public VMware network and returns a task ID.
func (c *CloudClient) CreateVmwarePublicNetwork(ctx context.Context, req *entities.VmwareCreatePublicNetworkRequest) (*TaskID, error) {
	if req == nil {
		return nil, fmt.Errorf("create public network request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return c.createVmwareNetwork(ctx, "public", req)
}

// createVmwareNetwork posts a create-network request of the given kind and returns a task ID.
func (c *CloudClient) createVmwareNetwork(ctx context.Context, kind string, body interface{}) (*TaskID, error) {
	// C-7: wrap the newRequest error with context.
	req, err := c.newRequest(ctx, http.MethodPost, fmt.Sprintf("%s/%s", vmwareNetworksBaseURL, kind), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s vmware network request: %w", kind, err)
	}
	var task TaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to create %s vmware network: %w", kind, err)
	}
	return &task, nil
}

// CreateVmwareIsolatedNetworkAndWait creates an isolated VMware network, waits for the
// background task to finish and returns the created network.
//
// C-3: ...AndWait variant, matching the convention of the rest of the SDK.
func (c *CloudClient) CreateVmwareIsolatedNetworkAndWait(ctx context.Context, req *entities.VmwareCreateIsolatedNetworkRequest) (*entities.VmwareNetwork, error) {
	task, err := c.CreateVmwareIsolatedNetwork(ctx, req)
	if err != nil {
		return nil, err
	}
	return c.waitVmwareNetworkCreated(ctx, task)
}

// CreateVmwareRoutedNetworkAndWait creates a routed VMware network, waits for the
// background task to finish and returns the created network.
//
// C-3: ...AndWait variant, matching the convention of the rest of the SDK.
func (c *CloudClient) CreateVmwareRoutedNetworkAndWait(ctx context.Context, req *entities.VmwareCreateRoutedNetworkRequest) (*entities.VmwareNetwork, error) {
	task, err := c.CreateVmwareRoutedNetwork(ctx, req)
	if err != nil {
		return nil, err
	}
	return c.waitVmwareNetworkCreated(ctx, task)
}

// CreateVmwarePublicNetworkAndWait creates a public VMware network, waits for the
// background task to finish and returns the created network.
//
// C-3: ...AndWait variant, matching the convention of the rest of the SDK.
func (c *CloudClient) CreateVmwarePublicNetworkAndWait(ctx context.Context, req *entities.VmwareCreatePublicNetworkRequest) (*entities.VmwareNetwork, error) {
	task, err := c.CreateVmwarePublicNetwork(ctx, req)
	if err != nil {
		return nil, err
	}
	return c.waitVmwareNetworkCreated(ctx, task)
}

// waitVmwareNetworkCreated waits for a create-network task to complete and fetches the
// resulting network by the network_id carried in the completed task.
//
// C-3: shared tail of the CreateVmware*NetworkAndWait helpers.
func (c *CloudClient) waitVmwareNetworkCreated(ctx context.Context, task *TaskID) (*entities.VmwareNetwork, error) {
	completed, err := c.WaitVmwareTask(ctx, task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for vmware network creation: %w", err)
	}
	if completed.NetworkID == nil {
		return nil, fmt.Errorf("network ID not found in vmware task %s result", task.ID)
	}
	return c.GetVmwareNetwork(ctx, *completed.NetworkID)
}

// EditVmwareNetwork updates the name/bandwidth of a VMware network.
//
// N3: for an isolated network the operation is synchronous - the backend replies 200
// with an empty body and no task_id - so this method returns (nil, nil) in that case.
// For routed/public networks it returns a non-nil task ID to await.
func (c *CloudClient) EditVmwareNetwork(ctx context.Context, networkID int, req *entities.VmwareEditNetworkRequest) (*TaskID, error) {
	// N4/C-5: validate the id before issuing the request.
	if networkID <= 0 {
		return nil, fmt.Errorf("network ID must be greater than 0")
	}
	if req == nil {
		return nil, fmt.Errorf("edit network request is required")
	}
	httpReq, err := c.newRequest(ctx, http.MethodPut, buildVmwareNetworkPath(networkID), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create edit vmware network %d request: %w", networkID, err)
	}
	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to edit vmware network %d: %w", networkID, err)
	}
	// N3: an empty task_id means the isolated-network edit completed synchronously;
	// return (nil, nil) so callers do not await a meaningless empty task.
	if task.ID == "" {
		return nil, nil
	}
	return &task, nil
}

// DeleteVmwareNetwork deletes a VMware network and returns a task ID.
func (c *CloudClient) DeleteVmwareNetwork(ctx context.Context, networkID int) (*TaskID, error) {
	// N4/C-5: validate the id before issuing the request.
	if networkID <= 0 {
		return nil, fmt.Errorf("network ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodDelete, buildVmwareNetworkPath(networkID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create delete vmware network %d request: %w", networkID, err)
	}
	var task TaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to delete vmware network %d: %w", networkID, err)
	}
	return &task, nil
}

// ConnectVmwareServers connects a set of servers to a network and returns one task ID
// per server.
func (c *CloudClient) ConnectVmwareServers(ctx context.Context, networkID int, req *entities.VmwareConnectServersRequest) ([]string, error) {
	// N4/C-5: validate the id before issuing the request.
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
	return resp.TaskIds, nil
}

// ===================== Edge: Firewall =====================

// GetVmwareEdgeFirewall retrieves the edge firewall configuration of a VMware network.
func (c *CloudClient) GetVmwareEdgeFirewall(ctx context.Context, networkID int) (*entities.VmwareEdgeFirewall, error) {
	// N4/C-5: validate the id before issuing the request.
	if networkID <= 0 {
		return nil, fmt.Errorf("network ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodGet, buildVmwareEdgePath(networkID, "firewall"), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get edge firewall of vmware network %d request: %w", networkID, err)
	}
	// C-10: the edge GET responses (firewall/nat/vpn) are decoded WITHOUT an envelope on
	// purpose - verified against the backend controllers (VmwareEdgeController.GetFirewall
	// returns a flat VmwareFirewallResponse via JsonResult, not {"firewall": {...}}).
	var firewall entities.VmwareEdgeFirewall
	if err := c.doJSON(req, &firewall); err != nil {
		return nil, fmt.Errorf("failed to get edge firewall of vmware network %d: %w", networkID, err)
	}
	return &firewall, nil
}

// UpdateVmwareEdgeFirewall atomically replaces the edge firewall rule set.
//
// N5: WARNING - omitting Enabled turns the network firewall OFF on the backend. To guard
// against a naive read-modify-write silently disabling protection, Validate requires
// Enabled and DefaultAction to be set explicitly; populate them from a preceding
// GetVmwareEdgeFirewall.
func (c *CloudClient) UpdateVmwareEdgeFirewall(ctx context.Context, networkID int, req *entities.VmwareUpdateEdgeFirewallRequest) (*TaskID, error) {
	// N4/C-5: validate the id before issuing the request.
	if networkID <= 0 {
		return nil, fmt.Errorf("network ID must be greater than 0")
	}
	if req == nil {
		return nil, fmt.Errorf("update edge firewall request is required")
	}
	// N5: enforce Enabled/DefaultAction presence so a missing enabled cannot disable the firewall.
	if err := req.Validate(); err != nil {
		return nil, err
	}
	httpReq, err := c.newRequest(ctx, http.MethodPut, buildVmwareEdgePath(networkID, "firewall"), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create update edge firewall of vmware network %d request: %w", networkID, err)
	}
	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to update edge firewall of vmware network %d: %w", networkID, err)
	}
	return &task, nil
}

// ===================== Edge: NAT =====================

// GetVmwareEdgeNAT retrieves the edge NAT configuration of a VMware network.
//
// C-12: Nat -> NAT.
func (c *CloudClient) GetVmwareEdgeNAT(ctx context.Context, networkID int) (*entities.VmwareEdgeNAT, error) {
	// N4/C-5: validate the id before issuing the request.
	if networkID <= 0 {
		return nil, fmt.Errorf("network ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodGet, buildVmwareEdgePath(networkID, "nat"), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get edge nat of vmware network %d request: %w", networkID, err)
	}
	// C-10: decoded WITHOUT an envelope on purpose - VmwareEdgeController.GetNat returns a
	// flat VmwareNatResponse, not {"nat": {...}}.
	var nat entities.VmwareEdgeNAT
	if err := c.doJSON(req, &nat); err != nil {
		return nil, fmt.Errorf("failed to get edge nat of vmware network %d: %w", networkID, err)
	}
	return &nat, nil
}

// UpsertVmwareEdgeNATRule creates or updates a NAT rule (update when rule_id is set in
// the request).
//
// C-12: Nat -> NAT.
func (c *CloudClient) UpsertVmwareEdgeNATRule(ctx context.Context, networkID int, req *entities.VmwareUpsertNATRuleRequest) (*TaskID, error) {
	// N4/C-5: validate the id before issuing the request.
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
	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to upsert edge nat rule of vmware network %d: %w", networkID, err)
	}
	return &task, nil
}

// DeleteVmwareEdgeNATRule deletes a NAT rule and returns a task ID.
//
// C-12: Nat -> NAT.
//
// N1: currently unusable (API-blocked). The ruleID this method expects is the internal
// database id, but the only id a caller can obtain - VmwareEdgeNATRule.ID from
// GetVmwareEdgeNAT - is the vCloud object id, which the DELETE endpoint does not accept.
// There is thus no source for a valid ruleID until the API is fixed (networks-sdk.md, N1,
// networks-api.md, NET-1). The signature is left unchanged deliberately.
func (c *CloudClient) DeleteVmwareEdgeNATRule(ctx context.Context, networkID, ruleID int) (*TaskID, error) {
	// N4/C-5: validate the ids before issuing the request.
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
	var task TaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to delete edge nat rule %d of vmware network %d: %w", ruleID, networkID, err)
	}
	return &task, nil
}

// ===================== Edge: VPN =====================

// GetVmwareEdgeVPN retrieves the edge VPN configuration of a VMware network.
//
// C-12: Vpn -> VPN.
func (c *CloudClient) GetVmwareEdgeVPN(ctx context.Context, networkID int) (*entities.VmwareEdgeVPN, error) {
	// N4/C-5: validate the id before issuing the request.
	if networkID <= 0 {
		return nil, fmt.Errorf("network ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodGet, buildVmwareEdgePath(networkID, "vpn"), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get edge vpn of vmware network %d request: %w", networkID, err)
	}
	// C-10: decoded WITHOUT an envelope on purpose - VmwareEdgeController.GetVpn returns a
	// flat VmwareVpnResponse, not {"vpn": {...}}.
	var vpn entities.VmwareEdgeVPN
	if err := c.doJSON(req, &vpn); err != nil {
		return nil, fmt.Errorf("failed to get edge vpn of vmware network %d: %w", networkID, err)
	}
	return &vpn, nil
}

// UpsertVmwareEdgeVPNTunnel creates or updates a VPN tunnel (update when tunnel_id is set
// in the request).
//
// C-12: Vpn -> VPN.
//
// N2: Mtu, DiffieHellmanGroup and EncryptionType are in fact mandatory despite their
// optional-looking tags; Validate enforces their presence before the request is sent.
func (c *CloudClient) UpsertVmwareEdgeVPNTunnel(ctx context.Context, networkID int, req *entities.VmwareUpsertVPNTunnelRequest) (*TaskID, error) {
	// N4/C-5: validate the id before issuing the request.
	if networkID <= 0 {
		return nil, fmt.Errorf("network ID must be greater than 0")
	}
	if req == nil {
		return nil, fmt.Errorf("upsert vpn tunnel request is required")
	}
	// N2: require mtu/diffie_hellman_group/encryption_type - the backend rejects the request without them.
	if err := req.Validate(); err != nil {
		return nil, err
	}
	httpReq, err := c.newRequest(ctx, http.MethodPost, buildVmwareEdgePath(networkID, "vpn"), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create upsert edge vpn tunnel of vmware network %d request: %w", networkID, err)
	}
	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to upsert edge vpn tunnel of vmware network %d: %w", networkID, err)
	}
	return &task, nil
}

// DeleteVmwareEdgeVPNTunnel deletes a VPN tunnel and returns a task ID.
//
// C-12: Vpn -> VPN.
//
// N1: currently unusable (API-blocked). The tunnelID this method expects is the internal
// database id, but the only id a caller can obtain - VmwareEdgeVPNTunnel.ID from
// GetVmwareEdgeVPN - is the vCloud object id, which the DELETE endpoint does not accept.
// There is thus no source for a valid tunnelID until the API is fixed (networks-sdk.md,
// N1, networks-api.md, NET-1). The signature is left unchanged deliberately.
func (c *CloudClient) DeleteVmwareEdgeVPNTunnel(ctx context.Context, networkID, tunnelID int) (*TaskID, error) {
	// N4/C-5: validate the ids before issuing the request.
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
	var task TaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to delete edge vpn tunnel %d of vmware network %d: %w", tunnelID, networkID, err)
	}
	return &task, nil
}

// ===================== Edge: Bandwidth =====================

// UpdateVmwareEdgeBandwidth sets the edge uplink bandwidth and returns a task ID.
func (c *CloudClient) UpdateVmwareEdgeBandwidth(ctx context.Context, networkID int, req *entities.VmwareEdgeBandwidthRequest) (*TaskID, error) {
	// N4/C-5: validate the id before issuing the request.
	if networkID <= 0 {
		return nil, fmt.Errorf("network ID must be greater than 0")
	}
	if req == nil {
		return nil, fmt.Errorf("edge bandwidth request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	httpReq, err := c.newRequest(ctx, http.MethodPut, buildVmwareEdgePath(networkID, "bandwidth"), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create update edge bandwidth of vmware network %d request: %w", networkID, err)
	}
	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to update edge bandwidth of vmware network %d: %w", networkID, err)
	}
	return &task, nil
}
