package sdk

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

const vmwareServersBaseURL = "vmware/servers"

// Ответы серверов и субресурсов VMware.
type (
	vmwareServerResponse struct {
		Server *entities.VmwareServer `json:"server,omitempty"`
	}
	vmwareServersResponse struct {
		Servers []*entities.VmwareServer `json:"servers,omitempty"`
	}
	vmwareVolumeResponse struct {
		Volume *entities.VmwareVolume `json:"volume,omitempty"`
	}
	vmwareVolumesResponse struct {
		Volumes []*entities.VmwareVolume `json:"volumes,omitempty"`
	}
	vmwareSnapshotResponse struct {
		Snapshot *entities.VmwareSnapshot `json:"snapshot,omitempty"`
	}
	vmwareNicsResponse struct {
		Nics []*entities.VmwareNic `json:"nics,omitempty"`
	}
	vmwareServerFirewallResponse struct {
		Rules []*entities.VmwareServerFirewallRule `json:"rules,omitempty"`
	}
)

func vmwareServerPath(serverID int, parts ...string) string {
	path := fmt.Sprintf("%s/%d", vmwareServersBaseURL, serverID)
	for _, p := range parts {
		path = fmt.Sprintf("%s/%s", path, p)
	}
	return path
}

// ===================== Серверы =====================

func (c *CloudClient) GetVmwareServerList(ctx context.Context, locationID *int) ([]*entities.VmwareServer, error) {
	params := url.Values{}
	if locationID != nil {
		params.Set("location_id", strconv.Itoa(*locationID))
	}
	req, err := c.newRequest(ctx, http.MethodGet, withQuery(vmwareServersBaseURL, params), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list vmware servers request: %w", err)
	}
	var resp vmwareServersResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list vmware servers: %w", err)
	}
	return resp.Servers, nil
}

func (c *CloudClient) GetVmwareServer(ctx context.Context, serverID int) (*entities.VmwareServer, error) {
	req, err := c.newRequest(ctx, http.MethodGet, vmwareServerPath(serverID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get vmware server request: %w", err)
	}
	var resp vmwareServerResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get vmware server %d: %w", serverID, err)
	}
	if resp.Server == nil {
		return nil, fmt.Errorf("vmware server %d not found in response: %w", serverID, ErrNotFound)
	}
	return resp.Server, nil
}

func (c *CloudClient) CreateVmwareServer(ctx context.Context, req *entities.VmwareCreateServerRequest) (*entities.VmwareServerOrder, error) {
	if req == nil {
		return nil, fmt.Errorf("create vmware server request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid create vmware server request: %w", err)
	}
	httpReq, err := c.newRequest(ctx, http.MethodPost, vmwareServersBaseURL, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create vmware server request: %w", err)
	}
	var order entities.VmwareServerOrder
	if err := c.doJSON(httpReq, &order); err != nil {
		return nil, fmt.Errorf("failed to create vmware server: %w", err)
	}
	return &order, nil
}

// VerifyVmwareServer — dry-run валидация заказа (200 без тела при успехе).
func (c *CloudClient) VerifyVmwareServer(ctx context.Context, req *entities.VmwareCreateServerRequest) error {
	if req == nil {
		return fmt.Errorf("verify vmware server request is required")
	}
	if err := req.Validate(); err != nil {
		return fmt.Errorf("invalid verify vmware server request: %w", err)
	}
	httpReq, err := c.newRequest(ctx, http.MethodPost, fmt.Sprintf("%s/verify", vmwareServersBaseURL), req)
	if err != nil {
		return fmt.Errorf("failed to create verify vmware server request: %w", err)
	}
	if err := c.doJSON(httpReq, nil); err != nil {
		return fmt.Errorf("failed to verify vmware server: %w", err)
	}
	return nil
}

func (c *CloudClient) ChangeVmwareServerConfiguration(ctx context.Context, serverID int, req *entities.VmwareChangeConfigurationRequest) (*TaskID, error) {
	if req == nil {
		return nil, fmt.Errorf("change configuration request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid change configuration request: %w", err)
	}
	httpReq, err := c.newRequest(ctx, http.MethodPut, vmwareServerPath(serverID), req)
	if err != nil {
		return nil, err
	}
	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to change vmware server %d configuration: %w", serverID, err)
	}
	return &task, nil
}

// RenameVmwareServer меняет отображаемое имя сервера (200 без тела).
func (c *CloudClient) RenameVmwareServer(ctx context.Context, serverID int, name string) error {
	req := &entities.VmwareRenameServerRequest{Name: name}
	if err := req.Validate(); err != nil {
		return err
	}
	httpReq, err := c.newRequest(ctx, http.MethodPut, vmwareServerPath(serverID, "name"), req)
	if err != nil {
		return err
	}
	if err := c.doJSON(httpReq, nil); err != nil {
		return fmt.Errorf("failed to rename vmware server %d: %w", serverID, err)
	}
	return nil
}

// ChangeVmwareServerComputerName меняет hostname гостевой ОС.
func (c *CloudClient) ChangeVmwareServerComputerName(ctx context.Context, serverID int, req *entities.VmwareComputerNameRequest) (*TaskID, error) {
	if req == nil {
		return nil, fmt.Errorf("computer name request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	httpReq, err := c.newRequest(ctx, http.MethodPut, vmwareServerPath(serverID, "computer-name"), req)
	if err != nil {
		return nil, err
	}
	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to change computer name of vmware server %d: %w", serverID, err)
	}
	return &task, nil
}

func (c *CloudClient) CopyVmwareServer(ctx context.Context, serverID int, req *entities.VmwareCopyServerRequest) (*entities.VmwareServerOrder, error) {
	if req == nil {
		return nil, fmt.Errorf("copy request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	httpReq, err := c.newRequest(ctx, http.MethodPost, vmwareServerPath(serverID, "copy"), req)
	if err != nil {
		return nil, err
	}
	var order entities.VmwareServerOrder
	if err := c.doJSON(httpReq, &order); err != nil {
		return nil, fmt.Errorf("failed to copy vmware server %d: %w", serverID, err)
	}
	return &order, nil
}

// RebuildVmwareServer пересоздаёт сервер (создаётся НОВЫЙ сервер — server_id в ответе).
func (c *CloudClient) RebuildVmwareServer(ctx context.Context, serverID int, req *entities.VmwareRebuildServerRequest) (*entities.VmwareServerOrder, error) {
	if req == nil {
		return nil, fmt.Errorf("rebuild request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	httpReq, err := c.newRequest(ctx, http.MethodPost, vmwareServerPath(serverID, "rebuild"), req)
	if err != nil {
		return nil, err
	}
	var order entities.VmwareServerOrder
	if err := c.doJSON(httpReq, &order); err != nil {
		return nil, fmt.Errorf("failed to rebuild vmware server %d: %w", serverID, err)
	}
	return &order, nil
}

func (c *CloudClient) DeleteVmwareServer(ctx context.Context, serverID int) (*TaskID, error) {
	req, err := c.newRequest(ctx, http.MethodDelete, vmwareServerPath(serverID), nil)
	if err != nil {
		return nil, err
	}
	var task TaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to delete vmware server %d: %w", serverID, err)
	}
	return &task, nil
}

// ===================== Питание =====================

func (c *CloudClient) vmwarePower(ctx context.Context, serverID int, action string) (*TaskID, error) {
	req, err := c.newRequest(ctx, http.MethodPost, vmwareServerPath(serverID, "power", action), nil)
	if err != nil {
		return nil, err
	}
	var task TaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to %s vmware server %d: %w", action, serverID, err)
	}
	return &task, nil
}

func (c *CloudClient) PowerOnVmwareServer(ctx context.Context, serverID int) (*TaskID, error) {
	return c.vmwarePower(ctx, serverID, "on")
}
func (c *CloudClient) PowerOffVmwareServer(ctx context.Context, serverID int) (*TaskID, error) {
	return c.vmwarePower(ctx, serverID, "off")
}
func (c *CloudClient) ShutdownVmwareServer(ctx context.Context, serverID int) (*TaskID, error) {
	return c.vmwarePower(ctx, serverID, "shutdown")
}
func (c *CloudClient) RebootVmwareServer(ctx context.Context, serverID int) (*TaskID, error) {
	return c.vmwarePower(ctx, serverID, "reboot")
}
func (c *CloudClient) ResetVmwareServer(ctx context.Context, serverID int) (*TaskID, error) {
	return c.vmwarePower(ctx, serverID, "reset")
}

// ===================== Диски =====================

func (c *CloudClient) GetVmwareVolumeList(ctx context.Context, serverID int) ([]*entities.VmwareVolume, error) {
	req, err := c.newRequest(ctx, http.MethodGet, vmwareServerPath(serverID, "volumes"), nil)
	if err != nil {
		return nil, err
	}
	var resp vmwareVolumesResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list volumes of vmware server %d: %w", serverID, err)
	}
	return resp.Volumes, nil
}

func (c *CloudClient) GetVmwareVolume(ctx context.Context, serverID, volumeID int) (*entities.VmwareVolume, error) {
	req, err := c.newRequest(ctx, http.MethodGet, vmwareServerPath(serverID, "volumes", strconv.Itoa(volumeID)), nil)
	if err != nil {
		return nil, err
	}
	var resp vmwareVolumeResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get volume %d of vmware server %d: %w", volumeID, serverID, err)
	}
	if resp.Volume == nil {
		return nil, fmt.Errorf("volume %d not found: %w", volumeID, ErrNotFound)
	}
	return resp.Volume, nil
}

func (c *CloudClient) CreateVmwareVolume(ctx context.Context, serverID int, req *entities.VmwareCreateVolumeRequest) (*TaskID, error) {
	if req == nil {
		return nil, fmt.Errorf("create volume request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	httpReq, err := c.newRequest(ctx, http.MethodPost, vmwareServerPath(serverID, "volumes"), req)
	if err != nil {
		return nil, err
	}
	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to create volume on vmware server %d: %w", serverID, err)
	}
	return &task, nil
}

func (c *CloudClient) EditVmwareVolume(ctx context.Context, serverID, volumeID int, req *entities.VmwareEditVolumeRequest) (*TaskID, error) {
	if req == nil {
		return nil, fmt.Errorf("edit volume request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	httpReq, err := c.newRequest(ctx, http.MethodPut, vmwareServerPath(serverID, "volumes", strconv.Itoa(volumeID)), req)
	if err != nil {
		return nil, err
	}
	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to edit volume %d of vmware server %d: %w", volumeID, serverID, err)
	}
	return &task, nil
}

func (c *CloudClient) DeleteVmwareVolume(ctx context.Context, serverID, volumeID int) (*TaskID, error) {
	req, err := c.newRequest(ctx, http.MethodDelete, vmwareServerPath(serverID, "volumes", strconv.Itoa(volumeID)), nil)
	if err != nil {
		return nil, err
	}
	var task TaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to delete volume %d of vmware server %d: %w", volumeID, serverID, err)
	}
	return &task, nil
}

// ===================== Снимок (единичный на сервер) =====================

func (c *CloudClient) GetVmwareSnapshot(ctx context.Context, serverID int) (*entities.VmwareSnapshot, error) {
	req, err := c.newRequest(ctx, http.MethodGet, vmwareServerPath(serverID, "snapshot"), nil)
	if err != nil {
		return nil, err
	}
	var resp vmwareSnapshotResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get snapshot of vmware server %d: %w", serverID, err)
	}
	if resp.Snapshot == nil {
		return nil, fmt.Errorf("snapshot of vmware server %d not found: %w", serverID, ErrNotFound)
	}
	return resp.Snapshot, nil
}

func (c *CloudClient) CreateVmwareSnapshot(ctx context.Context, serverID int, name string) (*TaskID, error) {
	req := &entities.VmwareCreateSnapshotRequest{Name: name}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	httpReq, err := c.newRequest(ctx, http.MethodPost, vmwareServerPath(serverID, "snapshot"), req)
	if err != nil {
		return nil, err
	}
	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to create snapshot of vmware server %d: %w", serverID, err)
	}
	return &task, nil
}

func (c *CloudClient) RestoreVmwareSnapshot(ctx context.Context, serverID int) (*TaskID, error) {
	req, err := c.newRequest(ctx, http.MethodPost, vmwareServerPath(serverID, "snapshot", "restore"), nil)
	if err != nil {
		return nil, err
	}
	var task TaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to restore snapshot of vmware server %d: %w", serverID, err)
	}
	return &task, nil
}

func (c *CloudClient) DeleteVmwareSnapshot(ctx context.Context, serverID int) (*TaskID, error) {
	req, err := c.newRequest(ctx, http.MethodDelete, vmwareServerPath(serverID, "snapshot"), nil)
	if err != nil {
		return nil, err
	}
	var task TaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to delete snapshot of vmware server %d: %w", serverID, err)
	}
	return &task, nil
}

// ===================== Сетевые интерфейсы =====================

func (c *CloudClient) GetVmwareServerNicList(ctx context.Context, serverID int) ([]*entities.VmwareNic, error) {
	req, err := c.newRequest(ctx, http.MethodGet, vmwareServerPath(serverID, "nics"), nil)
	if err != nil {
		return nil, err
	}
	var resp vmwareNicsResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list nics of vmware server %d: %w", serverID, err)
	}
	return resp.Nics, nil
}

func (c *CloudClient) ConnectVmwareClientNetwork(ctx context.Context, serverID int, req *entities.VmwareConnectClientNetworkRequest) (*TaskID, error) {
	if req == nil {
		return nil, fmt.Errorf("connect client network request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	httpReq, err := c.newRequest(ctx, http.MethodPost, vmwareServerPath(serverID, "nics"), req)
	if err != nil {
		return nil, err
	}
	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to connect client network to vmware server %d: %w", serverID, err)
	}
	return &task, nil
}

func (c *CloudClient) ConnectVmwareSharedNetwork(ctx context.Context, serverID int, req *entities.VmwareConnectSharedNetworkRequest) (*TaskID, error) {
	if req == nil {
		return nil, fmt.Errorf("connect shared network request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	httpReq, err := c.newRequest(ctx, http.MethodPost, vmwareServerPath(serverID, "nics", "shared"), req)
	if err != nil {
		return nil, err
	}
	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to connect shared network to vmware server %d: %w", serverID, err)
	}
	return &task, nil
}

func (c *CloudClient) UpdateVmwareNic(ctx context.Context, serverID, nicID int, req *entities.VmwareUpdateNicRequest) (*TaskID, error) {
	if req == nil {
		return nil, fmt.Errorf("update nic request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	httpReq, err := c.newRequest(ctx, http.MethodPut, vmwareServerPath(serverID, "nics", strconv.Itoa(nicID)), req)
	if err != nil {
		return nil, err
	}
	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to update nic %d of vmware server %d: %w", nicID, serverID, err)
	}
	return &task, nil
}

func (c *CloudClient) DeleteVmwareNic(ctx context.Context, serverID, nicID int) (*TaskID, error) {
	req, err := c.newRequest(ctx, http.MethodDelete, vmwareServerPath(serverID, "nics", strconv.Itoa(nicID)), nil)
	if err != nil {
		return nil, err
	}
	var task TaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to delete nic %d of vmware server %d: %w", nicID, serverID, err)
	}
	return &task, nil
}

// ===================== Firewall сервера =====================

func (c *CloudClient) GetVmwareServerFirewall(ctx context.Context, serverID int) ([]*entities.VmwareServerFirewallRule, error) {
	req, err := c.newRequest(ctx, http.MethodGet, vmwareServerPath(serverID, "firewall"), nil)
	if err != nil {
		return nil, err
	}
	var resp vmwareServerFirewallResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get firewall of vmware server %d: %w", serverID, err)
	}
	return resp.Rules, nil
}

// UpdateVmwareServerFirewall — атомарный replace набора правил (set-семантика).
func (c *CloudClient) UpdateVmwareServerFirewall(ctx context.Context, serverID int, req *entities.VmwareUpdateServerFirewallRequest) (*TaskID, error) {
	if req == nil {
		return nil, fmt.Errorf("update firewall request is required")
	}
	httpReq, err := c.newRequest(ctx, http.MethodPut, vmwareServerPath(serverID, "firewall"), req)
	if err != nil {
		return nil, err
	}
	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to update firewall of vmware server %d: %w", serverID, err)
	}
	return &task, nil
}
