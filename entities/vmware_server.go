package entities

import "fmt"

// VmwareServer state constants. These are the values returned in
// VmwareServer.State and form the public state enum.
const (
	// VmwareServerStateCreating - the server is being provisioned.
	VmwareServerStateCreating = "creating"
	// VmwareServerStateActive - the server is running.
	VmwareServerStateActive = "active"
	// VmwareServerStateSuspended - the server is suspended.
	VmwareServerStateSuspended = "suspended"
	// VmwareServerStatePoweredOff - the server is powered off.
	VmwareServerStatePoweredOff = "powered_off"
	// VmwareServerStateNeedMoney - the server is blocked pending payment.
	VmwareServerStateNeedMoney = "need_money"
	// VmwareServerStateDeleting - the server is being deleted.
	VmwareServerStateDeleting = "deleting"
	// VmwareServerStateError - the server is in an error state.
	VmwareServerStateError = "error"
)

// VmwareGPU represents the GPU allocation attached to a server.
type VmwareGPU struct {
	ModelID   int `json:"model_id"`
	VramMB    int `json:"vram_mb"`
	CardCount int `json:"card_count"`
}

// VmwareServer represents a VMware server instance.
type VmwareServer struct {
	ID             int     `json:"id"`
	ProjectID      int     `json:"project_id"`
	LocationID     int     `json:"location_id"`
	Name           string  `json:"name"`
	ComputerName   *string `json:"computer_name,omitempty"`
	State          string  `json:"state"`
	CPU            int     `json:"cpu"`
	RamMB          int     `json:"ram_mb"`
	SystemDiskMB   int     `json:"system_disk_mb"`
	SystemDiskType *string `json:"system_disk_type,omitempty"`
	ImageID        int     `json:"image_id"`
	IsPowerOn      bool    `json:"is_power_on"`
	// VmToolsInstalled is live-only: it is populated only by Get-by-id and is
	// absent from list responses.
	VmToolsInstalled *bool       `json:"vm_tools_installed,omitempty"`
	GPU              *VmwareGPU  `json:"gpu,omitempty"`
	NICs             []VmwareNIC `json:"nics"`
	Created          string      `json:"created"`
}

// VmwareGPURequest specifies the GPU allocation for a server order.
//
// Although VramMB and CardCount are pointers with omitempty (suggesting they
// are optional), all three fields are in fact required - the backend looks up a
// slicing policy by the exact triple (gpu_model_id, vram_mb, card_count) and the
// order fails without them. The pointer types are kept, but
// Validate requires them to be non-nil.
type VmwareGPURequest struct {
	GPUModelID int  `json:"gpu_model_id"`
	VramMB     *int `json:"vram_mb,omitempty"`
	CardCount  *int `json:"card_count,omitempty"`
}

// Validate checks the GPU request: the full triple is mandatory.
func (r *VmwareGPURequest) Validate() error {
	if r.GPUModelID <= 0 {
		return fmt.Errorf("gpu_model_id is required")
	}
	if r.VramMB == nil {
		return fmt.Errorf("vram_mb is required")
	}
	if r.CardCount == nil {
		return fmt.Errorf("card_count is required")
	}
	return nil
}

// VmwareCreateServerRequest represents a request to create a VMware server.
type VmwareCreateServerRequest struct {
	LocationID           int               `json:"location_id"`
	Name                 string            `json:"name"`
	ComputerName         string            `json:"computer_name,omitempty"`
	ImageID              int               `json:"image_id"`
	CPUCount             int               `json:"cpu_count"`
	RamMB                int               `json:"ram_mb"`
	SystemDiskSizeMB     int               `json:"system_disk_size_mb"`
	SystemDiskType       string            `json:"system_disk_type,omitempty"`
	PublicNetworkID      *int              `json:"public_network_id,omitempty"`
	NetworkBandwidthMbps *int              `json:"network_bandwidth_mbps,omitempty"`
	BackupEnabled        *bool             `json:"backup_enabled,omitempty"`
	BackupPeriod         *int              `json:"backup_period,omitempty"`
	SSHKeys              []int             `json:"ssh_keys,omitempty"`
	NeedSysprep          *bool             `json:"need_sysprep,omitempty"`
	GPU                  *VmwareGPURequest `json:"gpu,omitempty"`
}

// Validate checks the create server request.
func (r *VmwareCreateServerRequest) Validate() error {
	if r.LocationID <= 0 {
		return fmt.Errorf("location_id is required")
	}
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if r.ImageID <= 0 {
		return fmt.Errorf("image_id is required")
	}
	if r.CPUCount <= 0 {
		return fmt.Errorf("cpu_count must be greater than 0")
	}
	if r.RamMB <= 0 {
		return fmt.Errorf("ram_mb must be greater than 0")
	}
	if r.SystemDiskSizeMB <= 0 {
		return fmt.Errorf("system_disk_size_mb must be greater than 0")
	}
	if r.GPU != nil {
		if err := r.GPU.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// VmwareServerOrder is the response of create/copy/rebuild: the id of the created
// server plus the task_id of the background operation.
type VmwareServerOrder struct {
	ServerID int    `json:"server_id"`
	TaskID   string `json:"task_id"`
}

// VmwareChangeConfigurationRequest represents a request to change server resources.
type VmwareChangeConfigurationRequest struct {
	CPU              int `json:"cpu"`
	RamMB            int `json:"ram_mb"`
	SystemDiskSizeMB int `json:"system_disk_size_mb"`
}

// Validate checks the change configuration request.
func (r *VmwareChangeConfigurationRequest) Validate() error {
	if r.CPU <= 0 || r.RamMB <= 0 || r.SystemDiskSizeMB <= 0 {
		return fmt.Errorf("cpu, ram_mb and system_disk_size_mb must be greater than 0")
	}
	return nil
}

// VmwareRenameServerRequest represents a request to rename a server.
type VmwareRenameServerRequest struct {
	Name string `json:"name"`
}

// Validate checks the rename server request.
func (r *VmwareRenameServerRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

// VmwareComputerNameRequest represents a request to change the guest computer name.
type VmwareComputerNameRequest struct {
	ComputerName       string `json:"computer_name"`
	ForceCustomization *bool  `json:"force_customization,omitempty"`
}

// Validate checks the computer name request.
func (r *VmwareComputerNameRequest) Validate() error {
	if r.ComputerName == "" {
		return fmt.Errorf("computer_name is required")
	}
	return nil
}

// VmwareCopyServerRequest represents a request to copy a server.
type VmwareCopyServerRequest struct {
	Name            string `json:"name"`
	ClientNetworkID *int   `json:"client_network_id,omitempty"`
}

// Validate checks the copy server request.
func (r *VmwareCopyServerRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

// VmwareRebuildServerRequest represents a request to rebuild a server from an image.
type VmwareRebuildServerRequest struct {
	ImageID     int   `json:"image_id"`
	NeedSysprep *bool `json:"need_sysprep,omitempty"`
}

// Validate checks the rebuild server request.
func (r *VmwareRebuildServerRequest) Validate() error {
	if r.ImageID <= 0 {
		return fmt.Errorf("image_id is required")
	}
	return nil
}

// ===================== Volumes =====================

// VmwareVolume represents an additional data volume attached to a server.
type VmwareVolume struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	SizeMB   int     `json:"size_mb"`
	DiskType *string `json:"disk_type,omitempty"`
}

// VmwareCreateVolumeRequest represents a request to create a volume.
type VmwareCreateVolumeRequest struct {
	Name     string `json:"name"`
	DiskType string `json:"disk_type"`
	SizeMB   int    `json:"size_mb"`
}

// Validate checks the create volume request.
func (r *VmwareCreateVolumeRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if r.DiskType == "" {
		return fmt.Errorf("disk_type is required")
	}
	if r.SizeMB <= 0 {
		return fmt.Errorf("size_mb must be greater than 0")
	}
	return nil
}

// VmwareEditVolumeRequest represents a request to edit a volume.
type VmwareEditVolumeRequest struct {
	Name   string `json:"name,omitempty"`
	SizeMB int    `json:"size_mb"`
}

// Validate checks the edit volume request.
func (r *VmwareEditVolumeRequest) Validate() error {
	if r.SizeMB <= 0 {
		return fmt.Errorf("size_mb must be greater than 0")
	}
	return nil
}

// ===================== Snapshot =====================

// VmwareSnapshot represents a server snapshot.
type VmwareSnapshot struct {
	Name    string `json:"name"`
	Created string `json:"created"`
}

// VmwareCreateSnapshotRequest represents a request to create a snapshot.
type VmwareCreateSnapshotRequest struct {
	Name string `json:"name"`
}

// Validate checks the create snapshot request.
func (r *VmwareCreateSnapshotRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

// ===================== Network interfaces =====================

// VmwareNIC represents a network interface attached to a server.
type VmwareNIC struct {
	ID            int     `json:"id"`
	Number        int     `json:"number"`
	IsPrimary     bool    `json:"is_primary"`
	NetworkID     int     `json:"network_id"`
	IP            *string `json:"ip,omitempty"`
	Mac           string  `json:"mac"`
	BandwidthMbps int     `json:"bandwidth_mbps"` // persisted NIC bandwidth, also returned on read
}

// VmwareConnectClientNetworkRequest represents a request to attach a server to a
// client network.
type VmwareConnectClientNetworkRequest struct {
	NetworkID          int    `json:"network_id"`
	IP                 string `json:"ip,omitempty"`
	ForceCustomization *bool  `json:"force_customization,omitempty"`
}

// Validate checks the connect client network request.
func (r *VmwareConnectClientNetworkRequest) Validate() error {
	if r.NetworkID <= 0 {
		return fmt.Errorf("network_id is required")
	}
	return nil
}

// VmwareConnectSharedNetworkRequest represents a request to attach a server to a
// shared (public) network.
type VmwareConnectSharedNetworkRequest struct {
	IsIPv6             *bool `json:"is_ipv6,omitempty"`
	BandwidthMbps      int   `json:"bandwidth_mbps"`
	ForceCustomization *bool `json:"force_customization,omitempty"`
}

// Validate checks the connect shared network request.
func (r *VmwareConnectSharedNetworkRequest) Validate() error {
	if r.BandwidthMbps <= 0 {
		return fmt.Errorf("bandwidth_mbps must be greater than 0")
	}
	return nil
}

// VmwareUpdateNICRequest represents a request to update a server network interface.
//
// The two optional fields apply to different kinds of NIC, and sending the wrong
// one is a 400 rather than a no-op:
//
//   - BandwidthMbps is only accepted for a NIC on a shared/public network. On a
//     client-network NIC (one attached to an isolated or routed network) it is
//     refused with "The network bandwidth cannot be changed" (-8094).
//   - IP is only accepted when the target network has DHCP disabled; otherwise the
//     API answers -8111 "Cannot set an IP address for a network with DHCP enabled".
//
// NetworkID is mandatory in every case: the update replaces the NIC's placement, so
// pass the NIC's current network to keep it where it is.
type VmwareUpdateNICRequest struct {
	NetworkID int `json:"network_id"`
	// BandwidthMbps applies to shared/public-network NICs only — see above.
	BandwidthMbps *int `json:"bandwidth_mbps,omitempty"`
	// IP requires the target network to have DHCP disabled — see above.
	IP                 string `json:"ip,omitempty"`
	ForceCustomization *bool  `json:"force_customization,omitempty"`
}

// Validate checks the update NIC request.
func (r *VmwareUpdateNICRequest) Validate() error {
	if r.NetworkID <= 0 {
		return fmt.Errorf("network_id is required")
	}
	if r.BandwidthMbps != nil && *r.BandwidthMbps <= 0 {
		return fmt.Errorf("bandwidth_mbps must be greater than 0 when set")
	}
	return nil
}

// ===================== Server firewall =====================

// Server firewall traffic directions.
//
// The API accepts these case-insensitively and returns them lower
// case, so writing the constants round-trips exactly. Anything outside the enum is
// rejected with HTTP 400 and code -2002 ("The request body is not formatted or not
// specified"), which names neither the field nor the offending value - hence the
// constants.
const (
	// VmwareTrafficDirectionIncoming filters traffic entering the server.
	VmwareTrafficDirectionIncoming = "incoming"
	// VmwareTrafficDirectionOutgoing filters traffic leaving the server.
	VmwareTrafficDirectionOutgoing = "outgoing"
)

// Server firewall actions.
const (
	// VmwareFirewallActionAllow lets matching traffic through.
	VmwareFirewallActionAllow = "allow"
	// VmwareFirewallActionDeny drops matching traffic.
	VmwareFirewallActionDeny = "deny"
)

// VmwareFirewallProtocolAny matches every protocol in a server firewall rule.
// Alongside it the usual named protocols ("tcp", "udp", "icmp") are accepted.
const VmwareFirewallProtocolAny = "any"

// VmwareFirewallAny is the wildcard accepted by the source/destination and port
// fields of a server firewall rule.
const VmwareFirewallAny = "any"

// VmwareServerFirewallRule represents a single server firewall rule. The same
// type is used for read and write: the API round-trips it field for field.
//
// name and traffic_direction are mandatory on the backend and are
// now exposed by the public API, so a full rule set can be both read and
// written. Use the VmwareTrafficDirection* and VmwareFirewallAction* constants.
type VmwareServerFirewallRule struct {
	Name             string  `json:"name"`
	TrafficDirection string  `json:"traffic_direction"`
	Action           string  `json:"action"`
	Protocol         string  `json:"protocol"`
	Source           *string `json:"source,omitempty"`
	SourcePort       *string `json:"source_port,omitempty"`
	Destination      *string `json:"destination,omitempty"`
	DestinationPort  *string `json:"destination_port,omitempty"`
}

// VmwareUpdateServerFirewallRequest replaces the full server firewall rule set.
type VmwareUpdateServerFirewallRequest struct {
	Rules []VmwareServerFirewallRule `json:"rules"`
}

// Validate checks the update server firewall request. An empty rule set is
// allowed (it clears the firewall); every provided rule must carry name,
// traffic_direction, action and protocol — all mandatory on the backend.
func (r *VmwareUpdateServerFirewallRequest) Validate() error {
	for i, rule := range r.Rules {
		if rule.Name == "" {
			return fmt.Errorf("rules[%d]: name is required", i)
		}
		if rule.TrafficDirection == "" {
			return fmt.Errorf("rules[%d]: traffic_direction is required", i)
		}
		if rule.Action == "" {
			return fmt.Errorf("rules[%d]: action is required", i)
		}
		// protocol is a non-pointer required field on the backend.
		if rule.Protocol == "" {
			return fmt.Errorf("rules[%d]: protocol is required", i)
		}
	}
	return nil
}

// ===================== Rule set semantics =====================
//
// VmwareUpdateServerFirewallRequest replaces the whole rule set, so a
// read-modify-write must resend every rule it wants to keep. The read type is the
// same VmwareServerFirewallRule, which makes the round trip a direct copy:
//
//	rules, err := client.GetVmwareServerFirewall(ctx, serverID)
//	// append / edit, then
//	_, err = client.UpdateVmwareServerFirewall(ctx, serverID,
//		&entities.VmwareUpdateServerFirewallRequest{Rules: rules})
