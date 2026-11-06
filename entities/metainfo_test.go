package entities

import "testing"

// The catalog sends the values of a closed set as PascalCase strings, and the
// spellings are fixed by the contract, not by the SDK.
func TestApplicationCatalogClosedSetsOnTheWire(t *testing.T) {
	fixed := map[string]struct {
		got  string
		want string
	}{
		"ApplicationCredentialsModeServicePassword":      {string(ApplicationCredentialsModeServicePassword), "ServicePassword"},
		"ApplicationCredentialsModeNoPasswordInTemplate": {string(ApplicationCredentialsModeNoPasswordInTemplate), "NoPasswordInTemplate"},
		"ApplicationLLMKindKey":                          {string(ApplicationLLMKindKey), "Key"},
		"ApplicationLLMKindEndpoint":                     {string(ApplicationLLMKindEndpoint), "Endpoint"},
	}
	for name, c := range fixed {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", name, c.got, c.want)
		}
	}
}

// Both dates are optional, so only a malformed date or a reversed period is
// refused; one day (From == To) is a valid period.
func TestConsumptionPeriodRequestValidate(t *testing.T) {
	valid := map[string]ConsumptionPeriodRequest{
		"empty":      {},
		"month":      {From: "2026-10-01", To: "2026-10-31"},
		"from only":  {From: "2026-10-01"},
		"to only":    {To: "2026-10-31"},
		"single day": {From: "2026-10-15", To: "2026-10-15"},
	}
	for name, r := range valid {
		if err := r.Validate(); err != nil {
			t.Errorf("%s: Validate(%+v) = %v, want nil", name, r, err)
		}
	}

	invalid := map[string]ConsumptionPeriodRequest{
		"reversed":          {From: "2026-10-31", To: "2026-10-01"},
		"bad month in from": {From: "2026-13-01"},
		"dotted to":         {To: "01.10.2026"},
	}
	for name, r := range invalid {
		if err := r.Validate(); err == nil {
			t.Errorf("%s: Validate(%+v) = nil, want an error", name, r)
		}
	}
}

// The spellings of service and type are the values of the contract on the wire,
// the same strings the service filter is sent with.
func TestConsumptionClosedSetsOnTheWire(t *testing.T) {
	services := map[ConsumptionService]string{
		ConsumptionServiceVstack:          "vstack",
		ConsumptionServiceVmware:          "vmware",
		ConsumptionServiceKubernetes:      "kubernetes",
		ConsumptionServiceVPN:             "vpn",
		ConsumptionServiceCloudDirector:   "cloud_director",
		ConsumptionServiceSSLCertificates: "ssl_certificates",
		ConsumptionServiceUnocV1:          "unoc_v1",
		ConsumptionServiceUnocV2:          "unoc_v2",
		ConsumptionServiceMonitoring:      "monitoring",
		ConsumptionServiceObjectStorage:   "object_storage",
		ConsumptionServiceCDN:             "cdn",
		ConsumptionServiceLLM:             "llm",
		ConsumptionServiceManagedServers:  "managed_servers",
		ConsumptionServiceVDI:             "vdi",
		ConsumptionServiceStreaming:       "streaming",
	}
	if len(services) != 15 {
		t.Errorf("got %d distinct services, want 15", len(services))
	}
	for got, want := range services {
		if string(got) != want {
			t.Errorf("service %q, want %q", got, want)
		}
	}

	types := map[ConsumptionResourceType]string{
		ConsumptionResourceTypeVstackServer:                 "vstack_server",
		ConsumptionResourceTypeVstackEdge:                   "vstack_edge",
		ConsumptionResourceTypeVmwareServer:                 "vmware_server",
		ConsumptionResourceTypeVmwareServerTemplate:         "vmware_server_template",
		ConsumptionResourceTypeVmwareNetwork:                "vmware_network",
		ConsumptionResourceTypeKubernetesCluster:            "kubernetes_cluster",
		ConsumptionResourceTypeVPNServer:                    "vpn_server",
		ConsumptionResourceTypeCloudDirectorVDC:             "cloud_director_vdc",
		ConsumptionResourceTypeSSLCertificate:               "ssl_certificate",
		ConsumptionResourceTypeUnocV1Company:                "unoc_v1_company",
		ConsumptionResourceTypeUnocV2RDPUser:                "unoc_v2_rdp_user",
		ConsumptionResourceTypeUnocV2WebUser:                "unoc_v2_web_user",
		ConsumptionResourceTypeUnocV2TerminalServer:         "unoc_v2_terminal_server",
		ConsumptionResourceTypeUnocV2AdditionalInfobases:    "unoc_v2_additional_infobases",
		ConsumptionResourceTypeUnocV2AdditionalInfobaseSize: "unoc_v2_additional_infobase_size",
		ConsumptionResourceTypeUnocV2InfobaseBackup:         "unoc_v2_infobase_backup",
		ConsumptionResourceTypeMonitoringHost:               "monitoring_host",
		ConsumptionResourceTypeObjectStorage:                "object_storage",
		ConsumptionResourceTypeCDNSite:                      "cdn_site",
		ConsumptionResourceTypeCDNS3Resource:                "cdn_s3_resource",
		ConsumptionResourceTypeCDNVOD:                       "cdn_vod",
		ConsumptionResourceTypeStream:                       "stream",
		ConsumptionResourceTypeLLMAccessKey:                 "llm_access_key",
		ConsumptionResourceTypeLLMCredits:                   "llm_credits",
		ConsumptionResourceTypeLLMPackage:                   "llm_package",
		ConsumptionResourceTypeManagedServersSubscription:   "managed_servers_subscription",
		ConsumptionResourceTypeManualBilling:                "manual_billing",
	}
	if len(types) != 27 {
		t.Errorf("got %d distinct resource types, want 27", len(types))
	}
	for got, want := range types {
		if string(got) != want {
			t.Errorf("resource type %q, want %q", got, want)
		}
	}
}
