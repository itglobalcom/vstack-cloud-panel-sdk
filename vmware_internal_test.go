package sdk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

func TestBuildVMwarePath(t *testing.T) {
	if got := buildVMwarePath(vmwareLocationsPath, nil); got != "vmware/locations" {
		t.Errorf("no filters: got %q", got)
	}

	// url.Values.Encode sorts keys, so the path is deterministic.
	filters := url.Values{}
	if err := addIDFilter(filters, "location_id", 2); err != nil {
		t.Fatalf("location_id filter: %v", err)
	}
	if err := addIDFilter(filters, "disk_type_id", 7); err != nil {
		t.Fatalf("disk_type_id filter: %v", err)
	}
	want := "vmware/storage-profiles?disk_type_id=7&location_id=2"
	if got := buildVMwarePath(vmwareStorageProfilesPath, filters); got != want {
		t.Errorf("both filters: got %q, want %q", got, want)
	}

	// An empty (but non-nil) set of filters must not add a bare "?".
	if got := buildVMwarePath(vmwareImagesPath, url.Values{}); got != "vmware/images" {
		t.Errorf("empty filters: got %q", got)
	}
}

// Zero means "not specified" (the API treats an absent parameter as no filter
// and 0 is not a member of DCLocationEnum), while a negative id is a caller
// error: dropping it would answer a broken id with the full unfiltered list.
func TestAddIDFilter(t *testing.T) {
	filters := url.Values{}
	if err := addIDFilter(filters, "location_id", 0); err != nil {
		t.Errorf("zero must mean \"no filter\", got error: %v", err)
	}
	if len(filters) != 0 {
		t.Errorf("zero must not add a filter, got %v", filters)
	}

	if err := addIDFilter(filters, "disk_type_id", -1); err == nil {
		t.Error("negative value must be rejected")
	}
	if len(filters) != 0 {
		t.Errorf("rejected value must not be added, got %v", filters)
	}

	if err := addIDFilter(filters, "location_id", 14); err != nil {
		t.Fatalf("positive value: %v", err)
	}
	if filters.Get("location_id") != "14" {
		t.Errorf("positive value must be set, got %v", filters)
	}
}

// A negative id must fail before the request is built — the caller must not get
// a full unfiltered list back.
func TestGetVMwareCatalogRejectsNegativeID(t *testing.T) {
	config, err := NewConfig("token", "https://api.example.com")
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	ctx := context.Background()

	if _, err := client.GetVMwareDiskTypes(ctx, -1); err == nil {
		t.Error("GetVMwareDiskTypes must reject a negative location_id")
	}
	if _, err := client.GetVMwareGPUModels(ctx, -1); err == nil {
		t.Error("GetVMwareGPUModels must reject a negative location_id")
	}
	if _, err := client.GetVMwareStorageProfiles(ctx, 2, -1); err == nil {
		t.Error("GetVMwareStorageProfiles must reject a negative disk_type_id")
	}
	if _, err := client.GetVMwareImages(ctx, -1, false); err == nil {
		t.Error("GetVMwareImages must reject a negative location_id")
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

// A network that still has servers or gateways attached cannot be deleted: the
// API answers -19511, and the caller has to tell that apart from other refusals
// to be able to name what to detach.
func TestIsNetworkInUse(t *testing.T) {
	inUse := &RequestError{
		Status:     "400 Bad Request",
		StatusCode: http.StatusBadRequest,
		Codes:      []int{APICodeNetworkInUse},
	}
	if !IsNetworkInUse(inUse) {
		t.Error("IsNetworkInUse must match -19511")
	}
	if IsNetworkInUse(&RequestError{StatusCode: http.StatusBadRequest, Codes: []int{APICodeConflict}}) {
		t.Error("IsNetworkInUse must not match other codes")
	}
	if IsNetworkInUse(nil) {
		t.Error("IsNetworkInUse(nil) must be false")
	}
}

// -12043 ("no free network at the moment") and -12042 ("the capacity is invalid")
// arrive the same way but mean opposite things: the first says the request was
// fine and the location is exhausted, the second that the size is not offered.
// The predicate must not blur them.
func TestIsVmwareNoFreePublicNetwork(t *testing.T) {
	exhausted := &RequestError{
		Status:     "400 Bad Request",
		StatusCode: http.StatusBadRequest,
		Codes:      []int{APICodeVmwareNoFreePublicNetwork},
	}
	if !IsVmwareNoFreePublicNetwork(exhausted) {
		t.Error("IsVmwareNoFreePublicNetwork must match -12043")
	}
	badCapacity := &RequestError{
		Status:     "400 Bad Request",
		StatusCode: http.StatusBadRequest,
		Codes:      []int{APICodeVmwareInvalidPublicNetworkCapacity},
	}
	if IsVmwareNoFreePublicNetwork(badCapacity) {
		t.Error("IsVmwareNoFreePublicNetwork must not match -12042 (invalid capacity)")
	}
	if IsVmwareNoFreePublicNetwork(nil) {
		t.Error("IsVmwareNoFreePublicNetwork(nil) must be false")
	}
}

// The two switch routes are fixed by the contract:
// POST vmware/servers/{id}/nested-hypervisor/{enable|disable}. The id is also
// validated before any request is built — the action has no body, so there is no
// request Validate to do it.
func TestVmwareServerNestedHypervisorPaths(t *testing.T) {
	type call struct {
		method string
		path   string
	}
	var (
		mu    sync.Mutex
		calls []call
	)
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls = append(calls, call{r.Method, r.URL.Path})
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"task_id":"vmw77"}`))
	}))
	ctx := context.Background()

	if _, err := client.EnableVmwareServerNestedHypervisor(ctx, 42); err != nil {
		t.Fatalf("EnableVmwareServerNestedHypervisor: %v", err)
	}
	if _, err := client.DisableVmwareServerNestedHypervisor(ctx, 42); err != nil {
		t.Fatalf("DisableVmwareServerNestedHypervisor: %v", err)
	}

	want := []call{
		{http.MethodPost, "/api/v1/vmware/servers/42/nested-hypervisor/enable"},
		{http.MethodPost, "/api/v1/vmware/servers/42/nested-hypervisor/disable"},
	}
	mu.Lock()
	got := calls
	mu.Unlock()
	if len(got) != len(want) {
		t.Fatalf("got %d request(s) %v, want %v", len(got), got, want)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("request %d = %s %s, want %s %s", i, got[i].method, got[i].path, w.method, w.path)
		}
	}

	// A bad server id must be rejected locally, in the ...AndWait form too.
	guard := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("request must not reach the API: %s %s", r.Method, r.URL.Path)
	}))
	if _, err := guard.EnableVmwareServerNestedHypervisor(ctx, 0); err == nil {
		t.Error("EnableVmwareServerNestedHypervisor must reject server id 0")
	}
	if _, err := guard.DisableVmwareServerNestedHypervisorAndWait(ctx, -1); err == nil {
		t.Error("DisableVmwareServerNestedHypervisorAndWait must reject a negative server id")
	}
}

// The three refusals of the nested-virtualization switch are distinct business
// outcomes and the caller has to be able to name them: -8149 is "the server has
// a GPU" (nothing to retry, the exclusion is permanent), -8154 is "resume the
// server first" (retryable by the user), -8155 is "this location has no VDC with
// the capability" (predictable from VmwareLocation.NestedHypervisorSupported).
// A predicate that blurs them would leave the provider with one opaque error.
func TestVmwareNestedHypervisorErrorPredicates(t *testing.T) {
	// The codes themselves are fixed by the contract, not by the SDK.
	fixed := map[string]struct {
		got  int
		want int
	}{
		"APICodeVmwareOperationNotSupportedForGpuServer":      {APICodeVmwareOperationNotSupportedForGpuServer, -8149},
		"APICodeVmwareServerIsSuspended":                      {APICodeVmwareServerIsSuspended, -8154},
		"APICodeVmwareNestedHypervisorNotSupportedInLocation": {APICodeVmwareNestedHypervisorNotSupportedInLocation, -8155},
	}
	for name, c := range fixed {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", name, c.got, c.want)
		}
	}

	predicates := map[string]struct {
		match func(error) bool
		code  int
	}{
		"IsVmwareOperationNotSupportedForGpuServer":      {IsVmwareOperationNotSupportedForGpuServer, APICodeVmwareOperationNotSupportedForGpuServer},
		"IsVmwareServerSuspended":                        {IsVmwareServerSuspended, APICodeVmwareServerIsSuspended},
		"IsVmwareNestedHypervisorNotSupportedInLocation": {IsVmwareNestedHypervisorNotSupportedInLocation, APICodeVmwareNestedHypervisorNotSupportedInLocation},
	}
	for name, p := range predicates {
		for _, code := range []int{
			APICodeVmwareOperationNotSupportedForGpuServer,
			APICodeVmwareServerIsSuspended,
			APICodeVmwareNestedHypervisorNotSupportedInLocation,
			APICodeConflict,
		} {
			err := &RequestError{
				Status:     "400 Bad Request",
				StatusCode: http.StatusBadRequest,
				Codes:      []int{code},
			}
			want := code == p.code
			if got := p.match(err); got != want {
				t.Errorf("%s(code %d) = %v, want %v", name, code, got, want)
			}
		}
		if p.match(nil) {
			t.Errorf("%s(nil) must be false", name)
		}
	}

	// End to end: the switch wraps its error with %w, so the predicate has to
	// keep working on what the method actually returns.
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"errors":[{"code":-8154,"message":"The operation is not available for a suspended VM"}]}`))
	}))
	_, err := client.EnableVmwareServerNestedHypervisorAndWait(context.Background(), 42)
	if err == nil {
		t.Fatal("a refused switch must return an error")
	}
	if !IsVmwareServerSuspended(err) {
		t.Errorf("IsVmwareServerSuspended = false for the error the method returns: %v", err)
	}
	if IsVmwareOperationNotSupportedForGpuServer(err) {
		t.Errorf("a suspended-server refusal must not look like the GPU refusal: %v", err)
	}
}

// The switch is asynchronous and its task belongs to the VMware family: it is
// polled through GET /tasks/{id} with the state in "is_completed", not through
// the base task channel. ...AndWait must wait for the task and then report the
// server as it is after the switch.
func TestEnableVmwareServerNestedHypervisorAndWait(t *testing.T) {
	var taskPolls, serverReads int32

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost:
			_, _ = w.Write([]byte(`{"task_id":"vmw77"}`))
		case strings.HasPrefix(r.URL.Path, "/api/v1/tasks/"):
			// Not settled on the first poll: the saga power-cycles the guest.
			state := entities.VmwareTaskStateInProgress
			if atomic.AddInt32(&taskPolls, 1) >= 2 {
				state = entities.VmwareTaskStateCompleted
			}
			_, _ = w.Write([]byte(`{"task":{"id":"vmw77","is_completed":"` + state + `"}}`))
		default:
			atomic.AddInt32(&serverReads, 1)
			_, _ = w.Write([]byte(`{"server":{"id":42,"state":"active","is_power_on":true,"nested_hypervisor":true}}`))
		}
	}))

	server, err := client.EnableVmwareServerNestedHypervisorAndWait(context.Background(), 42)
	if err != nil {
		t.Fatalf("EnableVmwareServerNestedHypervisorAndWait: %v", err)
	}
	if !server.NestedHypervisor {
		t.Errorf("the refreshed server must report nested_hypervisor, got %+v", server)
	}
	if got := atomic.LoadInt32(&taskPolls); got < 2 {
		t.Errorf("polled the task %d time(s), expected to keep polling while InProgress", got)
	}
	if got := atomic.LoadInt32(&serverReads); got != 1 {
		t.Errorf("read the server %d time(s), want 1 (after the task)", got)
	}
}

// The idempotent outcome (spec clarification 12): switching to the state the
// server is already in answers HTTP 200 with "task_id": null. There is no task,
// so the raw method reports no task at all and ...AndWait must return the
// current server state instead of awaiting an empty task id.
func TestVmwareServerNestedHypervisorAndWaitIdempotent(t *testing.T) {
	var serverReads int32

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost:
			_, _ = w.Write([]byte(`{"task_id":null}`))
		case strings.HasPrefix(r.URL.Path, "/api/v1/tasks/"):
			t.Errorf("there is no task to await, but %s was polled", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		default:
			atomic.AddInt32(&serverReads, 1)
			_, _ = w.Write([]byte(`{"server":{"id":42,"state":"powered_off","nested_hypervisor":false}}`))
		}
	}))
	ctx := context.Background()

	task, err := client.DisableVmwareServerNestedHypervisor(ctx, 42)
	if err != nil {
		t.Fatalf("DisableVmwareServerNestedHypervisor: %v", err)
	}
	// A non-nil reference to an empty id would be handed to the task waiter by
	// callers that keep the raw task.
	if task != nil {
		t.Errorf("an answer without a task must report no task, got %+v", task)
	}

	server, err := client.DisableVmwareServerNestedHypervisorAndWait(ctx, 42)
	if err != nil {
		t.Fatalf("DisableVmwareServerNestedHypervisorAndWait: %v", err)
	}
	if server == nil || server.ID != 42 {
		t.Fatalf("the current server state must be returned, got %+v", server)
	}
	if server.NestedHypervisor {
		t.Errorf("the server reports nested_hypervisor after disabling: %+v", server)
	}
	if got := atomic.LoadInt32(&serverReads); got != 1 {
		t.Errorf("read the server %d time(s), want 1", got)
	}
}
