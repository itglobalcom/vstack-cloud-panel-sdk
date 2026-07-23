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
}
