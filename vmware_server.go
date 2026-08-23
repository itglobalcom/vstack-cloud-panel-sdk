package sdk

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

// Single constant collected into a const block for consistency with the
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
		NICs []*entities.VmwareNIC `json:"nics,omitempty"`
	}
	vmwareServerFirewallResponse struct {
		// Values, not pointers: the read rule is the same type the update request
		// takes, so a read-modify-write is a direct assignment.
		Rules []entities.VmwareServerFirewallRule `json:"rules,omitempty"`
	}
)

// buildVmwareServerPath builds the request path for a VMware server and its
// sub-resources.
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
//
// List items omit the live-only VmwareServer.VmToolsInstalled, which stays
// nil here; fetch a server by id to read it.
//
// locationID is a pointer so that "no filter" is distinguishable from 0, and the
// parameter is omitted from the query string rather than sent empty — an empty
// value of a declared query parameter makes the API answer HTTP 500.
func (c *CloudClient) GetVmwareServerList(ctx context.Context, locationID *int) ([]*entities.VmwareServer, error) {
	if locationID != nil && *locationID <= 0 {
		return nil, fmt.Errorf("location ID must be positive")
	}
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
//
// The order carries the task id as a bare string; wrap it with VmwareTaskID{ID:
// order.TaskID} to await it, or use CreateVmwareServerAndWait.
func (c *CloudClient) CreateVmwareServer(ctx context.Context, req *entities.VmwareCreateServerRequest) (*entities.VmwareServerOrder, error) {
	if req == nil {
		return nil, fmt.Errorf("create vmware server request is required")
	}
	// req.Validate already enforces the full GPU triple when GPU != nil.
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

// CreateVmwareServerAndWait creates a VMware server, waits for the provisioning
// task to complete and returns the created server.
//
// Provisioning takes minutes, which is why VMware task waiting has its own
// timeout floor (VmwareTaskWaitDefaultTimeout) rather than the 2m base
// PollingTimeout.
func (c *CloudClient) CreateVmwareServerAndWait(ctx context.Context, req *entities.VmwareCreateServerRequest) (*entities.VmwareServer, error) {
	order, err := c.CreateVmwareServer(ctx, req)
	if err != nil {
		return nil, err
	}
	if _, err := c.WaitVmwareTask(ctx, order.TaskID); err != nil {
		// Name the server: it is already being provisioned, and a caller of the
		// ...AndWait form has no other way to learn which one to reconcile or delete.
		return nil, fmt.Errorf("failed to wait for vmware server %d creation: %w", order.ServerID, err)
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
// server and returns the background task to await.
func (c *CloudClient) ChangeVmwareServerConfiguration(ctx context.Context, serverID int, req *entities.VmwareChangeConfigurationRequest) (*VmwareTaskID, error) {
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
		return nil, fmt.Errorf("failed to create change vmware server %d configuration request: %w", serverID, err)
	}
	var task VmwareTaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to change vmware server %d configuration: %w", serverID, err)
	}
	return &task, nil
}

// ChangeVmwareServerConfigurationAndWait changes the configuration, waits for the
// task and returns the refreshed server.
func (c *CloudClient) ChangeVmwareServerConfigurationAndWait(ctx context.Context, serverID int, req *entities.VmwareChangeConfigurationRequest) (*entities.VmwareServer, error) {
	task, err := c.ChangeVmwareServerConfiguration(ctx, serverID, req)
	if err != nil {
		return nil, err
	}
	if err := c.awaitVmwareTask(ctx, task); err != nil {
		return nil, err
	}
	return c.GetVmwareServer(ctx, serverID)
}

// RenameVmwareServer changes the display name of a server. The operation is
// synchronous — the backend answers 200 with an empty body and starts no task, so
// there is nothing to await and no ...AndWait variant.
//
// Takes *entities.VmwareRenameServerRequest instead of a bare string, in
// line with the base RenameServer, so a future second field is not a breaking
// change.
func (c *CloudClient) RenameVmwareServer(ctx context.Context, serverID int, req *entities.VmwareRenameServerRequest) error {
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
		return fmt.Errorf("failed to create rename vmware server %d request: %w", serverID, err)
	}
	if err := c.doJSON(httpReq, nil); err != nil {
		return fmt.Errorf("failed to rename vmware server %d: %w", serverID, err)
	}
	return nil
}

// ChangeVmwareServerComputerName changes the guest OS hostname and returns the
// background task to await.
//
// The backend upper-cases the value it stores, so the name read back will
// not match the one sent unless it was already upper case. Compare
// case-insensitively.
func (c *CloudClient) ChangeVmwareServerComputerName(ctx context.Context, serverID int, req *entities.VmwareComputerNameRequest) (*VmwareTaskID, error) {
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
		return nil, fmt.Errorf("failed to create change computer name request for vmware server %d: %w", serverID, err)
	}
	var task VmwareTaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to change computer name of vmware server %d: %w", serverID, err)
	}
	return &task, nil
}

// ChangeVmwareServerComputerNameAndWait changes the guest hostname, waits for the
// task and returns the refreshed server.
func (c *CloudClient) ChangeVmwareServerComputerNameAndWait(ctx context.Context, serverID int, req *entities.VmwareComputerNameRequest) (*entities.VmwareServer, error) {
	task, err := c.ChangeVmwareServerComputerName(ctx, serverID, req)
	if err != nil {
		return nil, err
	}
	if err := c.awaitVmwareTask(ctx, task); err != nil {
		return nil, err
	}
	return c.GetVmwareServer(ctx, serverID)
}

// CopyVmwareServer creates a copy of a server and returns the order (the id of
// the new server plus the id of the background task).
func (c *CloudClient) CopyVmwareServer(ctx context.Context, serverID int, req *entities.VmwareCopyServerRequest) (*entities.VmwareServerOrder, error) {
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
		return nil, fmt.Errorf("failed to create copy vmware server %d request: %w", serverID, err)
	}
	var order entities.VmwareServerOrder
	if err := c.doJSON(httpReq, &order); err != nil {
		return nil, fmt.Errorf("failed to copy vmware server %d: %w", serverID, err)
	}
	return &order, nil
}

// CopyVmwareServerAndWait copies a server, waits for the task and returns the new
// server. Copying takes minutes.
func (c *CloudClient) CopyVmwareServerAndWait(ctx context.Context, serverID int, req *entities.VmwareCopyServerRequest) (*entities.VmwareServer, error) {
	order, err := c.CopyVmwareServer(ctx, serverID, req)
	if err != nil {
		return nil, err
	}
	if _, err := c.WaitVmwareTask(ctx, order.TaskID); err != nil {
		// Name the copy: it already exists — see RebuildVmwareServerAndWait.
		return nil, fmt.Errorf("failed to wait for vmware server %d copy (copy %d was created): %w",
			serverID, order.ServerID, err)
	}
	return c.GetVmwareServer(ctx, order.ServerID)
}

// RebuildVmwareServer rebuilds a server from an image. A NEW server is created,
// so the returned order carries the new server_id and the original one is
// scheduled for deletion.
func (c *CloudClient) RebuildVmwareServer(ctx context.Context, serverID int, req *entities.VmwareRebuildServerRequest) (*entities.VmwareServerOrder, error) {
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
		return nil, fmt.Errorf("failed to create rebuild vmware server %d request: %w", serverID, err)
	}
	var order entities.VmwareServerOrder
	if err := c.doJSON(httpReq, &order); err != nil {
		return nil, fmt.Errorf("failed to rebuild vmware server %d: %w", serverID, err)
	}
	return &order, nil
}

// RebuildVmwareServerAndWait rebuilds a server, waits for the task and returns
// the NEW server.
//
// This is the longest and least predictable operation in the section. Two rebuilds
// of the same server from the same image, in one run, took 4m40s and 24m30s;
// earlier measurements gave >12m, ~20m and ~26m. So it fits inside
// VmwareTaskWaitDefaultTimeout, but ~26m leaves little headroom — pass a larger
// WithPollingTimeout when calling it.
//
// Prefer the two-step form (RebuildVmwareServer, then WaitVmwareTaskWithTimeout)
// when a lost server would matter: the replacement is created as soon as the POST
// returns, so if the wait times out here the new server exists but its id is not
// returned. It is named in the error message, and RebuildVmwareServer hands it back
// directly.
//
// The replaced server outlives the task: it has been observed still readable in
// state "deleting" minutes after the task reported completed. Follow up with
// WaitVmwareServerGone on the original id if that matters.
func (c *CloudClient) RebuildVmwareServerAndWait(ctx context.Context, serverID int, req *entities.VmwareRebuildServerRequest) (*entities.VmwareServer, error) {
	order, err := c.RebuildVmwareServer(ctx, serverID, req)
	if err != nil {
		return nil, err
	}
	if _, err := c.WaitVmwareTask(ctx, order.TaskID); err != nil {
		// Name the replacement server: it already exists, and this is the only way a
		// caller of the ...AndWait form can find out which one to reconcile or clean up.
		return nil, fmt.Errorf("failed to wait for vmware server %d rebuild (replacement server %d was created): %w",
			serverID, order.ServerID, err)
	}
	return c.GetVmwareServer(ctx, order.ServerID)
}

// DeleteVmwareServer deletes a server and returns the background task to await.
func (c *CloudClient) DeleteVmwareServer(ctx context.Context, serverID int) (*VmwareTaskID, error) {
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodDelete, buildVmwareServerPath(serverID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create delete vmware server %d request: %w", serverID, err)
	}
	var task VmwareTaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to delete vmware server %d: %w", serverID, err)
	}
	return &task, nil
}

// DeleteVmwareServerAndWait deletes a server and waits until it is really gone.
//
// It waits for both the task AND the disappearance of the object: a completed
// delete task does not guarantee the server is no longer readable (it has been
// observed lingering in state "deleting" for minutes after a rebuild replaced it),
// and the difference is not something a caller can predict. See WaitVmwareServerGone.
func (c *CloudClient) DeleteVmwareServerAndWait(ctx context.Context, serverID int) error {
	task, err := c.DeleteVmwareServer(ctx, serverID)
	if err != nil {
		return err
	}
	if err := c.awaitVmwareTask(ctx, task); err != nil {
		return err
	}
	return c.WaitVmwareServerGone(ctx, serverID)
}

// ===================== Power =====================

// vmwarePower performs a power action on a server and returns the background task.
func (c *CloudClient) vmwarePower(ctx context.Context, serverID int, action string) (*VmwareTaskID, error) {
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodPost, buildVmwareServerPath(serverID, "power", action), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s vmware server %d request: %w", action, serverID, err)
	}
	var task VmwareTaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to %s vmware server %d: %w", action, serverID, err)
	}
	return &task, nil
}

// vmwarePowerAndWait performs a power action, waits for its task and returns the
// refreshed server.
func (c *CloudClient) vmwarePowerAndWait(ctx context.Context, serverID int, action string) (*entities.VmwareServer, error) {
	task, err := c.vmwarePower(ctx, serverID, action)
	if err != nil {
		return nil, err
	}
	if err := c.awaitVmwareTask(ctx, task); err != nil {
		return nil, err
	}
	return c.GetVmwareServer(ctx, serverID)
}

// PowerOnVmwareServer powers on a server and returns the background task to await.
func (c *CloudClient) PowerOnVmwareServer(ctx context.Context, serverID int) (*VmwareTaskID, error) {
	return c.vmwarePower(ctx, serverID, "on")
}

// PowerOnVmwareServerAndWait powers on a server, waits for the task and returns
// the refreshed server.
func (c *CloudClient) PowerOnVmwareServerAndWait(ctx context.Context, serverID int) (*entities.VmwareServer, error) {
	return c.vmwarePowerAndWait(ctx, serverID, "on")
}

// PowerOffVmwareServer hard-powers off a server and returns the background task
// to await.
func (c *CloudClient) PowerOffVmwareServer(ctx context.Context, serverID int) (*VmwareTaskID, error) {
	return c.vmwarePower(ctx, serverID, "off")
}

// PowerOffVmwareServerAndWait hard-powers off a server, waits for the task and
// returns the refreshed server.
func (c *CloudClient) PowerOffVmwareServerAndWait(ctx context.Context, serverID int) (*entities.VmwareServer, error) {
	return c.vmwarePowerAndWait(ctx, serverID, "off")
}

// ShutdownVmwareServer gracefully shuts down the guest OS and returns the
// background task to await. It requires VMware Tools in the guest
// (VmwareServer.VmToolsInstalled).
func (c *CloudClient) ShutdownVmwareServer(ctx context.Context, serverID int) (*VmwareTaskID, error) {
	return c.vmwarePower(ctx, serverID, "shutdown")
}

// ShutdownVmwareServerAndWait gracefully shuts down the guest OS, waits for the
// task and returns the refreshed server.
func (c *CloudClient) ShutdownVmwareServerAndWait(ctx context.Context, serverID int) (*entities.VmwareServer, error) {
	return c.vmwarePowerAndWait(ctx, serverID, "shutdown")
}

// RebootVmwareServer gracefully reboots the guest OS and returns the background
// task to await. It requires VMware Tools in the guest.
func (c *CloudClient) RebootVmwareServer(ctx context.Context, serverID int) (*VmwareTaskID, error) {
	return c.vmwarePower(ctx, serverID, "reboot")
}

// RebootVmwareServerAndWait gracefully reboots the guest OS, waits for the task
// and returns the refreshed server.
func (c *CloudClient) RebootVmwareServerAndWait(ctx context.Context, serverID int) (*entities.VmwareServer, error) {
	return c.vmwarePowerAndWait(ctx, serverID, "reboot")
}

// ResetVmwareServer hard-resets a server and returns the background task to await.
func (c *CloudClient) ResetVmwareServer(ctx context.Context, serverID int) (*VmwareTaskID, error) {
	return c.vmwarePower(ctx, serverID, "reset")
}

// ResetVmwareServerAndWait hard-resets a server, waits for the task and returns
// the refreshed server.
func (c *CloudClient) ResetVmwareServerAndWait(ctx context.Context, serverID int) (*entities.VmwareServer, error) {
	return c.vmwarePowerAndWait(ctx, serverID, "reset")
}

// ===================== Nested hypervisor =====================

// vmwareNestedHypervisor switches nested virtualization on a server and returns
// the background task, or (nil, nil) when the server is already in the requested
// state.
//
// The action has no request body, so the server id is validated here rather than
// through a request Validate (the power actions do the same).
func (c *CloudClient) vmwareNestedHypervisor(ctx context.Context, serverID int, action string) (*VmwareTaskID, error) {
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodPost, buildVmwareServerPath(serverID, "nested-hypervisor", action), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s nested hypervisor request for vmware server %d: %w", action, serverID, err)
	}
	var task VmwareTaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to %s nested hypervisor on vmware server %d: %w", action, serverID, err)
	}
	// The API answers 200 with "task_id": null when the server is already in the
	// requested state: there is nothing to do, hence no task. Report that as
	// (nil, nil), the same way EditVmwareNetwork reports its synchronous case, so
	// callers do not await a meaningless empty task.
	if task.ID == "" {
		return nil, nil
	}
	return &task, nil
}

// vmwareNestedHypervisorAndWait switches nested virtualization, waits for the
// task if the API started one and returns the refreshed server.
func (c *CloudClient) vmwareNestedHypervisorAndWait(ctx context.Context, serverID int, action string) (*entities.VmwareServer, error) {
	task, err := c.vmwareNestedHypervisor(ctx, serverID, action)
	if err != nil {
		return nil, err
	}
	// A nil task is the idempotent case — awaitVmwareTask treats it as "nothing to
	// await", so the current server state is returned without any waiting.
	if err := c.awaitVmwareTask(ctx, task); err != nil {
		return nil, err
	}
	return c.GetVmwareServer(ctx, serverID)
}

// EnableVmwareServerNestedHypervisor enables nested virtualization on a server
// and returns the background task to await, or (nil, nil) when it is already
// enabled.
//
// The switch is applied by a backend saga that power-cycles a running server: it
// powers the guest off, changes the setting and powers it back on. A server that
// is already off stays off.
//
// It is rejected for a server with a GPU allocation
// (APICodeVmwareOperationNotSupportedForGpuServer), for a suspended server
// (APICodeVmwareServerIsSuspended) and when the VDC the server lives in does not
// support nested virtualization — see VmwareLocation.NestedHypervisorSupported for
// what the catalog reports up front.
func (c *CloudClient) EnableVmwareServerNestedHypervisor(ctx context.Context, serverID int) (*VmwareTaskID, error) {
	return c.vmwareNestedHypervisor(ctx, serverID, "enable")
}

// EnableVmwareServerNestedHypervisorAndWait enables nested virtualization, waits
// for the task and returns the refreshed server. When it was already enabled
// there is no task to wait for and the current server state is returned.
func (c *CloudClient) EnableVmwareServerNestedHypervisorAndWait(ctx context.Context, serverID int) (*entities.VmwareServer, error) {
	return c.vmwareNestedHypervisorAndWait(ctx, serverID, "enable")
}

// DisableVmwareServerNestedHypervisor disables nested virtualization on a server
// and returns the background task to await, or (nil, nil) when it is already
// disabled. Like enabling, it power-cycles a running server.
func (c *CloudClient) DisableVmwareServerNestedHypervisor(ctx context.Context, serverID int) (*VmwareTaskID, error) {
	return c.vmwareNestedHypervisor(ctx, serverID, "disable")
}

// DisableVmwareServerNestedHypervisorAndWait disables nested virtualization,
// waits for the task and returns the refreshed server. When it was already
// disabled there is no task to wait for and the current server state is returned.
func (c *CloudClient) DisableVmwareServerNestedHypervisorAndWait(ctx context.Context, serverID int) (*entities.VmwareServer, error) {
	return c.vmwareNestedHypervisorAndWait(ctx, serverID, "disable")
}

// ===================== Volumes =====================

// GetVmwareServerVolumes returns the additional data volumes of a server. A
// server without extra volumes yields an empty slice, not an error.
//
// Renamed from GetVmwareVolumeList to the sub-resource plural form used
// elsewhere in the package.
func (c *CloudClient) GetVmwareServerVolumes(ctx context.Context, serverID int) ([]*entities.VmwareVolume, error) {
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodGet, buildVmwareServerPath(serverID, "volumes"), nil)
	if err != nil {
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
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	if volumeID <= 0 {
		return nil, fmt.Errorf("volume ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodGet, buildVmwareServerPath(serverID, "volumes", strconv.Itoa(volumeID)), nil)
	if err != nil {
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

// CreateVmwareVolume creates a data volume on a server and returns the background
// task to await.
//
// DiskType is the Title of one of the location's disk types
// (VmwareLocation.DiskTypes), and SizeMB must respect that entry's
// MinMB/MaxMB/StepMB.
func (c *CloudClient) CreateVmwareVolume(ctx context.Context, serverID int, req *entities.VmwareCreateVolumeRequest) (*VmwareTaskID, error) {
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
		return nil, fmt.Errorf("failed to create volume request for vmware server %d: %w", serverID, err)
	}
	var task VmwareTaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to create volume on vmware server %d: %w", serverID, err)
	}
	return &task, nil
}

// CreateVmwareVolumeAndWait creates a data volume and waits for its task.
//
// The API does not return the new volume's id — neither the response nor the
// completed task carries it — so this reports completion only; read the volume
// back with GetVmwareServerVolumes.
func (c *CloudClient) CreateVmwareVolumeAndWait(ctx context.Context, serverID int, req *entities.VmwareCreateVolumeRequest) error {
	task, err := c.CreateVmwareVolume(ctx, serverID, req)
	if err != nil {
		return err
	}
	return c.awaitVmwareTask(ctx, task)
}

// EditVmwareVolume edits a data volume of a server and returns the background
// task to await. A volume can only grow.
func (c *CloudClient) EditVmwareVolume(ctx context.Context, serverID, volumeID int, req *entities.VmwareEditVolumeRequest) (*VmwareTaskID, error) {
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
		return nil, fmt.Errorf("failed to create edit volume %d request for vmware server %d: %w", volumeID, serverID, err)
	}
	var task VmwareTaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to edit volume %d of vmware server %d: %w", volumeID, serverID, err)
	}
	return &task, nil
}

// EditVmwareVolumeAndWait edits a data volume, waits for the task and returns the
// refreshed volume.
func (c *CloudClient) EditVmwareVolumeAndWait(ctx context.Context, serverID, volumeID int, req *entities.VmwareEditVolumeRequest) (*entities.VmwareVolume, error) {
	task, err := c.EditVmwareVolume(ctx, serverID, volumeID, req)
	if err != nil {
		return nil, err
	}
	if err := c.awaitVmwareTask(ctx, task); err != nil {
		return nil, err
	}
	return c.GetVmwareVolume(ctx, serverID, volumeID)
}

// DeleteVmwareVolume deletes a data volume of a server and returns the background
// task to await.
func (c *CloudClient) DeleteVmwareVolume(ctx context.Context, serverID, volumeID int) (*VmwareTaskID, error) {
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	if volumeID <= 0 {
		return nil, fmt.Errorf("volume ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodDelete, buildVmwareServerPath(serverID, "volumes", strconv.Itoa(volumeID)), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create delete volume %d request for vmware server %d: %w", volumeID, serverID, err)
	}
	var task VmwareTaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to delete volume %d of vmware server %d: %w", volumeID, serverID, err)
	}
	return &task, nil
}

// DeleteVmwareVolumeAndWait deletes a data volume and waits for its task.
func (c *CloudClient) DeleteVmwareVolumeAndWait(ctx context.Context, serverID, volumeID int) error {
	task, err := c.DeleteVmwareVolume(ctx, serverID, volumeID)
	if err != nil {
		return err
	}
	return c.awaitVmwareTask(ctx, task)
}

// ===================== Snapshot (single per server) =====================

// GetVmwareSnapshot returns the snapshot of a server (a server has at most one).
//
// A server with no snapshot answers HTTP 404, so IsNotFound(err) is the normal
// "no snapshot" signal here rather than an error condition.
func (c *CloudClient) GetVmwareSnapshot(ctx context.Context, serverID int) (*entities.VmwareSnapshot, error) {
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodGet, buildVmwareServerPath(serverID, "snapshot"), nil)
	if err != nil {
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

// CreateVmwareSnapshot creates the snapshot of a server and returns the
// background task to await. A server holds at most one snapshot.
//
// Takes *entities.VmwareCreateSnapshotRequest instead of a bare string, in
// line with the base CreateServerSnapshot, so a future second field is not a
// breaking change.
func (c *CloudClient) CreateVmwareSnapshot(ctx context.Context, serverID int, req *entities.VmwareCreateSnapshotRequest) (*VmwareTaskID, error) {
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
		return nil, fmt.Errorf("failed to create snapshot request for vmware server %d: %w", serverID, err)
	}
	var task VmwareTaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to create snapshot of vmware server %d: %w", serverID, err)
	}
	return &task, nil
}

// CreateVmwareSnapshotAndWait creates the snapshot, waits for the task and
// returns it.
func (c *CloudClient) CreateVmwareSnapshotAndWait(ctx context.Context, serverID int, req *entities.VmwareCreateSnapshotRequest) (*entities.VmwareSnapshot, error) {
	task, err := c.CreateVmwareSnapshot(ctx, serverID, req)
	if err != nil {
		return nil, err
	}
	if err := c.awaitVmwareTask(ctx, task); err != nil {
		return nil, err
	}
	return c.GetVmwareSnapshot(ctx, serverID)
}

// RestoreVmwareSnapshot restores a server to its snapshot and returns the
// background task to await.
func (c *CloudClient) RestoreVmwareSnapshot(ctx context.Context, serverID int) (*VmwareTaskID, error) {
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodPost, buildVmwareServerPath(serverID, "snapshot", "restore"), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create restore snapshot request for vmware server %d: %w", serverID, err)
	}
	var task VmwareTaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to restore snapshot of vmware server %d: %w", serverID, err)
	}
	return &task, nil
}

// RestoreVmwareSnapshotAndWait restores the snapshot, waits for the task and
// returns the refreshed server.
func (c *CloudClient) RestoreVmwareSnapshotAndWait(ctx context.Context, serverID int) (*entities.VmwareServer, error) {
	task, err := c.RestoreVmwareSnapshot(ctx, serverID)
	if err != nil {
		return nil, err
	}
	if err := c.awaitVmwareTask(ctx, task); err != nil {
		return nil, err
	}
	return c.GetVmwareServer(ctx, serverID)
}

// DeleteVmwareSnapshot deletes the snapshot of a server and returns the
// background task to await.
func (c *CloudClient) DeleteVmwareSnapshot(ctx context.Context, serverID int) (*VmwareTaskID, error) {
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodDelete, buildVmwareServerPath(serverID, "snapshot"), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create delete snapshot request for vmware server %d: %w", serverID, err)
	}
	var task VmwareTaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to delete snapshot of vmware server %d: %w", serverID, err)
	}
	return &task, nil
}

// DeleteVmwareSnapshotAndWait deletes the snapshot and waits for its task.
func (c *CloudClient) DeleteVmwareSnapshotAndWait(ctx context.Context, serverID int) error {
	task, err := c.DeleteVmwareSnapshot(ctx, serverID)
	if err != nil {
		return err
	}
	return c.awaitVmwareTask(ctx, task)
}

// ===================== Network interfaces =====================

// GetVmwareServerNICs returns the network interfaces of a server.
//
// Renamed from GetVmwareServerNicList to the sub-resource plural form
// (Nic -> NIC) used elsewhere in the package.
func (c *CloudClient) GetVmwareServerNICs(ctx context.Context, serverID int) ([]*entities.VmwareNIC, error) {
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodGet, buildVmwareServerPath(serverID, "nics"), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list nics request for vmware server %d: %w", serverID, err)
	}
	var resp vmwareNicsResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list nics of vmware server %d: %w", serverID, err)
	}
	return resp.NICs, nil
}

// ConnectVmwareClientNetwork attaches a server to a client network and returns
// the background task to await.
func (c *CloudClient) ConnectVmwareClientNetwork(ctx context.Context, serverID int, req *entities.VmwareConnectClientNetworkRequest) (*VmwareTaskID, error) {
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
		return nil, fmt.Errorf("failed to create connect client network request for vmware server %d: %w", serverID, err)
	}
	var task VmwareTaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to connect client network to vmware server %d: %w", serverID, err)
	}
	return &task, nil
}

// ConnectVmwareClientNetworkAndWait attaches a server to a client network, waits
// for the task and returns the server's NICs.
func (c *CloudClient) ConnectVmwareClientNetworkAndWait(ctx context.Context, serverID int, req *entities.VmwareConnectClientNetworkRequest) ([]*entities.VmwareNIC, error) {
	task, err := c.ConnectVmwareClientNetwork(ctx, serverID, req)
	if err != nil {
		return nil, err
	}
	if err := c.awaitVmwareTask(ctx, task); err != nil {
		return nil, err
	}
	return c.GetVmwareServerNICs(ctx, serverID)
}

// ConnectVmwareSharedNetwork attaches a server to a shared (public) network and
// returns the background task to await.
func (c *CloudClient) ConnectVmwareSharedNetwork(ctx context.Context, serverID int, req *entities.VmwareConnectSharedNetworkRequest) (*VmwareTaskID, error) {
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
		return nil, fmt.Errorf("failed to create connect shared network request for vmware server %d: %w", serverID, err)
	}
	var task VmwareTaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to connect shared network to vmware server %d: %w", serverID, err)
	}
	return &task, nil
}

// ConnectVmwareSharedNetworkAndWait attaches a server to a shared network, waits
// for the task and returns the server's NICs.
func (c *CloudClient) ConnectVmwareSharedNetworkAndWait(ctx context.Context, serverID int, req *entities.VmwareConnectSharedNetworkRequest) ([]*entities.VmwareNIC, error) {
	task, err := c.ConnectVmwareSharedNetwork(ctx, serverID, req)
	if err != nil {
		return nil, err
	}
	if err := c.awaitVmwareTask(ctx, task); err != nil {
		return nil, err
	}
	return c.GetVmwareServerNICs(ctx, serverID)
}

// UpdateVmwareNIC updates a network interface of a server and returns the
// background task to await.
//
// req.NetworkID is mandatory — the update replaces the NIC's placement, so pass the
// NIC's current network to keep it there. req.BandwidthMbps only applies to a NIC on
// a shared/public network and req.IP only to a network with DHCP disabled; see
// entities.VmwareUpdateNICRequest for what the API rejects.
func (c *CloudClient) UpdateVmwareNIC(ctx context.Context, serverID, nicID int, req *entities.VmwareUpdateNICRequest) (*VmwareTaskID, error) {
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
		return nil, fmt.Errorf("failed to create update nic %d request for vmware server %d: %w", nicID, serverID, err)
	}
	var task VmwareTaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to update nic %d of vmware server %d: %w", nicID, serverID, err)
	}
	return &task, nil
}

// UpdateVmwareNICAndWait updates a NIC, waits for the task and returns the
// server's NICs.
func (c *CloudClient) UpdateVmwareNICAndWait(ctx context.Context, serverID, nicID int, req *entities.VmwareUpdateNICRequest) ([]*entities.VmwareNIC, error) {
	task, err := c.UpdateVmwareNIC(ctx, serverID, nicID, req)
	if err != nil {
		return nil, err
	}
	if err := c.awaitVmwareTask(ctx, task); err != nil {
		return nil, err
	}
	return c.GetVmwareServerNICs(ctx, serverID)
}

// DeleteVmwareNIC detaches a network interface from a server and returns the
// background task to await.
func (c *CloudClient) DeleteVmwareNIC(ctx context.Context, serverID, nicID int) (*VmwareTaskID, error) {
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	if nicID <= 0 {
		return nil, fmt.Errorf("NIC ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodDelete, buildVmwareServerPath(serverID, "nics", strconv.Itoa(nicID)), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create delete nic %d request for vmware server %d: %w", nicID, serverID, err)
	}
	var task VmwareTaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to delete nic %d of vmware server %d: %w", nicID, serverID, err)
	}
	return &task, nil
}

// DeleteVmwareNICAndWait detaches a NIC and waits for its task.
func (c *CloudClient) DeleteVmwareNICAndWait(ctx context.Context, serverID, nicID int) error {
	task, err := c.DeleteVmwareNIC(ctx, serverID, nicID)
	if err != nil {
		return err
	}
	return c.awaitVmwareTask(ctx, task)
}

// ===================== Server firewall =====================

// GetVmwareServerFirewall returns the firewall rules of a server.
//
// A server with no rules answers 200 with an empty set, so a 404
// here means only "no such server".
//
// The returned rules are the same type UpdateVmwareServerFirewall accepts, so
// they can be modified and written straight back.
func (c *CloudClient) GetVmwareServerFirewall(ctx context.Context, serverID int) ([]entities.VmwareServerFirewallRule, error) {
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	req, err := c.newRequest(ctx, http.MethodGet, buildVmwareServerPath(serverID, "firewall"), nil)
	if err != nil {
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
// A non-empty rule set works — each rule must carry name,
// traffic_direction, action and protocol (enforced by Validate). An empty rule set
// clears the firewall.
//
// A (nil, nil) result means the API answered without starting a task, so there is
// nothing to await — it is not an error. Both shapes are handled: a no-task answer
// is reported as a nil *VmwareTaskID rather than a &VmwareTaskID{ID: ""} that would
// break a wait.
func (c *CloudClient) UpdateVmwareServerFirewall(ctx context.Context, serverID int, req *entities.VmwareUpdateServerFirewallRequest) (*VmwareTaskID, error) {
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
		return nil, fmt.Errorf("failed to create update firewall request for vmware server %d: %w", serverID, err)
	}
	var task VmwareTaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to update firewall of vmware server %d: %w", serverID, err)
	}
	// Synchronous response (empty body) leaves task_id empty; report it as
	// "no task" rather than a &VmwareTaskID{ID: ""} that would break the wait.
	if task.ID == "" {
		return nil, nil
	}
	return &task, nil
}

// UpdateVmwareServerFirewallAndWait replaces the firewall rule set, waits for the
// task (if any) and returns the resulting rules.
func (c *CloudClient) UpdateVmwareServerFirewallAndWait(ctx context.Context, serverID int, req *entities.VmwareUpdateServerFirewallRequest) ([]entities.VmwareServerFirewallRule, error) {
	task, err := c.UpdateVmwareServerFirewall(ctx, serverID, req)
	if err != nil {
		return nil, err
	}
	// A nil task is the documented synchronous case (clearing the rules).
	if err := c.awaitVmwareTask(ctx, task); err != nil {
		return nil, err
	}
	return c.GetVmwareServerFirewall(ctx, serverID)
}
