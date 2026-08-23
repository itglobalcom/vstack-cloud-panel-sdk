package entities

// VMware catalog (lookup) entities.
//
// These resources are read-only: they describe what the VMware section of the
// project can be built from — locations, disk types, GPU models, storage
// profiles and OS images. They are exposed by the api/v1/vmware/* endpoints
// and are distinct from the vStack lookups in metainfo.go (Location, Image),
// which describe a different platform and use string identifiers.

// VMwareLocation represents a VMware data center location available to the project
type VMwareLocation struct {
	ID        int    `json:"id"`
	TechTitle string `json:"tech_title"`
	// GPUSupported reports whether at least one GPU model available to the
	// partner can be allocated in this location.
	GPUSupported bool `json:"gpu_supported"`
	// NestedHypervisorSupported reports whether the location has a VDC available
	// to the caller that supports nested virtualization. It is derived from the
	// VDCs the caller may provision in, so two projects can see different values
	// for the same location.
	NestedHypervisorSupported bool `json:"nested_hypervisor_supported"`
}

// VMwareDiskType represents a disk type allowed by the partner/location tariff.
//
// Note: write operations select a disk type by Title, not by ID (see the
// create server / create volume requests); ID is informational and links a
// disk type to its storage profiles.
type VMwareDiskType struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	// MinGB, MaxGB, StepGB and StartValueGB describe the allowed volume sizes:
	// a size must lie in [MinGB, MaxGB] and be a multiple of StepGB;
	// StartValueGB is the value the panel offers by default.
	MinGB                  int  `json:"min_gb"`
	MaxGB                  int  `json:"max_gb"`
	StepGB                 int  `json:"step_gb"`
	StartValueGB           int  `json:"start_value_gb"`
	IsAllowedForSystemDisk bool `json:"is_allowed_for_system_disk"`
	IsSSD                  bool `json:"is_ssd"`
}

// VMwareGPUModel represents a GPU model available to the partner together with
// its allocation limits
type VMwareGPUModel struct {
	ID        int    `json:"id"`
	TechTitle string `json:"tech_title"`
	Name      string `json:"name"`
	// CapacityVramMB is the video memory of a single card, in megabytes.
	CapacityVramMB int `json:"capacity_vram_mb"`
	GPUCardCount   int `json:"gpu_card_count"`
	// ServerAllocationLimit is the maximum number of cards of this model that
	// can be attached to one server.
	ServerAllocationLimit int `json:"server_allocation_limit"`
	// MaxServerRamMB is the maximum server RAM allowed with this model, in megabytes.
	MaxServerRamMB int `json:"max_server_ram_mb"`
	// IsAvailable reports whether the partner's GPU policy currently allows
	// allocating this model.
	IsAvailable bool `json:"is_available"`
}

// VMwareStorageProfile represents an active storage profile for a
// location/disk type combination
type VMwareStorageProfile struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	DiskTypeID int    `json:"disk_type_id"`
	IsDefault  bool   `json:"is_default"`
	// FreeSpaceGB is int64: profile capacity is reported in gigabytes and can
	// exceed the range of a 32-bit integer.
	FreeSpaceGB int64 `json:"free_space_gb"`
}

// VMwareImage represents an OS image (template) available to the project
type VMwareImage struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	OSFamily string `json:"os_family"`
	OSType   string `json:"os_type"`
	// MinRamMB is the minimum server RAM the image can be deployed with, in megabytes.
	MinRamMB int `json:"min_ram_mb"`
	// HddGB is the minimum system disk size, in gigabytes.
	HddGB           int  `json:"hdd_gb"`
	SSHKeySupported bool `json:"ssh_key_supported"`
	// CPUHotAdd, MemoryHotAdd and NICHotRemove report whether the resource can
	// be changed without powering the server off.
	CPUHotAdd    bool `json:"cpu_hot_add"`
	MemoryHotAdd bool `json:"memory_hot_add"`
	NICHotRemove bool `json:"nic_hot_remove"`
	// IsGPUOnly marks an image that can only be deployed on a GPU server.
	IsGPUOnly bool `json:"is_gpu_only"`
	// SupportedGPUModelIDs lists the VMwareGPUModel IDs the image can be used
	// with; empty means no GPU restriction.
	SupportedGPUModelIDs []int `json:"supported_gpu_model_ids"`
}
