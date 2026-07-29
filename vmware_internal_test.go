package sdk

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
)

func TestBuildVMwarePath(t *testing.T) {
	if got := buildVMwarePath(vmwareLocationsPath, nil); got != "vmware/locations" {
		t.Errorf("no filters: got %q", got)
	}

	// url.Values.Encode sorts keys, so the path is deterministic.
	filters := url.Values{}
	addIDFilter(filters, "location_id", 2)
	addIDFilter(filters, "disk_type_id", 7)
	want := "vmware/storage-profiles?disk_type_id=7&location_id=2"
	if got := buildVMwarePath(vmwareStorageProfilesPath, filters); got != want {
		t.Errorf("both filters: got %q, want %q", got, want)
	}

	// An empty (but non-nil) set of filters must not add a bare "?".
	if got := buildVMwarePath(vmwareImagesPath, url.Values{}); got != "vmware/images" {
		t.Errorf("empty filters: got %q", got)
	}
}

// A non-positive filter means "not specified": the API treats an absent
// parameter as no filter, while location_id=0 is not a valid location and
// would turn a list call into a 400.
func TestAddIDFilterSkipsNonPositive(t *testing.T) {
	filters := url.Values{}
	addIDFilter(filters, "location_id", 0)
	addIDFilter(filters, "disk_type_id", -1)
	if len(filters) != 0 {
		t.Errorf("non-positive values must be skipped, got %v", filters)
	}

	addIDFilter(filters, "location_id", 14)
	if filters.Get("location_id") != "14" {
		t.Errorf("positive value must be set, got %v", filters)
	}
}

func TestParseVMwareLocationsResponse(t *testing.T) {
	body := []byte(`{"locations":[
		{"id":2,"tech_title":"ds-msk","gpu_supported":true},
		{"id":6,"tech_title":"sdn-spb","gpu_supported":false}
	]}`)

	var resp ListVMwareLocationsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(resp.Locations) != 2 {
		t.Fatalf("got %d locations, want 2", len(resp.Locations))
	}
	first := resp.Locations[0]
	if first.ID != 2 || first.TechTitle != "ds-msk" || !first.GPUSupported {
		t.Errorf("first location: %+v", first)
	}
	if resp.Locations[1].GPUSupported {
		t.Error("second location must not report GPU support")
	}
}

func TestParseVMwareDiskTypesResponse(t *testing.T) {
	body := []byte(`{"disk_types":[{
		"id":3,"title":"SSD","min_gb":10,"max_gb":2048,"step_gb":10,
		"start_value_gb":20,"is_allowed_for_system_disk":true,"is_ssd":true
	}]}`)

	var resp ListVMwareDiskTypesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(resp.DiskTypes) != 1 {
		t.Fatalf("got %d disk types, want 1", len(resp.DiskTypes))
	}
	dt := resp.DiskTypes[0]
	if dt.ID != 3 || dt.Title != "SSD" {
		t.Errorf("id/title: %+v", dt)
	}
	if dt.MinGB != 10 || dt.MaxGB != 2048 || dt.StepGB != 10 || dt.StartValueGB != 20 {
		t.Errorf("size limits: %+v", dt)
	}
	if !dt.IsAllowedForSystemDisk || !dt.IsSSD {
		t.Errorf("flags: %+v", dt)
	}
}

func TestParseVMwareGPUModelsResponse(t *testing.T) {
	body := []byte(`{"gpu_models":[{
		"id":1,"tech_title":"nvidia-a100","name":"NVIDIA A100",
		"capacity_vram_mb":40960,"gpu_card_count":8,"server_allocation_limit":4,
		"max_server_ram_mb":524288,"is_available":true
	}]}`)

	var resp ListVMwareGPUModelsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(resp.GPUModels) != 1 {
		t.Fatalf("got %d GPU models, want 1", len(resp.GPUModels))
	}
	m := resp.GPUModels[0]
	if m.ID != 1 || m.TechTitle != "nvidia-a100" || m.Name != "NVIDIA A100" {
		t.Errorf("identity: %+v", m)
	}
	if m.CapacityVramMB != 40960 || m.GPUCardCount != 8 {
		t.Errorf("capacity: %+v", m)
	}
	if m.ServerAllocationLimit != 4 || m.MaxServerRamMB != 524288 || !m.IsAvailable {
		t.Errorf("allocation limits: %+v", m)
	}
}

func TestParseVMwareStorageProfilesResponse(t *testing.T) {
	// free_space_gb is int64: profile capacity does not fit into int32.
	body := []byte(`{"storage_profiles":[{
		"id":11,"name":"SSD-Fast","disk_type_id":3,"is_default":true,
		"free_space_gb":5000000000
	}]}`)

	var resp ListVMwareStorageProfilesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(resp.StorageProfiles) != 1 {
		t.Fatalf("got %d profiles, want 1", len(resp.StorageProfiles))
	}
	p := resp.StorageProfiles[0]
	if p.ID != 11 || p.Name != "SSD-Fast" || p.DiskTypeID != 3 || !p.IsDefault {
		t.Errorf("profile: %+v", p)
	}
	if p.FreeSpaceGB != 5000000000 {
		t.Errorf("free_space_gb = %d, want 5000000000", p.FreeSpaceGB)
	}
}

func TestParseVMwareImagesResponse(t *testing.T) {
	body := []byte(`{"images":[
		{
			"id":42,"name":"Ubuntu 24.04","os_family":"Linux","os_type":"ubuntu64Guest",
			"min_ram_mb":1024,"hdd_gb":20,"ssh_key_supported":true,
			"cpu_hot_add":true,"memory_hot_add":true,"nic_hot_remove":false,
			"is_gpu_only":false,"supported_gpu_model_ids":[1,2]
		},
		{"id":43,"name":"Windows Server 2022","os_family":"Windows","os_type":"windows9Server64Guest"}
	]}`)

	var resp ListVMwareImagesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(resp.Images) != 2 {
		t.Fatalf("got %d images, want 2", len(resp.Images))
	}
	img := resp.Images[0]
	if img.ID != 42 || img.Name != "Ubuntu 24.04" || img.OSFamily != "Linux" || img.OSType != "ubuntu64Guest" {
		t.Errorf("identity: %+v", img)
	}
	if img.MinRamMB != 1024 || img.HddGB != 20 {
		t.Errorf("requirements: %+v", img)
	}
	if !img.SSHKeySupported || !img.CPUHotAdd || !img.MemoryHotAdd || img.NICHotRemove || img.IsGPUOnly {
		t.Errorf("flags: %+v", img)
	}
	if len(img.SupportedGPUModelIDs) != 2 || img.SupportedGPUModelIDs[0] != 1 || img.SupportedGPUModelIDs[1] != 2 {
		t.Errorf("supported_gpu_model_ids = %v", img.SupportedGPUModelIDs)
	}

	// The API serializes with NullValueHandling.Ignore, so optional fields and
	// collections may be missing entirely — that must decode to zero values,
	// not fail.
	if resp.Images[1].SupportedGPUModelIDs != nil {
		t.Errorf("absent collection must stay nil, got %v", resp.Images[1].SupportedGPUModelIDs)
	}
}

// An empty list arrives either as an empty array or (NullValueHandling.Ignore)
// as an object without the collection at all.
func TestParseVMwareEmptyResponses(t *testing.T) {
	for _, body := range []string{`{"disk_types":[]}`, `{}`} {
		var resp ListVMwareDiskTypesResponse
		if err := json.Unmarshal([]byte(body), &resp); err != nil {
			t.Fatalf("unmarshal %s: %v", body, err)
		}
		if len(resp.DiskTypes) != 0 {
			t.Errorf("%s: got %d disk types, want 0", body, len(resp.DiskTypes))
		}
	}
}

// An unknown location_id is rejected by the API with 400 / -8049 rather than
// returning an empty list, so callers must be able to tell that case apart.
func TestIsInvalidLocation(t *testing.T) {
	invalid := &RequestError{
		Status:     "400 Bad Request",
		StatusCode: http.StatusBadRequest,
		Codes:      []int{APICodeDCLocationDoesNotExist},
	}
	if !IsInvalidLocation(invalid) {
		t.Error("IsInvalidLocation must match -8049")
	}
	if IsInvalidLocation(&RequestError{StatusCode: http.StatusBadRequest, Codes: []int{APICodeConflict}}) {
		t.Error("IsInvalidLocation must not match other codes")
	}
	if IsInvalidLocation(nil) {
		t.Error("IsInvalidLocation(nil) must be false")
	}
}
