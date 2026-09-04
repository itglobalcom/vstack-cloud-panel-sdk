package entities

// Superseded VMware catalog (lookup) entities.
//
// These types back the GetVMware* catalog methods, which the GetVmware*List
// methods of vmware_metainfo.go replaced. They describe the same three endpoints
// (api/v1/vmware/locations, /images, /gpu-models) in an older, lossier shape and
// are kept only so that published code keeps compiling — new code should use the
// VmwareLocation, VmwareImage and VmwareGPUModel types instead.
//
// They are distinct from the vStack lookups in metainfo.go (Location, Image),
// which describe a different platform and use string identifiers.

// VMwareLocation represents a VMware data center location available to the project.
//
// Deprecated: use VmwareLocation, which this now names. GetVMwareLocations is
// superseded by GetVmwareLocationList.
type VMwareLocation = VmwareLocation

// VMwareDiskType represents a disk type offered in a location.
//
// Deprecated: use VmwareLocationDiskType, which this now names. There is no
// standalone disk-type catalog: disk types travel inside VmwareLocation.DiskTypes,
// their sizes are in megabytes, they carry no id, and the write-side selection key
// is Title. The previous definition of this type described a gigabyte-based
// catalog with an id that the API never published, so its fields never decoded.
type VMwareDiskType = VmwareLocationDiskType

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
