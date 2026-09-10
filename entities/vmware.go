package entities

// Superseded VMware catalog (lookup) entities.
//
// These types back the GetVMware* catalog methods, which the GetVmware*List
// methods of vmware_metainfo.go supersede. They describe the same endpoints in an
// older, lossier shape and stay published so that existing code keeps compiling —
// new code should use the VmwareLocation, VmwareImage and VmwareGPUModel types
// instead.
//
// They are distinct from the vStack lookups in metainfo.go (Location, Image),
// which describe a different platform and use string identifiers.

// VMwareLocation represents a VMware data center location available to the project.
//
// Deprecated: use VmwareLocation, which this now names. GetVMwareLocations is
// superseded by GetVmwareLocationList.
type VMwareLocation = VmwareLocation

// VMwareDiskType represents a disk type allowed by the partner/location tariff.
//
// Deprecated: use VmwareLocationDiskType, reached through VmwareLocation.DiskTypes.
// The fields below do not match the wire: the Public API publishes disk types
// inside a location, without an id and with the limits in megabytes, so every
// field of this type decodes to zero. Only GetVMwareDiskTypes, itself deprecated,
// fills it.
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

// VMwareStorageProfile represents an active storage profile for a
// location/disk type combination.
//
// Deprecated: the Public API publishes no storage-profile resource — profiles are
// an internal join behind a location's disk types. Only GetVMwareStorageProfiles,
// itself deprecated, fills it.
type VMwareStorageProfile struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	DiskTypeID int    `json:"disk_type_id"`
	IsDefault  bool   `json:"is_default"`
	// FreeSpaceGB is int64: profile capacity is reported in gigabytes and can
	// exceed the range of a 32-bit integer.
	FreeSpaceGB int64 `json:"free_space_gb"`
}

// VMwareGPUModel represents a GPU model available to the partner together with
// its allocation limits.
//
// Deprecated: use VmwareGPUModel, which tells an absent MaxServerRamMB /
// IsAvailable from a zero one. GetVMwareGPUModels is superseded by
// GetVmwareGPUModelList.
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

// VMwareImage represents an OS image (template) available to the project.
//
// Deprecated: use VmwareImage. GetVMwareImages is superseded by
// GetVmwareImageList, whose gpu filter has three states rather than two.
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
