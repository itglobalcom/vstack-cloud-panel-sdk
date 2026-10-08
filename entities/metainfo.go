package entities

// Project represents project information
type Project struct {
	ID       int     `json:"id"`
	Balance  float64 `json:"balance"`
	Currency string  `json:"currency"`
	State    string  `json:"state"`
	Created  string  `json:"created"`
}

// Location represents a data center location with configuration limits
type Location struct {
	ID                     string `json:"id"`
	SystemVolumeMin        int    `json:"system_volume_min"`
	AdditionalVolumeMin    int    `json:"additional_volume_min"`
	VolumeMax              int    `json:"volume_max"`
	WindowsSystemVolumeMin int    `json:"windows_system_volume_min"`
	BandwidthMin           int    `json:"bandwidth_min"`
	BandwidthMax           int    `json:"bandwidth_max"`
	CPUQuantityOptions     []int  `json:"cpu_quantity_options"`
	RAMSizeOptions         []int  `json:"ram_size_options"`
}

// Image represents an OS image
type Image struct {
	ID           string `json:"id"`
	LocationID   string `json:"location_id"`
	Type         string `json:"type"`
	OSVersion    string `json:"os_version"`
	Architecture string `json:"architecture"`
	AllowSSHKeys bool   `json:"allow_ssh_keys"`
}

// Application represents an installable application
type Application struct {
	ID         string   `json:"id"`
	LocationID string   `json:"location_id"`
	Images     []string `json:"images"`
	// Category is reported in English only; the catalog does not localize it.
	Category         string                     `json:"category,omitempty"`
	DocumentationURL string                     `json:"documentation_url,omitempty"`
	CredentialsMode  ApplicationCredentialsMode `json:"credentials_mode,omitempty"`
	// LLMKeyEnabled reports that the application may be ordered with a platform
	// language model key.
	LLMKeyEnabled        bool                   `json:"llm_key_enabled"`
	RecommendedCPU       int                    `json:"recommended_cpu,omitempty"`
	RecommendedRamMB     int                    `json:"recommended_ram_mb,omitempty"`
	RecommendedStorageMB int                    `json:"recommended_storage_mb,omitempty"`
	Parameters           []ApplicationParameter `json:"parameters"`
}

// ApplicationParameter is a value the application asks for when a server is ordered.
type ApplicationParameter struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	Default  string `json:"default,omitempty"`
	// Secret marks a value that must not be logged or echoed back.
	Secret  bool               `json:"secret"`
	LLMKind ApplicationLLMKind `json:"llm_kind,omitempty"`
}

// ApplicationCredentialsMode is the class of sign-in the application offers
type ApplicationCredentialsMode string

const (
	ApplicationCredentialsModeServicePassword      ApplicationCredentialsMode = "ServicePassword"
	ApplicationCredentialsModeNoPasswordInTemplate ApplicationCredentialsMode = "NoPasswordInTemplate"
)

// ApplicationLLMKind is the role of an application parameter in language model access
type ApplicationLLMKind string

const (
	// ApplicationLLMKindKey — the parameter carries the language model key.
	ApplicationLLMKindKey ApplicationLLMKind = "Key"
	// ApplicationLLMKindEndpoint — the parameter carries the inference address.
	ApplicationLLMKindEndpoint ApplicationLLMKind = "Endpoint"
)
