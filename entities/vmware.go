package entities

import "fmt"

// Сущности услуги VMware (Public API, префикс /api/v1/vmware/**).
// Идентификаторы — плоские int; состояния и enum'ы — строками (snake_case).

// ===================== Метаинформация (lookup) =====================

type VmwareLocation struct {
	ID           int    `json:"id"`
	TechTitle    string `json:"tech_title"`
	GpuSupported bool   `json:"gpu_supported"`
}

type VmwareImage struct {
	ID                   int    `json:"id"`
	Name                 string `json:"name"`
	OsFamily             string `json:"os_family"`
	OsType               string `json:"os_type"`
	MinRamMB             int    `json:"min_ram_mb"`
	HddGB                int    `json:"hdd_gb"`
	SSHKeySupported      bool   `json:"ssh_key_supported"`
	CPUHotAdd            bool   `json:"cpu_hot_add"`
	MemoryHotAdd         bool   `json:"memory_hot_add"`
	NicHotRemove         bool   `json:"nic_hot_remove"`
	IsGpuOnly            bool   `json:"is_gpu_only"`
	SupportedGpuModelIDs []int  `json:"supported_gpu_model_ids"`
}

type VmwareDiskType struct {
	ID                     int    `json:"id"`
	Title                  string `json:"title"`
	MinGB                  int    `json:"min_gb"`
	MaxGB                  int    `json:"max_gb"`
	StepGB                 int    `json:"step_gb"`
	StartValueGB           int    `json:"start_value_gb"`
	IsAllowedForSystemDisk bool   `json:"is_allowed_for_system_disk"`
	IsSSD                  bool   `json:"is_ssd"`
}

type VmwareStorageProfile struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	DiskTypeID  int    `json:"disk_type_id"`
	IsDefault   bool   `json:"is_default"`
	FreeSpaceGB int    `json:"free_space_gb"`
}

type VmwareGpuModel struct {
	ID                    int    `json:"id"`
	TechTitle             string `json:"tech_title"`
	Name                  string `json:"name"`
	CapacityVramMB        int    `json:"capacity_vram_mb"`
	GpuCardCount          int    `json:"gpu_card_count"`
	ServerAllocationLimit int    `json:"server_allocation_limit"`
	MaxServerRamMB        *int   `json:"max_server_ram_mb,omitempty"`
	IsAvailable           *bool  `json:"is_available,omitempty"`
}

// ===================== Серверы =====================

type VmwareGpu struct {
	ModelID   int `json:"model_id"`
	VramMB    int `json:"vram_mb"`
	CardCount int `json:"card_count"`
}

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
	// live-only: заполняется только в Get-by-id (в списке отсутствует).
	VmToolsInstalled *bool       `json:"vm_tools_installed,omitempty"`
	Gpu              *VmwareGpu  `json:"gpu,omitempty"`
	Nics             []VmwareNic `json:"nics"`
	Created          string      `json:"created"`
}

type VmwareGpuRequest struct {
	GpuModelID int  `json:"gpu_model_id"`
	VramMB     *int `json:"vram_mb,omitempty"`
	CardCount  *int `json:"card_count,omitempty"`
}

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
	Gpu                  *VmwareGpuRequest `json:"gpu,omitempty"`
}

// VmwareServerOrder — ответ create/copy/rebuild: id созданного сервера + task_id.
type VmwareServerOrder struct {
	ServerID int    `json:"server_id"`
	TaskID   string `json:"task_id"`
}

type VmwareChangeConfigurationRequest struct {
	CPU              int `json:"cpu"`
	RamMB            int `json:"ram_mb"`
	SystemDiskSizeMB int `json:"system_disk_size_mb"`
}

type VmwareRenameServerRequest struct {
	Name string `json:"name"`
}

type VmwareComputerNameRequest struct {
	ComputerName       string `json:"computer_name"`
	ForceCustomization *bool  `json:"force_customization,omitempty"`
}

type VmwareCopyServerRequest struct {
	Name            string `json:"name"`
	ClientNetworkID *int   `json:"client_network_id,omitempty"`
}

type VmwareRebuildServerRequest struct {
	ImageID     int   `json:"image_id"`
	NeedSysprep *bool `json:"need_sysprep,omitempty"`
}

// ===================== Диски =====================

type VmwareVolume struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	SizeMB   int     `json:"size_mb"`
	DiskType *string `json:"disk_type,omitempty"`
}

type VmwareCreateVolumeRequest struct {
	Name     string `json:"name"`
	DiskType string `json:"disk_type"`
	SizeMB   int    `json:"size_mb"`
}

type VmwareEditVolumeRequest struct {
	Name   string `json:"name,omitempty"`
	SizeMB int    `json:"size_mb"`
}

// ===================== Снимок =====================

type VmwareSnapshot struct {
	Name    string `json:"name"`
	Created string `json:"created"`
}

type VmwareCreateSnapshotRequest struct {
	Name string `json:"name"`
}

// ===================== Сетевые интерфейсы =====================

type VmwareNic struct {
	ID        int     `json:"id"`
	Number    int     `json:"number"`
	IsPrimary bool    `json:"is_primary"`
	NetworkID int     `json:"network_id"`
	IP        *string `json:"ip,omitempty"`
	Mac       string  `json:"mac"`
}

type VmwareConnectClientNetworkRequest struct {
	NetworkID          int    `json:"network_id"`
	IP                 string `json:"ip,omitempty"`
	ForceCustomization *bool  `json:"force_customization,omitempty"`
}

type VmwareConnectSharedNetworkRequest struct {
	IsIPv6             *bool `json:"is_ipv6,omitempty"`
	BandwidthMbps      int   `json:"bandwidth_mbps"`
	ForceCustomization *bool `json:"force_customization,omitempty"`
}

type VmwareUpdateNicRequest struct {
	NetworkID          int    `json:"network_id"`
	BandwidthMbps      *int   `json:"bandwidth_mbps,omitempty"`
	IP                 string `json:"ip,omitempty"`
	ForceCustomization *bool  `json:"force_customization,omitempty"`
}

// ===================== Firewall сервера =====================

type VmwareServerFirewallRule struct {
	Action          string  `json:"action"`
	Protocol        string  `json:"protocol"`
	Source          *string `json:"source,omitempty"`
	SourcePort      *string `json:"source_port,omitempty"`
	Destination     *string `json:"destination,omitempty"`
	DestinationPort *string `json:"destination_port,omitempty"`
}

type VmwareUpdateServerFirewallRequest struct {
	Rules []VmwareServerFirewallRule `json:"rules"`
}

// ===================== Сети =====================

type VmwareNetwork struct {
	ID            int     `json:"id"`
	LocationID    int     `json:"location_id"`
	Type          string  `json:"type"`
	Name          string  `json:"name"`
	Address       *string `json:"address,omitempty"`
	Mask          *int    `json:"mask,omitempty"`
	Gateway       *string `json:"gateway,omitempty"`
	BandwidthMbps *int    `json:"bandwidth_mbps,omitempty"`
	IsDhcp        *bool   `json:"is_dhcp,omitempty"`
	Shared        *bool   `json:"shared,omitempty"`
	State         string  `json:"state"`
	NicsCount     int     `json:"nics_count"`
}

type VmwareCreateIsolatedNetworkRequest struct {
	LocationID int    `json:"location_id"`
	Name       string `json:"name"`
	Address    string `json:"address"`
	Mask       *int   `json:"mask,omitempty"`
	EnableDhcp *bool  `json:"enable_dhcp,omitempty"`
}

type VmwareCreateRoutedNetworkRequest struct {
	LocationID    int    `json:"location_id"`
	Name          string `json:"name"`
	Address       string `json:"address"`
	Mask          *int   `json:"mask,omitempty"`
	EnableDhcp    *bool  `json:"enable_dhcp,omitempty"`
	BandwidthMbps *int   `json:"bandwidth_mbps,omitempty"`
}

type VmwareCreatePublicNetworkRequest struct {
	LocationID    int    `json:"location_id"`
	Name          string `json:"name"`
	Capacity      string `json:"capacity"`
	BandwidthMbps *int   `json:"bandwidth_mbps,omitempty"`
}

type VmwareEditNetworkRequest struct {
	Name          string `json:"name,omitempty"`
	BandwidthMbps *int   `json:"bandwidth_mbps,omitempty"`
}

type VmwareConnectServerNic struct {
	ServerID int    `json:"server_id"`
	IP       string `json:"ip,omitempty"`
}

type VmwareConnectServersRequest struct {
	Nics               []VmwareConnectServerNic `json:"nics"`
	ForceCustomization *bool                    `json:"force_customization,omitempty"`
}

// ===================== Edge (на маршрутизируемой сети) =====================

type VmwareEdgeFirewallApplication struct {
	Protocol        *string `json:"protocol,omitempty"`
	SourcePort      *string `json:"source_port,omitempty"`
	DestinationPort *string `json:"destination_port,omitempty"`
}

type VmwareEdgeFirewallRule struct {
	ID           *string                         `json:"id,omitempty"`
	Enabled      *bool                           `json:"enabled,omitempty"`
	Name         *string                         `json:"name,omitempty"`
	Description  *string                         `json:"description,omitempty"`
	Action       *string                         `json:"action,omitempty"`
	Source       *string                         `json:"source,omitempty"`
	Destination  *string                         `json:"destination,omitempty"`
	Applications []VmwareEdgeFirewallApplication `json:"applications,omitempty"`
}

type VmwareEdgeFirewall struct {
	Enabled       *bool                    `json:"enabled,omitempty"`
	DefaultAction *string                  `json:"default_action,omitempty"`
	Rules         []VmwareEdgeFirewallRule `json:"rules"`
}

type VmwareUpdateEdgeFirewallRule struct {
	Name            *string `json:"name,omitempty"`
	Action          string  `json:"action"`
	Protocol        *string `json:"protocol,omitempty"`
	Source          *string `json:"source,omitempty"`
	SourcePort      *string `json:"source_port,omitempty"`
	Destination     *string `json:"destination,omitempty"`
	DestinationPort *string `json:"destination_port,omitempty"`
}

type VmwareUpdateEdgeFirewallRequest struct {
	Enabled       *bool                          `json:"enabled,omitempty"`
	DefaultAction *string                        `json:"default_action,omitempty"`
	Rules         []VmwareUpdateEdgeFirewallRule `json:"rules"`
}

type VmwareEdgeNatRule struct {
	ID             *string `json:"id,omitempty"`
	Description    *string `json:"description,omitempty"`
	Type           *string `json:"type,omitempty"`
	OriginalIP     *string `json:"original_ip,omitempty"`
	TranslatedIP   *string `json:"translated_ip,omitempty"`
	Protocol       *string `json:"protocol,omitempty"`
	OriginalPort   *string `json:"original_port,omitempty"`
	TranslatedPort *string `json:"translated_port,omitempty"`
	Enabled        *bool   `json:"enabled,omitempty"`
}

type VmwareEdgeNat struct {
	Rules []VmwareEdgeNatRule `json:"rules"`
}

type VmwareUpsertNatRuleRequest struct {
	RuleID         *int   `json:"rule_id,omitempty"`
	Type           string `json:"type"`
	Description    string `json:"description,omitempty"`
	Protocol       string `json:"protocol"`
	OriginalIP     string `json:"original_ip"`
	OriginalPort   string `json:"original_port,omitempty"`
	TranslatedIP   string `json:"translated_ip"`
	TranslatedPort string `json:"translated_port,omitempty"`
	Enabled        *bool  `json:"enabled,omitempty"`
}

type VmwareEdgeVpnTunnel struct {
	ID                    *string  `json:"id,omitempty"`
	Enabled               *bool    `json:"enabled,omitempty"`
	Name                  *string  `json:"name,omitempty"`
	Description           *string  `json:"description,omitempty"`
	LocalID               *string  `json:"local_id,omitempty"`
	LocalIP               *string  `json:"local_ip,omitempty"`
	LocalSubnets          []string `json:"local_subnets,omitempty"`
	PeerIdentificator     *string  `json:"peer_identificator,omitempty"`
	PeerEndpoint          *string  `json:"peer_endpoint,omitempty"`
	PeerSubnets           []string `json:"peer_subnets,omitempty"`
	Mtu                   *int     `json:"mtu,omitempty"`
	PerfectForwardSecrecy *bool    `json:"perfect_forward_secrecy,omitempty"`
	EncryptionType        *string  `json:"encryption_type,omitempty"`
	DigestAlgorithm       *string  `json:"digest_algorithm,omitempty"`
	DiffieHellmanGroup    *string  `json:"diffie_hellman_group,omitempty"`
}

type VmwareEdgeVpn struct {
	Enabled *bool                 `json:"enabled,omitempty"`
	Tunnels []VmwareEdgeVpnTunnel `json:"tunnels"`
}

type VmwareUpsertVpnTunnelRequest struct {
	TunnelID              *int   `json:"tunnel_id,omitempty"`
	Name                  string `json:"name"`
	Enabled               *bool  `json:"enabled,omitempty"`
	Mtu                   *int   `json:"mtu,omitempty"`
	EncryptionType        string `json:"encryption_type,omitempty"`
	SharedKey             string `json:"shared_key"`
	PeerNetwork           string `json:"peer_network"`
	PeerEndpoint          string `json:"peer_endpoint"`
	PeerIdentificator     string `json:"peer_identificator"`
	PerfectForwardSecrecy *bool  `json:"perfect_forward_secrecy,omitempty"`
	DiffieHellmanGroup    string `json:"diffie_hellman_group,omitempty"`
}

type VmwareEdgeBandwidthRequest struct {
	BandwidthMbps int `json:"bandwidth_mbps"`
}

// ===================== Задачи =====================

const (
	VmwareTaskStateNew        = "new"
	VmwareTaskStateInProgress = "in_progress"
	VmwareTaskStateCompleted  = "completed"
	VmwareTaskStateFailed     = "failed"
	VmwareTaskStateCanceled   = "canceled"
)

type VmwareTask struct {
	ID              string  `json:"id"`
	Type            string  `json:"type"`
	State           string  `json:"state"`
	ProgressPercent *int    `json:"progress_percent,omitempty"`
	ServerID        *int    `json:"server_id,omitempty"`
	NetworkID       *int    `json:"network_id,omitempty"`
	Created         string  `json:"created"`
	Completed       *string `json:"completed,omitempty"`
	Error           *string `json:"error,omitempty"`
}

func (t *VmwareTask) IsCompleted() bool { return t.State == VmwareTaskStateCompleted }
func (t *VmwareTask) IsFailed() bool    { return t.State == VmwareTaskStateFailed }
func (t *VmwareTask) IsTerminal() bool {
	return t.State == VmwareTaskStateCompleted || t.State == VmwareTaskStateFailed || t.State == VmwareTaskStateCanceled
}

// ===================== Валидация запросов =====================

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
	return nil
}

func (r *VmwareChangeConfigurationRequest) Validate() error {
	if r.CPU <= 0 || r.RamMB <= 0 || r.SystemDiskSizeMB <= 0 {
		return fmt.Errorf("cpu, ram_mb and system_disk_size_mb must be greater than 0")
	}
	return nil
}

func (r *VmwareRenameServerRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

func (r *VmwareComputerNameRequest) Validate() error {
	if r.ComputerName == "" {
		return fmt.Errorf("computer_name is required")
	}
	return nil
}

func (r *VmwareCopyServerRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

func (r *VmwareRebuildServerRequest) Validate() error {
	if r.ImageID <= 0 {
		return fmt.Errorf("image_id is required")
	}
	return nil
}

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

func (r *VmwareEditVolumeRequest) Validate() error {
	if r.SizeMB <= 0 {
		return fmt.Errorf("size_mb must be greater than 0")
	}
	return nil
}

func (r *VmwareCreateSnapshotRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

func (r *VmwareConnectClientNetworkRequest) Validate() error {
	if r.NetworkID <= 0 {
		return fmt.Errorf("network_id is required")
	}
	return nil
}

func (r *VmwareConnectSharedNetworkRequest) Validate() error {
	if r.BandwidthMbps <= 0 {
		return fmt.Errorf("bandwidth_mbps must be greater than 0")
	}
	return nil
}

func (r *VmwareUpdateNicRequest) Validate() error {
	if r.NetworkID <= 0 {
		return fmt.Errorf("network_id is required")
	}
	return nil
}

func (r *VmwareCreateIsolatedNetworkRequest) Validate() error {
	if r.LocationID <= 0 {
		return fmt.Errorf("location_id is required")
	}
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if r.Address == "" {
		return fmt.Errorf("address is required")
	}
	return nil
}

func (r *VmwareCreateRoutedNetworkRequest) Validate() error {
	if r.LocationID <= 0 {
		return fmt.Errorf("location_id is required")
	}
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if r.Address == "" {
		return fmt.Errorf("address is required")
	}
	return nil
}

func (r *VmwareCreatePublicNetworkRequest) Validate() error {
	if r.LocationID <= 0 {
		return fmt.Errorf("location_id is required")
	}
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if r.Capacity == "" {
		return fmt.Errorf("capacity is required")
	}
	return nil
}

func (r *VmwareConnectServersRequest) Validate() error {
	if len(r.Nics) == 0 {
		return fmt.Errorf("at least one nic is required")
	}
	for i, n := range r.Nics {
		if n.ServerID <= 0 {
			return fmt.Errorf("nics[%d]: server_id is required", i)
		}
	}
	return nil
}

func (r *VmwareUpsertNatRuleRequest) Validate() error {
	if r.Type == "" {
		return fmt.Errorf("type is required")
	}
	if r.Protocol == "" {
		return fmt.Errorf("protocol is required")
	}
	if r.OriginalIP == "" || r.TranslatedIP == "" {
		return fmt.Errorf("original_ip and translated_ip are required")
	}
	return nil
}

func (r *VmwareUpsertVpnTunnelRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if r.SharedKey == "" {
		return fmt.Errorf("shared_key is required")
	}
	if r.PeerNetwork == "" || r.PeerEndpoint == "" || r.PeerIdentificator == "" {
		return fmt.Errorf("peer_network, peer_endpoint and peer_identificator are required")
	}
	return nil
}

func (r *VmwareEdgeBandwidthRequest) Validate() error {
	if r.BandwidthMbps <= 0 {
		return fmt.Errorf("bandwidth_mbps must be greater than 0")
	}
	return nil
}
