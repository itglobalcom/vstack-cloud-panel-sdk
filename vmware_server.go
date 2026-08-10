package sdk

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

// C-12: single constant collected into a const block for consistency with the
// rest of the package.
const (
	vmwareServersBaseURL = "vmware/servers"
)

// Response envelopes for VMware servers and their sub-resources.
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
		// C-12: Nic -> NIC.
		NICs []*entities.VmwareNIC `json:"nics,omitempty"`
	}
	vmwareServerFirewallResponse struct {
		Rules []*entities.VmwareServerFirewallRule `json:"rules,omitempty"`
	}
)

// buildVmwareServerPath builds the request path for a VMware server and its
// sub-resources (C-12: renamed from vmwareServerPath for consistency with the
// buildXPath helpers used elsewhere in the package).
func buildVmwareServerPath(serverID int, parts ...string) string {
	path := fmt.Sprintf("%s/%d", vmwareServersBaseURL, serverID)
	for _, p := range parts {
		path = fmt.Sprintf("%s/%s", path, p)
	}
	return path
}

// ===================== Servers =====================

// GetVmwareServerList returns the VMware servers of the account, optionally
// filtered by location.
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

// GetVmwareServer returns a single VMware server by its id.
func (c *CloudClient) GetVmwareServer(ctx context.Context, serverID int) (*entities.VmwareServer, error) {
	// C-5/S6: validate the id before issuing the request.
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodGet, buildVmwareServerPath(serverID), nil)
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

// CreateVmwareServer places an order for a new VMware server and returns the
// order (the id of the created server plus the id of the background task).
func (c *CloudClient) CreateVmwareServer(ctx context.Context, req *entities.VmwareCreateServerRequest) (*entities.VmwareServerOrder, error) {
	if req == nil {
		return nil, fmt.Errorf("create vmware server request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid create vmware server request: %w", err)
	}
	// S4: when a GPU is requested the full triple (gpu_model_id, vram_mb,
	// card_count) is mandatory. req.Validate already enforces this, but the check
	// is made explicit here as the reviewer requested.
	if req.GPU != nil {
		if err := req.GPU.Validate(); err != nil {
			return nil, fmt.Errorf("invalid create vmware server request: %w", err)
		}
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

// CreateVmwareServerAndWait creates a VMware server, waits for the provisioning
// task to complete and returns the created server (C-3).
//
// This is a long-running operation: server provisioning takes several minutes
// (servers-sdk.md, S7), which exceeds the default PollingTimeout of 2 minutes.
// Configure a larger timeout with WithPollingTimeout when constructing the
// client, otherwise the wait will time out while the server is still being
// provisioned.
func (c *CloudClient) CreateVmwareServerAndWait(ctx context.Context, req *entities.VmwareCreateServerRequest) (*entities.VmwareServer, error) {
	order, err := c.CreateVmwareServer(ctx, req)
	if err != nil {
		return nil, err
	}
	if _, err := c.WaitVmwareTask(ctx, order.TaskID); err != nil {
		return nil, fmt.Errorf("failed to wait for vmware server creation: %w", err)
	}
	return c.GetVmwareServer(ctx, order.ServerID)
}

// VerifyVmwareServer performs a dry-run validation of a server order without
// creating it (the backend answers 200 with an empty body on success).
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

// ChangeVmwareServerConfiguration changes the CPU, RAM and system disk of a
// server and returns the id of the background task.
func (c *CloudClient) ChangeVmwareServerConfiguration(ctx context.Context, serverID int, req *entities.VmwareChangeConfigurationRequest) (*TaskID, error) {
	// C-5/S6: validate the id before issuing the request.
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	if req == nil {
		return nil, fmt.Errorf("change configuration request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid change configuration request: %w", err)
	}
	httpReq, err := c.newRequest(ctx, http.MethodPut, buildVmwareServerPath(serverID), req)
	if err != nil {
		// C-7: wrap the newRequest error in context.
		return nil, fmt.Errorf("failed to create change vmware server %d configuration request: %w", serverID, err)
	}
	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to change vmware server %d configuration: %w", serverID, err)
	}
	return &task, nil
}

// RenameVmwareServer changes the display name of a server (the backend answers
// 200 with an empty body).
//
// C-11: takes *entities.VmwareRenameServerRequest instead of a bare string, in
// line with the base RenameServer, so a future second field is not a breaking
// change.
func (c *CloudClient) RenameVmwareServer(ctx context.Context, serverID int, req *entities.VmwareRenameServerRequest) error {
	// C-5/S6: validate the id before issuing the request.
	if serverID <= 0 {
		return fmt.Errorf("server ID must be greater than 0")
	}
	if req == nil {
		return fmt.Errorf("rename server request is required")
	}
	if err := req.Validate(); err != nil {
		return fmt.Errorf("invalid rename server request: %w", err)
	}
	httpReq, err := c.newRequest(ctx, http.MethodPut, buildVmwareServerPath(serverID, "name"), req)
	if err != nil {
		// C-7: wrap the newRequest error in context.
		return fmt.Errorf("failed to create rename vmware server %d request: %w", serverID, err)
	}
	if err := c.doJSON(httpReq, nil); err != nil {
		return fmt.Errorf("failed to rename vmware server %d: %w", serverID, err)
	}
	return nil
}

// ChangeVmwareServerComputerName changes the guest OS hostname and returns the
// id of the background task.
func (c *CloudClient) ChangeVmwareServerComputerName(ctx context.Context, serverID int, req *entities.VmwareComputerNameRequest) (*TaskID, error) {
	// C-5/S6: validate the id before issuing the request.
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	if req == nil {
		return nil, fmt.Errorf("computer name request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid computer name request: %w", err)
	}
	httpReq, err := c.newRequest(ctx, http.MethodPut, buildVmwareServerPath(serverID, "computer-name"), req)
	if err != nil {
		// C-7: wrap the newRequest error in context.
		return nil, fmt.Errorf("failed to create change computer name request for vmware server %d: %w", serverID, err)
	}
	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to change computer name of vmware server %d: %w", serverID, err)
	}
	return &task, nil
}

// CopyVmwareServer creates a copy of a server and returns the order (the id of
// the new server plus the id of the background task).
func (c *CloudClient) CopyVmwareServer(ctx context.Context, serverID int, req *entities.VmwareCopyServerRequest) (*entities.VmwareServerOrder, error) {
	// C-5/S6: validate the id before issuing the request.
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	if req == nil {
		return nil, fmt.Errorf("copy request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid copy request: %w", err)
	}
	httpReq, err := c.newRequest(ctx, http.MethodPost, buildVmwareServerPath(serverID, "copy"), req)
	if err != nil {
		// C-7: wrap the newRequest error in context.
		return nil, fmt.Errorf("failed to create copy vmware server %d request: %w", serverID, err)
	}
	var order entities.VmwareServerOrder
	if err := c.doJSON(httpReq, &order); err != nil {
		return nil, fmt.Errorf("failed to copy vmware server %d: %w", serverID, err)
	}
	return &order, nil
}

// RebuildVmwareServer rebuilds a server from an image. A NEW server is created,
// so the returned order carries the new server_id.
func (c *CloudClient) RebuildVmwareServer(ctx context.Context, serverID int, req *entities.VmwareRebuildServerRequest) (*entities.VmwareServerOrder, error) {
	// C-5/S6: validate the id before issuing the request.
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	if req == nil {
		return nil, fmt.Errorf("rebuild request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid rebuild request: %w", err)
	}
	httpReq, err := c.newRequest(ctx, http.MethodPost, buildVmwareServerPath(serverID, "rebuild"), req)
	if err != nil {
		// C-7: wrap the newRequest error in context.
		return nil, fmt.Errorf("failed to create rebuild vmware server %d request: %w", serverID, err)
	}
	var order entities.VmwareServerOrder
	if err := c.doJSON(httpReq, &order); err != nil {
		return nil, fmt.Errorf("failed to rebuild vmware server %d: %w", serverID, err)
	}
	return &order, nil
}

// DeleteVmwareServer deletes a server and returns the id of the background task.
func (c *CloudClient) DeleteVmwareServer(ctx context.Context, serverID int) (*TaskID, error) {
	// C-5/S6: validate the id before issuing the request.
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodDelete, buildVmwareServerPath(serverID), nil)
	if err != nil {
		// C-7: wrap the newRequest error in context.
		return nil, fmt.Errorf("failed to create delete vmware server %d request: %w", serverID, err)
	}
	var task TaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to delete vmware server %d: %w", serverID, err)
	}
	return &task, nil
}

// ===================== Power =====================

// vmwarePower performs a power action on a server and returns the id of the
// background task.
func (c *CloudClient) vmwarePower(ctx context.Context, serverID int, action string) (*TaskID, error) {
	// C-5/S6: validate the id before issuing the request (covers all power methods).
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodPost, buildVmwareServerPath(serverID, "power", action), nil)
	if err != nil {
		// C-7: wrap the newRequest error in context.
		return nil, fmt.Errorf("failed to create %s vmware server %d request: %w", action, serverID, err)
	}
	var task TaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to %s vmware server %d: %w", action, serverID, err)
	}
	return &task, nil
}

// PowerOnVmwareServer powers on a server and returns the id of the background task.
func (c *CloudClient) PowerOnVmwareServer(ctx context.Context, serverID int) (*TaskID, error) {
	return c.vmwarePower(ctx, serverID, "on")
}

// PowerOffVmwareServer hard-powers off a server and returns the id of the background task.
func (c *CloudClient) PowerOffVmwareServer(ctx context.Context, serverID int) (*TaskID, error) {
	return c.vmwarePower(ctx, serverID, "off")
}

// ShutdownVmwareServer gracefully shuts down the guest OS and returns the id of
// the background task.
func (c *CloudClient) ShutdownVmwareServer(ctx context.Context, serverID int) (*TaskID, error) {
	return c.vmwarePower(ctx, serverID, "shutdown")
}

// RebootVmwareServer gracefully reboots the guest OS and returns the id of the
// background task.
func (c *CloudClient) RebootVmwareServer(ctx context.Context, serverID int) (*TaskID, error) {
	return c.vmwarePower(ctx, serverID, "reboot")
}

// ResetVmwareServer hard-resets a server and returns the id of the background task.
func (c *CloudClient) ResetVmwareServer(ctx context.Context, serverID int) (*TaskID, error) {
	return c.vmwarePower(ctx, serverID, "reset")
}

// ===================== Volumes =====================

// GetVmwareServerVolumes returns the additional data volumes of a server.
//
// C-12: renamed from GetVmwareVolumeList to the sub-resource plural form used
// elsewhere in the package.
func (c *CloudClient) GetVmwareServerVolumes(ctx context.Context, serverID int) ([]*entities.VmwareVolume, error) {
	// C-5/S6: validate the id before issuing the request.
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodGet, buildVmwareServerPath(serverID, "volumes"), nil)
	if err != nil {
		// C-7: wrap the newRequest error in context.
		return nil, fmt.Errorf("failed to create list volumes request for vmware server %d: %w", serverID, err)
	}
	var resp vmwareVolumesResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list volumes of vmware server %d: %w", serverID, err)
	}
	return resp.Volumes, nil
}

// GetVmwareVolume returns a single data volume of a server by its id.
func (c *CloudClient) GetVmwareVolume(ctx context.Context, serverID, volumeID int) (*entities.VmwareVolume, error) {
	// C-5/S6: validate the ids before issuing the request.
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	if volumeID <= 0 {
		return nil, fmt.Errorf("volume ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodGet, buildVmwareServerPath(serverID, "volumes", strconv.Itoa(volumeID)), nil)
	if err != nil {
		// C-7: wrap the newRequest error in context.
		return nil, fmt.Errorf("failed to create get volume %d request for vmware server %d: %w", volumeID, serverID, err)
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

// CreateVmwareVolume creates a data volume on a server and returns the id of the
// background task.
func (c *CloudClient) CreateVmwareVolume(ctx context.Context, serverID int, req *entities.VmwareCreateVolumeRequest) (*TaskID, error) {
	// C-5/S6: validate the id before issuing the request.
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	if req == nil {
		return nil, fmt.Errorf("create volume request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid create volume request: %w", err)
	}
	httpReq, err := c.newRequest(ctx, http.MethodPost, buildVmwareServerPath(serverID, "volumes"), req)
	if err != nil {
		// C-7: wrap the newRequest error in context.
		return nil, fmt.Errorf("failed to create volume request for vmware server %d: %w", serverID, err)
	}
	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to create volume on vmware server %d: %w", serverID, err)
	}
	return &task, nil
}

// EditVmwareVolume edits a data volume of a server and returns the id of the
// background task.
func (c *CloudClient) EditVmwareVolume(ctx context.Context, serverID, volumeID int, req *entities.VmwareEditVolumeRequest) (*TaskID, error) {
	// C-5/S6: validate the ids before issuing the request.
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	if volumeID <= 0 {
		return nil, fmt.Errorf("volume ID must be greater than 0")
	}
	if req == nil {
		return nil, fmt.Errorf("edit volume request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid edit volume request: %w", err)
	}
	httpReq, err := c.newRequest(ctx, http.MethodPut, buildVmwareServerPath(serverID, "volumes", strconv.Itoa(volumeID)), req)
	if err != nil {
		// C-7: wrap the newRequest error in context.
		return nil, fmt.Errorf("failed to create edit volume %d request for vmware server %d: %w", volumeID, serverID, err)
	}
	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to edit volume %d of vmware server %d: %w", volumeID, serverID, err)
	}
	return &task, nil
}

// DeleteVmwareVolume deletes a data volume of a server and returns the id of the
// background task.
func (c *CloudClient) DeleteVmwareVolume(ctx context.Context, serverID, volumeID int) (*TaskID, error) {
	// C-5/S6: validate the ids before issuing the request.
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	if volumeID <= 0 {
		return nil, fmt.Errorf("volume ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodDelete, buildVmwareServerPath(serverID, "volumes", strconv.Itoa(volumeID)), nil)
	if err != nil {
		// C-7: wrap the newRequest error in context.
		return nil, fmt.Errorf("failed to create delete volume %d request for vmware server %d: %w", volumeID, serverID, err)
	}
	var task TaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to delete volume %d of vmware server %d: %w", volumeID, serverID, err)
	}
	return &task, nil
}

// ===================== Snapshot (single per server) =====================

// GetVmwareSnapshot returns the snapshot of a server (a server has at most one).
func (c *CloudClient) GetVmwareSnapshot(ctx context.Context, serverID int) (*entities.VmwareSnapshot, error) {
	// C-5/S6: validate the id before issuing the request.
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodGet, buildVmwareServerPath(serverID, "snapshot"), nil)
	if err != nil {
		// C-7: wrap the newRequest error in context.
		return nil, fmt.Errorf("failed to create get snapshot request for vmware server %d: %w", serverID, err)
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

// CreateVmwareSnapshot creates the snapshot of a server and returns the id of
// the background task.
//
// C-11: takes *entities.VmwareCreateSnapshotRequest instead of a bare string, in
// line with the base CreateServerSnapshot, so a future second field is not a
// breaking change.
func (c *CloudClient) CreateVmwareSnapshot(ctx context.Context, serverID int, req *entities.VmwareCreateSnapshotRequest) (*TaskID, error) {
	// C-5/S6: validate the id before issuing the request.
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	if req == nil {
		return nil, fmt.Errorf("create snapshot request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid create snapshot request: %w", err)
	}
	httpReq, err := c.newRequest(ctx, http.MethodPost, buildVmwareServerPath(serverID, "snapshot"), req)
	if err != nil {
		// C-7: wrap the newRequest error in context.
		return nil, fmt.Errorf("failed to create snapshot request for vmware server %d: %w", serverID, err)
	}
	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to create snapshot of vmware server %d: %w", serverID, err)
	}
	return &task, nil
}

// RestoreVmwareSnapshot restores a server to its snapshot and returns the id of
// the background task.
func (c *CloudClient) RestoreVmwareSnapshot(ctx context.Context, serverID int) (*TaskID, error) {
	// C-5/S6: validate the id before issuing the request.
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodPost, buildVmwareServerPath(serverID, "snapshot", "restore"), nil)
	if err != nil {
		// C-7: wrap the newRequest error in context.
		return nil, fmt.Errorf("failed to create restore snapshot request for vmware server %d: %w", serverID, err)
	}
	var task TaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to restore snapshot of vmware server %d: %w", serverID, err)
	}
	return &task, nil
}

// DeleteVmwareSnapshot deletes the snapshot of a server and returns the id of
// the background task.
func (c *CloudClient) DeleteVmwareSnapshot(ctx context.Context, serverID int) (*TaskID, error) {
	// C-5/S6: validate the id before issuing the request.
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodDelete, buildVmwareServerPath(serverID, "snapshot"), nil)
	if err != nil {
		// C-7: wrap the newRequest error in context.
		return nil, fmt.Errorf("failed to create delete snapshot request for vmware server %d: %w", serverID, err)
	}
	var task TaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to delete snapshot of vmware server %d: %w", serverID, err)
	}
	return &task, nil
}

// ===================== Network interfaces =====================

// GetVmwareServerNICs returns the network interfaces of a server.
//
// C-12: renamed from GetVmwareServerNicList to the sub-resource plural form
// (Nic -> NIC) used elsewhere in the package.
func (c *CloudClient) GetVmwareServerNICs(ctx context.Context, serverID int) ([]*entities.VmwareNIC, error) {
	// C-5/S6: validate the id before issuing the request.
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodGet, buildVmwareServerPath(serverID, "nics"), nil)
	if err != nil {
		// C-7: wrap the newRequest error in context.
		return nil, fmt.Errorf("failed to create list nics request for vmware server %d: %w", serverID, err)
	}
	var resp vmwareNicsResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list nics of vmware server %d: %w", serverID, err)
	}
	return resp.NICs, nil
}

// ConnectVmwareClientNetwork attaches a server to a client network and returns
// the id of the background task.
func (c *CloudClient) ConnectVmwareClientNetwork(ctx context.Context, serverID int, req *entities.VmwareConnectClientNetworkRequest) (*TaskID, error) {
	// C-5/S6: validate the id before issuing the request.
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	if req == nil {
		return nil, fmt.Errorf("connect client network request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid connect client network request: %w", err)
	}
	httpReq, err := c.newRequest(ctx, http.MethodPost, buildVmwareServerPath(serverID, "nics"), req)
	if err != nil {
		// C-7: wrap the newRequest error in context.
		return nil, fmt.Errorf("failed to create connect client network request for vmware server %d: %w", serverID, err)
	}
	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to connect client network to vmware server %d: %w", serverID, err)
	}
	return &task, nil
}

// ConnectVmwareSharedNetwork attaches a server to a shared (public) network and
// returns the id of the background task.
func (c *CloudClient) ConnectVmwareSharedNetwork(ctx context.Context, serverID int, req *entities.VmwareConnectSharedNetworkRequest) (*TaskID, error) {
	// C-5/S6: validate the id before issuing the request.
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	if req == nil {
		return nil, fmt.Errorf("connect shared network request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid connect shared network request: %w", err)
	}
	httpReq, err := c.newRequest(ctx, http.MethodPost, buildVmwareServerPath(serverID, "nics", "shared"), req)
	if err != nil {
		// C-7: wrap the newRequest error in context.
		return nil, fmt.Errorf("failed to create connect shared network request for vmware server %d: %w", serverID, err)
	}
	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to connect shared network to vmware server %d: %w", serverID, err)
	}
	return &task, nil
}

// UpdateVmwareNIC updates a network interface of a server and returns the id of
// the background task.
//
// C-12: renamed from UpdateVmwareNic (Nic -> NIC).
func (c *CloudClient) UpdateVmwareNIC(ctx context.Context, serverID, nicID int, req *entities.VmwareUpdateNICRequest) (*TaskID, error) {
	// C-5/S6: validate the ids before issuing the request.
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	if nicID <= 0 {
		return nil, fmt.Errorf("NIC ID must be greater than 0")
	}
	if req == nil {
		return nil, fmt.Errorf("update nic request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid update nic request: %w", err)
	}
	httpReq, err := c.newRequest(ctx, http.MethodPut, buildVmwareServerPath(serverID, "nics", strconv.Itoa(nicID)), req)
	if err != nil {
		// C-7: wrap the newRequest error in context.
		return nil, fmt.Errorf("failed to create update nic %d request for vmware server %d: %w", nicID, serverID, err)
	}
	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to update nic %d of vmware server %d: %w", nicID, serverID, err)
	}
	return &task, nil
}

// DeleteVmwareNIC detaches a network interface from a server and returns the id
// of the background task.
//
// C-12: renamed from DeleteVmwareNic (Nic -> NIC).
func (c *CloudClient) DeleteVmwareNIC(ctx context.Context, serverID, nicID int) (*TaskID, error) {
	// C-5/S6: validate the ids before issuing the request.
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	if nicID <= 0 {
		return nil, fmt.Errorf("NIC ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodDelete, buildVmwareServerPath(serverID, "nics", strconv.Itoa(nicID)), nil)
	if err != nil {
		// C-7: wrap the newRequest error in context.
		return nil, fmt.Errorf("failed to create delete nic %d request for vmware server %d: %w", nicID, serverID, err)
	}
	var task TaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to delete nic %d of vmware server %d: %w", nicID, serverID, err)
	}
	return &task, nil
}

// ===================== Server firewall =====================

// GetVmwareServerFirewall returns the firewall rules of a server.
//
// S1: the returned VmwareServerFirewallRule is intentionally incomplete - the
// public API DTO does not expose the name and traffic_direction fields the
// backend uses (SRV-1), so through the public API the server firewall can only
// be cleared, not populated. This is an API-side limitation; do not add fields
// here that would never reach the backend.
//
// S2: a server without rules answers 404, which surfaces as an error for which
// IsNotFound reports true. For this endpoint a 404 therefore means "no rules OR
// no such server" - the two cannot be told apart on the SDK side, this is API
// behaviour (SRV-2).
func (c *CloudClient) GetVmwareServerFirewall(ctx context.Context, serverID int) ([]*entities.VmwareServerFirewallRule, error) {
	// C-5/S6: validate the id before issuing the request.
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodGet, buildVmwareServerPath(serverID, "firewall"), nil)
	if err != nil {
		// C-7: wrap the newRequest error in context.
		return nil, fmt.Errorf("failed to create get firewall request for vmware server %d: %w", serverID, err)
	}
	var resp vmwareServerFirewallResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get firewall of vmware server %d: %w", serverID, err)
	}
	return resp.Rules, nil
}

// UpdateVmwareServerFirewall atomically replaces the whole server firewall rule
// set (set semantics).
//
// S1: through the public API this method is usable only to CLEAR the firewall
// (an empty rule set). Any non-empty rule set is rejected with 400 because the
// public API DTO does not carry the name and traffic_direction fields the
// backend requires (SRV-1). Do not add those fields here until the API exposes
// them, otherwise they would never reach the backend.
//
// S3: a synchronous response (empty 200 body, which is what clearing the rules
// returns) carries no task_id. In that case this method returns (nil, nil) so
// callers do not feed an empty id into WaitVmwareTask; a non-nil TaskID means a
// background task was actually started.
func (c *CloudClient) UpdateVmwareServerFirewall(ctx context.Context, serverID int, req *entities.VmwareUpdateServerFirewallRequest) (*TaskID, error) {
	// C-5/S6: validate the id before issuing the request.
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	if req == nil {
		return nil, fmt.Errorf("update firewall request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid update firewall request: %w", err)
	}
	httpReq, err := c.newRequest(ctx, http.MethodPut, buildVmwareServerPath(serverID, "firewall"), req)
	if err != nil {
		// C-7: wrap the newRequest error in context.
		return nil, fmt.Errorf("failed to create update firewall request for vmware server %d: %w", serverID, err)
	}
	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to update firewall of vmware server %d: %w", serverID, err)
	}
	// S3: synchronous response (empty body) leaves task_id empty; report it as
	// "no task" rather than a &TaskID{ID: ""} that would break WaitVmwareTask.
	if task.ID == "" {
		return nil, nil
	}
	return &task, nil
}
