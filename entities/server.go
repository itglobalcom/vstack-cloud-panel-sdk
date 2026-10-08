package entities

import "fmt"

// Server represents a server instance
type Server struct {
	ID             string   `json:"id"`
	LocationID     string   `json:"location_id"`
	CPU            int      `json:"cpu"`
	RamMB          int      `json:"ram_mb"`
	Volumes        []Volume `json:"volumes"`
	NICs           []NIC    `json:"nics"`
	ImageID        string   `json:"image_id"`
	IsPowerOn      bool     `json:"is_power_on"`
	Name           string   `json:"name"`
	Login          string   `json:"login"`
	Password       string   `json:"password"`
	SSHKeyIDs      []int    `json:"ssh_key_ids"`
	State          string   `json:"state"`
	Created        string   `json:"created"`
	Tags           []string `json:"tags"`
	ApplicationIDs []string `json:"application_ids"`
	// ApplicationInstallations is nil when the response carries no such field and
	// non-nil empty when it carries an empty list.
	ApplicationInstallations []ApplicationInstallation `json:"application_installations"`
	AffinityGroupID          string                    `json:"affinity_group_id"`
}

// CreateServerRequest represents a request to create a server
type CreateServerRequest struct {
	LocationID     string        `json:"location_id"`
	ImageID        string        `json:"image_id"`
	CPU            int           `json:"cpu"`
	RamMB          int           `json:"ram_mb"`
	Volumes        []VolumeSpec  `json:"volumes"`
	Networks       []NetworkSpec `json:"networks"`
	Name           string        `json:"name"`
	SSHKeyIDs      []int         `json:"ssh_key_ids,omitempty"`
	ApplicationIDs []string      `json:"application_ids,omitempty"`
	// Applications supersedes ApplicationIDs: while it is not empty, application_ids
	// is not applied.
	Applications     []ApplicationSpec `json:"applications,omitempty"`
	Tags             []string          `json:"tags,omitempty"`
	AffinityGroupID  string            `json:"affinity_group_id,omitempty"`
	ServerInitScript string            `json:"server_init_script,omitempty"`
	// Backup enables the backup service with this schedule once the server is
	// created; nil creates the server without backup.
	Backup *BackupSchedule `json:"backup,omitempty"`
}

// ApplicationSpec specifies an application to install on a server being created
type ApplicationSpec struct {
	ID string `json:"id"`
	// Parameters are values by parameter name; the catalog entry names them and
	// tells which of them are required.
	Parameters map[string]string `json:"parameters,omitempty"`
	// IssueLLMKey asks for a platform language model key for the application; it is
	// accepted only for a catalog entry with LLMKeyEnabled.
	IssueLLMKey bool `json:"issue_llm_key"`
}

// VolumeSpec specifies volume configuration for server creation
type VolumeSpec struct {
	Name   string `json:"name"`
	SizeMB int    `json:"size_mb"`
}

// NetworkSpec specifies network configuration for server creation
type NetworkSpec struct {
	BandwidthMbps int    `json:"bandwidth_mbps,omitempty"`
	NetworkID     string `json:"network_id,omitempty"`
	IPAddress     string `json:"ip_address,omitempty"`
}

// UpdateServerRequest represents a PUT request to update server resources
type UpdateServerRequest struct {
	CPU   int `json:"cpu"`
	RamMB int `json:"ram_mb"`
}

// PatchServerRequest represents a PATCH request to update server resources
type PatchServerRequest struct {
	CPU   *int `json:"cpu,omitempty"`
	RamMB *int `json:"ram_mb,omitempty"`
}

// RenameServerRequest represents a request to rename a server
type RenameServerRequest struct {
	Name string `json:"name"`
}

// CreateServerTagRequest represents a single tag in a server create request.
type CreateServerTagRequest struct {
	Tag string `json:"value"`
}

// Server state constants
const (
	ServerStateNew     = "New"
	ServerStateActive  = "Active"
	ServerStateBusy    = "Busy"
	ServerStateBlocked = "Blocked"
)

// ApplicationInstallation is the outcome of installing one application on a server
type ApplicationInstallation struct {
	ID            string                   `json:"id"`
	State         ApplicationInstallState  `json:"state"`
	OutcomeReason ApplicationOutcomeReason `json:"outcome_reason,omitempty"`
	// InstalledAt is the time of the terminal outcome, a failed installation included.
	InstalledAt string                 `json:"installed_at,omitempty"`
	Addresses   []ApplicationAddress   `json:"addresses"`
	Components  []ApplicationComponent `json:"components"`
	AppLogin    string                 `json:"app_login,omitempty"`
	AppPassword string                 `json:"app_password,omitempty"`
	LLMKeyID    string                 `json:"llm_key_id,omitempty"`
}

// ApplicationAddress is the address of one service of an installed application
type ApplicationAddress struct {
	Service string `json:"service"`
	Address string `json:"address"`
}

// ApplicationComponent is one service of the application as of the installation
type ApplicationComponent struct {
	Name string                   `json:"name"`
	Kind ApplicationComponentKind `json:"kind"`
	// ObservedStatus is a status string of the platform, not a value of a closed set.
	ObservedStatus string `json:"observed_status,omitempty"`
	Address        string `json:"address,omitempty"`
}

// ApplicationInstallState is the state of an application installation
type ApplicationInstallState string

const (
	ApplicationInstallStateInstalling ApplicationInstallState = "Installing"
	ApplicationInstallStateInstalled  ApplicationInstallState = "Installed"
	ApplicationInstallStateFailed     ApplicationInstallState = "Failed"
)

// ApplicationOutcomeReason is the reason an installation did not succeed
type ApplicationOutcomeReason string

const (
	// ApplicationOutcomeReasonRegistrationFailed — the server was not registered on
	// the installation platform.
	ApplicationOutcomeReasonRegistrationFailed ApplicationOutcomeReason = "RegistrationFailed"
	// ApplicationOutcomeReasonDockerNotReady — the container engine of the server was
	// not ready.
	ApplicationOutcomeReasonDockerNotReady ApplicationOutcomeReason = "DockerNotReady"
	// ApplicationOutcomeReasonServiceCreateFailed — the application service was not
	// created.
	ApplicationOutcomeReasonServiceCreateFailed ApplicationOutcomeReason = "ServiceCreateFailed"
	// ApplicationOutcomeReasonDeployTimeout — the installation did not finish in the
	// allotted time.
	ApplicationOutcomeReasonDeployTimeout ApplicationOutcomeReason = "DeployTimeout"
	// ApplicationOutcomeReasonHostUnreachable — the server was unreachable.
	ApplicationOutcomeReasonHostUnreachable ApplicationOutcomeReason = "HostUnreachable"
	// ApplicationOutcomeReasonLLMKeyValueMissing — the language model key was not issued.
	ApplicationOutcomeReasonLLMKeyValueMissing ApplicationOutcomeReason = "LlmKeyValueMissing"
)

// ApplicationComponentKind is the kind of an application service
type ApplicationComponentKind string

const (
	ApplicationComponentKindWeb      ApplicationComponentKind = "Web"
	ApplicationComponentKindWorker   ApplicationComponentKind = "Worker"
	ApplicationComponentKindDatabase ApplicationComponentKind = "Database"
)

// Validate checks if the create server request is valid
func (r *CreateServerRequest) Validate() error {
	if r.LocationID == "" {
		return fmt.Errorf("location_id is required")
	}
	if r.ImageID == "" {
		return fmt.Errorf("image_id is required")
	}
	if r.CPU <= 0 {
		return fmt.Errorf("cpu must be greater than 0")
	}
	if r.RamMB <= 0 {
		return fmt.Errorf("ram_mb must be greater than 0")
	}
	if len(r.Volumes) == 0 {
		return fmt.Errorf("at least one volume is required")
	}
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}

	// Validate volumes
	hasBootVolume := false
	for _, vol := range r.Volumes {
		if vol.Name == "boot" {
			hasBootVolume = true
		}
		if vol.SizeMB <= 0 {
			return fmt.Errorf("volume size must be greater than 0")
		}
	}
	if !hasBootVolume {
		return fmt.Errorf("boot volume is required")
	}

	if r.Backup != nil {
		if err := r.Backup.Validate(); err != nil {
			return fmt.Errorf("invalid backup: %w", err)
		}
	}

	for i, app := range r.Applications {
		if app.ID == "" {
			return fmt.Errorf("applications[%d]: id is required", i)
		}
		if _, ok := app.Parameters[""]; ok {
			return fmt.Errorf("applications[%d]: parameter name is required", i)
		}
	}

	return nil
}

// GetServerPriceRequest represents a request to get server price
type GetServerPriceRequest struct {
	LocationID string        `json:"location_id"`
	ImageID    string        `json:"image_id"`
	CPU        int           `json:"cpu"`
	RamMB      int           `json:"ram_mb"`
	Volumes    []VolumeSpec  `json:"volumes"`
	Networks   []NetworkSpec `json:"networks,omitempty"`
	Name       string        `json:"name,omitempty"`
	SSHKeyIDs  []int         `json:"ssh_key_ids,omitempty"`
}

// GetServerPriceResponse represents the price response
type GetServerPriceResponse struct {
	Price float64 `json:"price"`
}

// Validate checks if the get price request is valid
func (r *GetServerPriceRequest) Validate() error {
	if r.LocationID == "" {
		return fmt.Errorf("location_id is required")
	}
	if r.ImageID == "" {
		return fmt.Errorf("image_id is required")
	}
	if r.CPU <= 0 {
		return fmt.Errorf("cpu must be greater than 0")
	}
	if r.RamMB <= 0 {
		return fmt.Errorf("ram_mb must be greater than 0")
	}
	if len(r.Volumes) == 0 {
		return fmt.Errorf("at least one volume is required")
	}

	// Validate volumes
	for i, vol := range r.Volumes {
		if vol.SizeMB <= 0 {
			return fmt.Errorf("volume[%d]: size must be greater than 0", i)
		}
	}

	return nil
}

// Validate checks if the update server request is valid
func (r *UpdateServerRequest) Validate() error {
	if r.CPU <= 0 {
		return fmt.Errorf("cpu must be greater than 0")
	}
	if r.RamMB <= 0 {
		return fmt.Errorf("ram_mb must be greater than 0")
	}
	return nil
}

// Validate checks if the patch server request is valid
func (r *PatchServerRequest) Validate() error {
	if r.CPU == nil && r.RamMB == nil {
		return fmt.Errorf("at least one field (cpu or ram_mb) must be provided")
	}
	if r.CPU != nil && *r.CPU <= 0 {
		return fmt.Errorf("cpu must be greater than 0")
	}
	if r.RamMB != nil && *r.RamMB <= 0 {
		return fmt.Errorf("ram_mb must be greater than 0")
	}
	return nil
}

// Validate checks if the rename server request is valid
func (r *RenameServerRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

// Validate checks if the create server tag request is valid
func (r *CreateServerTagRequest) Validate() error {
	if r.Tag == "" {
		return fmt.Errorf("value is required")
	}
	return nil
}

// IsActive checks if the server is in active state
func (s *Server) IsActive() bool {
	return s.State == ServerStateActive
}

// IsPoweredOn checks if the server is powered on
func (s *Server) IsPoweredOn() bool {
	return s.IsPowerOn
}
