package entities

// VMware service entities, for the endpoints under /api/v1/vmware/.
//
// The types are split by domain across vmware_metainfo.go, vmware_server.go,
// vmware_network.go, vmware_edge.go and vmware_task.go, mirroring the root SDK
// layout, and each Validate() sits next to the type it validates.
//
// Two conventions differ from the rest of the package because the API dictates
// them: identifiers are ints rather than strings, and states and enums are
// snake_case strings.

// VmwareLocation represents a VMware location available for provisioning.
type VmwareLocation struct {
	ID           int    `json:"id"`
	TechTitle    string `json:"tech_title"`
	GPUSupported bool   `json:"gpu_supported"`
	// NestedHypervisorSupported reports whether a server in this location can run
	// nested virtualization. It is derived from the VDCs available to the caller
	// rather than stored on the location, so it can differ between projects.
	NestedHypervisorSupported bool `json:"nested_hypervisor_supported"`
	// DiskTypes are the disk types offered in this location; there is no standalone
	// disk-type catalog. Limits are in MB, matching the create/verify requests
	// (system_disk_size_mb, size_mb).
	DiskTypes []*VmwareLocationDiskType `json:"disk_types,omitempty"`
}

// VmwareLocationDiskType is a disk type offered inside a specific location. The
// write-side selection key is Title (system_disk_type / disk_type).
type VmwareLocationDiskType struct {
	Title                  string `json:"title"`
	IsDefault              bool   `json:"is_default"`
	IsSSD                  bool   `json:"is_ssd"`
	IsAllowedForSystemDisk bool   `json:"is_allowed_for_system_disk"`
	MinMB                  int    `json:"min_mb"`
	MaxMB                  int    `json:"max_mb"`
	StepMB                 int    `json:"step_mb"`
	DefaultSizeMB          int    `json:"default_size_mb"`
}

// GPU filter values for GetVmwareImageList. These are the only two
// accepted values; any other one is rejected with HTTP 400.
const (
	VmwareImageGPURequired    = "required"    // only images that require a GPU
	VmwareImageGPUUnsupported = "unsupported" // only images that cannot use a GPU
)

// VmwareImage represents an OS image offered in the VMware catalog.
//
// IsGPUOnly images can only be ordered with a GPU attached, and
// SupportedGPUModelIDs then lists the VmwareGPUModel.ID values they accept.
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
	NICHotRemove         bool   `json:"nic_hot_remove"`
	IsGPUOnly            bool   `json:"is_gpu_only"`
	SupportedGPUModelIDs []int  `json:"supported_gpu_model_ids"`
}

// VmwareGPUModel represents one GPU slicing profile.
//
// ID is NOT a unique key. /vmware/gpu-models returns the same ID up to
// six times - the unit of the listing is a slicing profile, not a model, so the
// real key is the triple (id, capacity_vram_mb, gpu_card_count). Keying by ID
// alone yields collisions. Ordering (VmwareGPURequest)
// is unaffected: it carries the full triple.
type VmwareGPUModel struct {
	ID                    int    `json:"id"`
	TechTitle             string `json:"tech_title"`
	Name                  string `json:"name"`
	CapacityVramMB        int    `json:"capacity_vram_mb"`
	GPUCardCount          int    `json:"gpu_card_count"`
	ServerAllocationLimit int    `json:"server_allocation_limit"`
	MaxServerRamMB        *int   `json:"max_server_ram_mb,omitempty"`
	IsAvailable           *bool  `json:"is_available,omitempty"`
}
