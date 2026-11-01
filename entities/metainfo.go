package entities

import (
	"fmt"
	"time"
)

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

// ConsumptionService is the billed service a consumption amount belongs to.
// It is a different vocabulary from the services of GET /project/services.
// The set grows by addition: a value without a constant here is still valid.
type ConsumptionService string

const (
	ConsumptionServiceVstack          ConsumptionService = "vstack"
	ConsumptionServiceVmware          ConsumptionService = "vmware"
	ConsumptionServiceKubernetes      ConsumptionService = "kubernetes"
	ConsumptionServiceVPN             ConsumptionService = "vpn"
	ConsumptionServiceCloudDirector   ConsumptionService = "cloud_director"
	ConsumptionServiceSSLCertificates ConsumptionService = "ssl_certificates"
	ConsumptionServiceUnocV1          ConsumptionService = "unoc_v1"
	ConsumptionServiceUnocV2          ConsumptionService = "unoc_v2"
	ConsumptionServiceMonitoring      ConsumptionService = "monitoring"
	ConsumptionServiceObjectStorage   ConsumptionService = "object_storage"
	ConsumptionServiceCDN             ConsumptionService = "cdn"
	ConsumptionServiceLLM             ConsumptionService = "llm"
	ConsumptionServiceManagedServers  ConsumptionService = "managed_servers"
	ConsumptionServiceVDI             ConsumptionService = "vdi"
	ConsumptionServiceStreaming       ConsumptionService = "streaming"
)

// ConsumptionResourceType is the kind of a billed root resource.
// The set grows by addition: a value without a constant here is still valid.
type ConsumptionResourceType string

const (
	ConsumptionResourceTypeVstackServer                 ConsumptionResourceType = "vstack_server"
	ConsumptionResourceTypeVstackEdge                   ConsumptionResourceType = "vstack_edge"
	ConsumptionResourceTypeVmwareServer                 ConsumptionResourceType = "vmware_server"
	ConsumptionResourceTypeVmwareServerTemplate         ConsumptionResourceType = "vmware_server_template"
	ConsumptionResourceTypeVmwareNetwork                ConsumptionResourceType = "vmware_network"
	ConsumptionResourceTypeKubernetesCluster            ConsumptionResourceType = "kubernetes_cluster"
	ConsumptionResourceTypeVPNServer                    ConsumptionResourceType = "vpn_server"
	ConsumptionResourceTypeCloudDirectorVDC             ConsumptionResourceType = "cloud_director_vdc"
	ConsumptionResourceTypeSSLCertificate               ConsumptionResourceType = "ssl_certificate"
	ConsumptionResourceTypeUnocV1Company                ConsumptionResourceType = "unoc_v1_company"
	ConsumptionResourceTypeUnocV2RDPUser                ConsumptionResourceType = "unoc_v2_rdp_user"
	ConsumptionResourceTypeUnocV2WebUser                ConsumptionResourceType = "unoc_v2_web_user"
	ConsumptionResourceTypeUnocV2TerminalServer         ConsumptionResourceType = "unoc_v2_terminal_server"
	ConsumptionResourceTypeUnocV2AdditionalInfobases    ConsumptionResourceType = "unoc_v2_additional_infobases"
	ConsumptionResourceTypeUnocV2AdditionalInfobaseSize ConsumptionResourceType = "unoc_v2_additional_infobase_size"
	ConsumptionResourceTypeUnocV2InfobaseBackup         ConsumptionResourceType = "unoc_v2_infobase_backup"
	ConsumptionResourceTypeMonitoringHost               ConsumptionResourceType = "monitoring_host"
	ConsumptionResourceTypeObjectStorage                ConsumptionResourceType = "object_storage"
	ConsumptionResourceTypeCDNSite                      ConsumptionResourceType = "cdn_site"
	ConsumptionResourceTypeCDNS3Resource                ConsumptionResourceType = "cdn_s3_resource"
	ConsumptionResourceTypeCDNVOD                       ConsumptionResourceType = "cdn_vod"
	ConsumptionResourceTypeStream                       ConsumptionResourceType = "stream"
	ConsumptionResourceTypeLLMAccessKey                 ConsumptionResourceType = "llm_access_key"
	ConsumptionResourceTypeManagedServersSubscription   ConsumptionResourceType = "managed_servers_subscription"
	ConsumptionResourceTypeManualBilling                ConsumptionResourceType = "manual_billing"
)

// ConsumptionDateLayout is the layout of the consumption period dates (YYYY-MM-DD).
const ConsumptionDateLayout = "2006-01-02"

// ConsumptionPeriodRequest selects the period of a project consumption read.
// Both dates are in the partner's time zone and inclusive; an empty From means
// the first day of the current month, an empty To means today.
type ConsumptionPeriodRequest struct {
	From string
	To   string
}

// Validate checks the date format and that To is not earlier than From.
func (r *ConsumptionPeriodRequest) Validate() error {
	var from, to time.Time
	var err error
	if r.From != "" {
		if from, err = time.Parse(ConsumptionDateLayout, r.From); err != nil {
			return fmt.Errorf("from %q is not a YYYY-MM-DD date", r.From)
		}
	}
	if r.To != "" {
		if to, err = time.Parse(ConsumptionDateLayout, r.To); err != nil {
			return fmt.Errorf("to %q is not a YYYY-MM-DD date", r.To)
		}
	}
	if r.From != "" && r.To != "" && to.Before(from) {
		return fmt.Errorf("to %s is earlier than from %s", r.To, r.From)
	}
	return nil
}

// ConsumptionTotals is the envelope shared by every project consumption read.
type ConsumptionTotals struct {
	From string `json:"from"`
	To   string `json:"to"`
	// TimeZone is the IANA zone of the partner the dates are counted in.
	TimeZone string `json:"time_zone"`
	Currency string `json:"currency"`
	// IsFinal is false while To is not earlier than today: the last day may still grow.
	IsFinal bool    `json:"is_final"`
	Total   float64 `json:"total"`
}

// ProjectConsumption is the consumption of the project broken down by service.
type ProjectConsumption struct {
	ConsumptionTotals
	Services []ServiceConsumption `json:"services"`
}

// ProjectConsumptionResources is the consumption of the project broken down by root resource.
type ProjectConsumptionResources struct {
	ConsumptionTotals
	Resources []ResourceConsumption `json:"resources"`
}

// ProjectConsumptionDaily is the consumption of the project broken down by day.
type ProjectConsumptionDaily struct {
	ConsumptionTotals
	Days []DailyConsumption `json:"days"`
}

// ServiceConsumption is the amount of one service over the period.
type ServiceConsumption struct {
	Service ConsumptionService `json:"service"`
	Total   float64            `json:"total"`
}

// ResourceConsumption is the amount of one root resource over the period.
type ResourceConsumption struct {
	Service ConsumptionService      `json:"service"`
	Type    ConsumptionResourceType `json:"type"`
	// ID is the Public API identifier of the resource; nil when the resource is
	// not addressable through the Public API.
	ID *string `json:"id,omitempty"`
	// Name is nil when the resource has no name of its own.
	Name  *string `json:"name,omitempty"`
	Total float64 `json:"total"`
}

// DailyConsumption is the amount of one day; a day without charges carries 0.
type DailyConsumption struct {
	Date  string  `json:"date"`
	Total float64 `json:"total"`
}
