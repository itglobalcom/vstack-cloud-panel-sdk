package entities

// VMware service entities (Public API, prefixed with /api/v1/vmware/**).
//
// C-8: the former single entities/vmware.go was intentionally split by domain,
// mirroring the root SDK layout: vmware_metainfo.go, vmware_server.go,
// vmware_network.go, vmware_edge.go, vmware_task.go. Each Validate() lives next
// to the type it validates.
//
// Identifiers are flat ints; states and enums are snake_case strings.

// VmwareLocation represents a VMware location available for provisioning.
type VmwareLocation struct {
	ID           int    `json:"id"`
	TechTitle    string `json:"tech_title"`
	GPUSupported bool   `json:"gpu_supported"` // C-12: Gpu -> GPU
}

// VmwareImage represents an OS image offered in the VMware catalog.
//
// RISK-1: this mapping has not been verified against a live response - the image
// catalog was empty during review (metainfo-api.md, API-1). It matches the
// backend DTO and schema statically; because encoding/json ignores unknown keys,
// a mismatched field name would yield zero values rather than a decode error.
// Re-verify against a live response once API-1 is fixed.
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
	NICHotRemove         bool   `json:"nic_hot_remove"`          // C-12: Nic -> NIC
	IsGPUOnly            bool   `json:"is_gpu_only"`             // C-12: Gpu -> GPU
	SupportedGPUModelIDs []int  `json:"supported_gpu_model_ids"` // C-12: Gpu -> GPU
}

// VmwareDiskType represents a disk type available for volumes.
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

// VmwareStorageProfile represents a storage profile bound to a disk type.
type VmwareStorageProfile struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	DiskTypeID  int    `json:"disk_type_id"`
	IsDefault   bool   `json:"is_default"`
	FreeSpaceGB int    `json:"free_space_gb"`
}

// VmwareGPUModel represents one GPU slicing profile.
//
// C-12: Gpu -> GPU.
//
// SDK-7: ID is NOT a unique key. /vmware/gpu-models returns the same ID up to
// six times - the unit of the listing is a slicing profile, not a model, so the
// real key is the triple (id, capacity_vram_mb, gpu_card_count). Keying by ID
// alone yields collisions (metainfo-sdk.md, SDK-7). Ordering (VmwareGPURequest)
// is unaffected: it carries the full triple.
type VmwareGPUModel struct {
	ID                    int    `json:"id"`
	TechTitle             string `json:"tech_title"`
	Name                  string `json:"name"`
	CapacityVramMB        int    `json:"capacity_vram_mb"`
	GPUCardCount          int    `json:"gpu_card_count"` // C-12: Gpu -> GPU
	ServerAllocationLimit int    `json:"server_allocation_limit"`
	MaxServerRamMB        *int   `json:"max_server_ram_mb,omitempty"`
	IsAvailable           *bool  `json:"is_available,omitempty"`
}
