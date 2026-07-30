package sdk

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

const vmwareNetworksBaseURL = "vmware/networks"

// Ответы сетей и edge VMware.
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

func vmwareNetworkPath(networkID int, parts ...string) string {
	path := fmt.Sprintf("%s/%d", vmwareNetworksBaseURL, networkID)
	for _, p := range parts {
		path = fmt.Sprintf("%s/%s", path, p)
	}
	return path
}

func vmwareEdgePath(networkID int, parts ...string) string {
	return vmwareNetworkPath(networkID, append([]string{"edge"}, parts...)...)
}

// ===================== Сети =====================

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

func (c *CloudClient) GetVmwareNetwork(ctx context.Context, networkID int) (*entities.VmwareNetwork, error) {
	req, err := c.newRequest(ctx, http.MethodGet, vmwareNetworkPath(networkID), nil)
	if err != nil {
		return nil, err
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

func (c *CloudClient) CreateVmwareIsolatedNetwork(ctx context.Context, req *entities.VmwareCreateIsolatedNetworkRequest) (*TaskID, error) {
	if req == nil {
		return nil, fmt.Errorf("create isolated network request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return c.createVmwareNetwork(ctx, "isolated", req)
}

func (c *CloudClient) CreateVmwareRoutedNetwork(ctx context.Context, req *entities.VmwareCreateRoutedNetworkRequest) (*TaskID, error) {
	if req == nil {
		return nil, fmt.Errorf("create routed network request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return c.createVmwareNetwork(ctx, "routed", req)
}

func (c *CloudClient) CreateVmwarePublicNetwork(ctx context.Context, req *entities.VmwareCreatePublicNetworkRequest) (*TaskID, error) {
	if req == nil {
		return nil, fmt.Errorf("create public network request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return c.createVmwareNetwork(ctx, "public", req)
}

func (c *CloudClient) createVmwareNetwork(ctx context.Context, kind string, body interface{}) (*TaskID, error) {
	req, err := c.newRequest(ctx, http.MethodPost, fmt.Sprintf("%s/%s", vmwareNetworksBaseURL, kind), body)
	if err != nil {
		return nil, err
	}
	var task TaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to create %s vmware network: %w", kind, err)
	}
	return &task, nil
}

// EditVmwareNetwork меняет имя/полосу. Для изолированной сети операция синхронна
// (task_id в ответе пуст); для routed/public возвращается task_id.
func (c *CloudClient) EditVmwareNetwork(ctx context.Context, networkID int, req *entities.VmwareEditNetworkRequest) (*TaskID, error) {
	if req == nil {
		return nil, fmt.Errorf("edit network request is required")
	}
	httpReq, err := c.newRequest(ctx, http.MethodPut, vmwareNetworkPath(networkID), req)
	if err != nil {
		return nil, err
	}
	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to edit vmware network %d: %w", networkID, err)
	}
	return &task, nil
}

func (c *CloudClient) DeleteVmwareNetwork(ctx context.Context, networkID int) (*TaskID, error) {
	req, err := c.newRequest(ctx, http.MethodDelete, vmwareNetworkPath(networkID), nil)
	if err != nil {
		return nil, err
	}
	var task TaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to delete vmware network %d: %w", networkID, err)
	}
	return &task, nil
}

// ConnectVmwareServers подключает набор серверов к сети; возвращает task_id по каждому серверу.
func (c *CloudClient) ConnectVmwareServers(ctx context.Context, networkID int, req *entities.VmwareConnectServersRequest) ([]string, error) {
	if req == nil {
		return nil, fmt.Errorf("connect servers request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	httpReq, err := c.newRequest(ctx, http.MethodPost, vmwareNetworkPath(networkID, "servers"), req)
	if err != nil {
		return nil, err
	}
	var resp vmwareTaskRefsResponse
	if err := c.doJSON(httpReq, &resp); err != nil {
		return nil, fmt.Errorf("failed to connect servers to vmware network %d: %w", networkID, err)
	}
	return resp.TaskIds, nil
}

// ===================== Edge: Firewall =====================

func (c *CloudClient) GetVmwareEdgeFirewall(ctx context.Context, networkID int) (*entities.VmwareEdgeFirewall, error) {
	req, err := c.newRequest(ctx, http.MethodGet, vmwareEdgePath(networkID, "firewall"), nil)
	if err != nil {
		return nil, err
	}
	var firewall entities.VmwareEdgeFirewall
	if err := c.doJSON(req, &firewall); err != nil {
		return nil, fmt.Errorf("failed to get edge firewall of vmware network %d: %w", networkID, err)
	}
	return &firewall, nil
}

// UpdateVmwareEdgeFirewall атомарно заменяет набор правил firewall шлюза.
func (c *CloudClient) UpdateVmwareEdgeFirewall(ctx context.Context, networkID int, req *entities.VmwareUpdateEdgeFirewallRequest) (*TaskID, error) {
	if req == nil {
		return nil, fmt.Errorf("update edge firewall request is required")
	}
	httpReq, err := c.newRequest(ctx, http.MethodPut, vmwareEdgePath(networkID, "firewall"), req)
	if err != nil {
		return nil, err
	}
	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to update edge firewall of vmware network %d: %w", networkID, err)
	}
	return &task, nil
}

// ===================== Edge: NAT =====================

func (c *CloudClient) GetVmwareEdgeNat(ctx context.Context, networkID int) (*entities.VmwareEdgeNat, error) {
	req, err := c.newRequest(ctx, http.MethodGet, vmwareEdgePath(networkID, "nat"), nil)
	if err != nil {
		return nil, err
	}
	var nat entities.VmwareEdgeNat
	if err := c.doJSON(req, &nat); err != nil {
		return nil, fmt.Errorf("failed to get edge nat of vmware network %d: %w", networkID, err)
	}
	return &nat, nil
}

// UpsertVmwareEdgeNatRule создаёт или обновляет NAT-правило (обновление — при rule_id в запросе).
func (c *CloudClient) UpsertVmwareEdgeNatRule(ctx context.Context, networkID int, req *entities.VmwareUpsertNatRuleRequest) (*TaskID, error) {
	if req == nil {
		return nil, fmt.Errorf("upsert nat rule request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	httpReq, err := c.newRequest(ctx, http.MethodPost, vmwareEdgePath(networkID, "nat"), req)
	if err != nil {
		return nil, err
	}
	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to upsert edge nat rule of vmware network %d: %w", networkID, err)
	}
	return &task, nil
}

func (c *CloudClient) DeleteVmwareEdgeNatRule(ctx context.Context, networkID, ruleID int) (*TaskID, error) {
	req, err := c.newRequest(ctx, http.MethodDelete, vmwareEdgePath(networkID, "nat", strconv.Itoa(ruleID)), nil)
	if err != nil {
		return nil, err
	}
	var task TaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to delete edge nat rule %d of vmware network %d: %w", ruleID, networkID, err)
	}
	return &task, nil
}

// ===================== Edge: VPN =====================

func (c *CloudClient) GetVmwareEdgeVpn(ctx context.Context, networkID int) (*entities.VmwareEdgeVpn, error) {
	req, err := c.newRequest(ctx, http.MethodGet, vmwareEdgePath(networkID, "vpn"), nil)
	if err != nil {
		return nil, err
	}
	var vpn entities.VmwareEdgeVpn
	if err := c.doJSON(req, &vpn); err != nil {
		return nil, fmt.Errorf("failed to get edge vpn of vmware network %d: %w", networkID, err)
	}
	return &vpn, nil
}

// UpsertVmwareEdgeVpnTunnel создаёт или обновляет VPN-туннель (обновление — при tunnel_id в запросе).
func (c *CloudClient) UpsertVmwareEdgeVpnTunnel(ctx context.Context, networkID int, req *entities.VmwareUpsertVpnTunnelRequest) (*TaskID, error) {
	if req == nil {
		return nil, fmt.Errorf("upsert vpn tunnel request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	httpReq, err := c.newRequest(ctx, http.MethodPost, vmwareEdgePath(networkID, "vpn"), req)
	if err != nil {
		return nil, err
	}
	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to upsert edge vpn tunnel of vmware network %d: %w", networkID, err)
	}
	return &task, nil
}

func (c *CloudClient) DeleteVmwareEdgeVpnTunnel(ctx context.Context, networkID, tunnelID int) (*TaskID, error) {
	req, err := c.newRequest(ctx, http.MethodDelete, vmwareEdgePath(networkID, "vpn", strconv.Itoa(tunnelID)), nil)
	if err != nil {
		return nil, err
	}
	var task TaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to delete edge vpn tunnel %d of vmware network %d: %w", tunnelID, networkID, err)
	}
	return &task, nil
}

// ===================== Edge: полоса пропускания =====================

func (c *CloudClient) UpdateVmwareEdgeBandwidth(ctx context.Context, networkID int, req *entities.VmwareEdgeBandwidthRequest) (*TaskID, error) {
	if req == nil {
		return nil, fmt.Errorf("edge bandwidth request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	httpReq, err := c.newRequest(ctx, http.MethodPut, vmwareEdgePath(networkID, "bandwidth"), req)
	if err != nil {
		return nil, err
	}
	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to update edge bandwidth of vmware network %d: %w", networkID, err)
	}
	return &task, nil
}
