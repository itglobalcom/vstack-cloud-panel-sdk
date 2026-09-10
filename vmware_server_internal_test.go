package sdk

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

func TestBuildVmwareServerPathNestedHypervisor(t *testing.T) {
	cases := map[string]string{
		"enable":  "vmware/servers/48214/nested-hypervisor/enable",
		"disable": "vmware/servers/48214/nested-hypervisor/disable",
	}
	for action, want := range cases {
		if got := buildVmwareServerPath(48214, "nested-hypervisor", action); got != want {
			t.Errorf("buildVmwareServerPath(48214, %q) = %q, want %q", action, got, want)
		}
	}
}

// A broken server id must be refused before the request is built.
func TestNestedHypervisorRejectsInvalidServerID(t *testing.T) {
	client := newTestClient(t, noRequestHandler(t))
	ctx := context.Background()

	if _, err := client.EnableVmwareServerNestedHypervisor(ctx, 0); err == nil {
		t.Error("EnableVmwareServerNestedHypervisor must reject server ID 0")
	}
	if _, err := client.DisableVmwareServerNestedHypervisor(ctx, -1); err == nil {
		t.Error("DisableVmwareServerNestedHypervisor must reject a negative server ID")
	}
	if _, err := client.EnableVmwareServerNestedHypervisorAndWait(ctx, 0); err == nil {
		t.Error("EnableVmwareServerNestedHypervisorAndWait must reject server ID 0")
	}
}

func TestEnableVmwareServerNestedHypervisor(t *testing.T) {
	var path atomic.Value

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path.Store(r.Method + " " + r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"task_id":"vmw901"}`))
	}))

	task, err := client.EnableVmwareServerNestedHypervisor(context.Background(), 48214)
	if err != nil {
		t.Fatalf("enable: %v", err)
	}
	if task.IsZero() || task.ID != "vmw901" {
		t.Errorf("task = %+v, want id vmw901", task)
	}
	want := "POST /api/v1/vmware/servers/48214/nested-hypervisor/enable"
	if got := path.Load(); got != want {
		t.Errorf("request = %v, want %q", got, want)
	}
}

// An idempotent outcome creates no task: the API answers 200 with an explicit
// "task_id": null (the one place it overrides its global "omit nulls"), which
// must decode to an empty reference rather than fail or look like a task.
func TestNestedHypervisorIdempotentOutcome(t *testing.T) {
	var reads int32

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&reads, 1)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			_, _ = w.Write([]byte(`{"task_id":null}`))
			return
		}
		_, _ = w.Write([]byte(`{"server":{"id":48214,"name":"srv","state":"active","nested_hypervisor":true}}`))
	}))
	ctx := context.Background()

	task, err := client.DisableVmwareServerNestedHypervisor(ctx, 48214)
	if err != nil {
		t.Fatalf("disable: %v", err)
	}
	if !task.IsZero() {
		t.Errorf("task = %+v, want an empty reference", task)
	}

	// Awaiting an empty reference must not poll anything.
	awaited, err := client.WaitVmwareTaskRef(ctx, task)
	if err != nil {
		t.Fatalf("WaitVmwareTaskRef: %v", err)
	}
	if awaited != nil {
		t.Errorf("WaitVmwareTaskRef = %+v, want nil", awaited)
	}

	// The ...AndWait form still refreshes the server: one POST plus one GET, no
	// task read in between.
	atomic.StoreInt32(&reads, 0)
	server, err := client.EnableVmwareServerNestedHypervisorAndWait(ctx, 48214)
	if err != nil {
		t.Fatalf("enable and wait: %v", err)
	}
	if !server.NestedHypervisor {
		t.Error("refreshed server must report nested_hypervisor")
	}
	if got := atomic.LoadInt32(&reads); got != 2 {
		t.Errorf("got %d requests, want 2 (the mutation and the refresh)", got)
	}
}

// nested_hypervisor is part of both sides of the server contract.
func TestParseVmwareServerNestedHypervisor(t *testing.T) {
	body := []byte(`{"server":{"id":48214,"project_id":11,"location_id":7,"name":"srv","state":"active","cpu":2,"ram_mb":4096,"system_disk_mb":61440,"image_id":42,"is_power_on":true,"nested_hypervisor":true,"nics":[]}}`)

	var resp vmwareServerResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Server.NestedHypervisor {
		t.Errorf("NestedHypervisor = false, want true: %+v", resp.Server)
	}

	// The field is absent for a server that never had it, which is false.
	var off vmwareServerResponse
	if err := json.Unmarshal([]byte(`{"server":{"id":1,"name":"srv","state":"active"}}`), &off); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if off.Server.NestedHypervisor {
		t.Error("an absent nested_hypervisor must decode to false")
	}
}

// The order carries nested_hypervisor only when the caller asked for it, so an
// order that does not mention the feature stays byte-identical to before.
func TestVmwareCreateServerRequestNestedHypervisor(t *testing.T) {
	req := &entities.VmwareCreateServerRequest{
		LocationID:       7,
		Name:             "srv",
		ImageID:          42,
		CPUCount:         2,
		RamMB:            4096,
		SystemDiskSizeMB: 61440,
	}

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(body), "nested_hypervisor") {
		t.Errorf("an order that does not set the field must omit it: %s", body)
	}

	enabled := true
	req.NestedHypervisor = &enabled
	body, err = json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(body), `"nested_hypervisor":true`) {
		t.Errorf("an order with the field set must send it: %s", body)
	}
}

// nested_hypervisor_supported says where the feature can be ordered at all; it
// travels in the location catalog next to gpu_supported.
func TestParseVmwareLocationNestedHypervisorSupported(t *testing.T) {
	body := []byte(`{"locations":[
		{"id":7,"tech_title":"minsk","gpu_supported":true,"nested_hypervisor_supported":true},
		{"id":2,"tech_title":"ds-msk","gpu_supported":false,"nested_hypervisor_supported":false}
	]}`)

	var resp vmwareLocationsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Locations) != 2 {
		t.Fatalf("got %d locations, want 2", len(resp.Locations))
	}
	if !resp.Locations[0].NestedHypervisorSupported {
		t.Errorf("location 7 must support nested virtualization: %+v", resp.Locations[0])
	}
	if resp.Locations[1].NestedHypervisorSupported {
		t.Errorf("location 2 must not support nested virtualization: %+v", resp.Locations[1])
	}
}
