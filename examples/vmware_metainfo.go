package main

import (
	"context"
	"fmt"
	"log"
	"time"

	sdk "github.com/itglobalcom/vstack-cloud-panel-sdk"
	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

// runVmwareMetainfoExample walks through the read-only VMware Cloud catalogs and
// the task helpers. Everything a server or network request needs — location id,
// disk type name and size limits, image id, GPU triple — comes from here, so it is
// the natural starting point.
//
// Methods covered:
//
//	GetVmwareLocationList
//	GetVmwareImageList        (no filter, GPU-required, GPU-unsupported)
//	GetVmwareGPUModelList     (all locations, one location)
//	GetVmwareTask
//	WaitVmwareTask / WaitVmwareTaskWithTimeout / WaitVmwareTaskRef
func runVmwareMetainfoExample(ctx context.Context, client *sdk.CloudClient) {
	fmt.Println("=== VMware Cloud metadata example ===")

	// ---------- Locations ----------
	section("GetVmwareLocationList")
	locations, err := client.GetVmwareLocationList(ctx)
	if err != nil {
		log.Fatalf("GetVmwareLocationList: %v", err)
	}
	step("locations: %d", len(locations))
	for _, loc := range locations {
		step("#%d %s — GPU supported: %v, nested hypervisor supported: %v, disk types: %d",
			loc.ID, loc.TechTitle, loc.GPUSupported, loc.NestedHypervisorSupported, len(loc.DiskTypes))
		// Disk types are published per location, not as a standalone
		// catalog. Title is the value the create/verify requests take, and the
		// limits are in MB to match system_disk_size_mb / size_mb.
		for _, dt := range loc.DiskTypes {
			step("    %-8s default=%-5v ssd=%-5v system=%-5v %d..%d MB step %d, default %d MB",
				dt.Title, dt.IsDefault, dt.IsSSD, dt.IsAllowedForSystemDisk,
				dt.MinMB, dt.MaxMB, dt.StepMB, dt.DefaultSizeMB)
		}
	}

	wanted := vmwareExampleLocation()
	location := findVmwareLocation(locations, wanted)
	if location == nil {
		log.Fatalf("location %q not found (set VMWARE_LOCATION to one of the tech_titles above)", wanted)
	}
	fmt.Printf("\nSelected location: %s (id %d)\n", location.TechTitle, location.ID)
	locationID := &location.ID

	// ---------- Images ----------
	section("GetVmwareImageList — no GPU filter")
	images, err := client.GetVmwareImageList(ctx, locationID, nil)
	if err != nil {
		log.Fatalf("GetVmwareImageList: %v", err)
	}
	step("images in %s: %d", location.TechTitle, len(images))
	for i, img := range images {
		if i >= 5 {
			step("... and %d more", len(images)-i)
			break
		}
		step("#%d %s — %s/%s, min %d MB RAM, %d GB disk, ssh=%v cpu_hot=%v ram_hot=%v nic_hot_remove=%v gpu_only=%v models=%v",
			img.ID, img.Name, img.OsFamily, img.OsType, img.MinRamMB, img.HddGB,
			img.SSHKeySupported, img.CPUHotAdd, img.MemoryHotAdd, img.NICHotRemove,
			img.IsGPUOnly, img.SupportedGPUModelIDs)
	}

	// The gpu filter is a two-value enum. Any other value is a 400, and an
	// empty one makes the API answer 500 — which is why the SDK omits the
	// parameter instead of sending it empty.
	section("GetVmwareImageList — gpu=%q", entities.VmwareImageGPURequired)
	gpuImages, err := client.GetVmwareImageList(ctx, locationID, strPtr(entities.VmwareImageGPURequired))
	if err != nil {
		log.Fatalf("GetVmwareImageList(gpu=required): %v", err)
	}
	step("GPU-only images: %d", len(gpuImages))
	for _, img := range gpuImages {
		step("#%d %s — supported GPU models: %v", img.ID, img.Name, img.SupportedGPUModelIDs)
	}

	section("GetVmwareImageList — gpu=%q", entities.VmwareImageGPUUnsupported)
	nonGPUImages, err := client.GetVmwareImageList(ctx, locationID, strPtr(entities.VmwareImageGPUUnsupported))
	if err != nil {
		log.Fatalf("GetVmwareImageList(gpu=unsupported): %v", err)
	}
	step("images that cannot use a GPU: %d", len(nonGPUImages))

	// An unknown location is reported as HTTP 400 with code -8049, not as a 404,
	// so IsNotFound does not see it — IsInvalidLocation does.
	section("GetVmwareImageList — unknown location (error handling)")
	if _, err := client.GetVmwareImageList(ctx, intPtr(999999), nil); err != nil {
		step("error: %v", err)
		step("IsNotFound=%v IsInvalidLocation=%v", sdk.IsNotFound(err), sdk.IsInvalidLocation(err))
	} else {
		step("unexpected success for location 999999")
	}

	// ---------- GPU models ----------
	//
	// The unit of this listing is a slicing profile, not a model, so the
	// same ID comes back once per (vRAM, card count) variant — key by the triple.
	section("GetVmwareGPUModelList — location %s", location.TechTitle)
	gpuModels, err := client.GetVmwareGPUModelList(ctx, locationID)
	if err != nil {
		log.Fatalf("GetVmwareGPUModelList: %v", err)
	}
	step("GPU slicing profiles in %s: %d", location.TechTitle, len(gpuModels))
	for _, g := range gpuModels {
		step("id=%d %s (%s) — vRAM %d MB, cards %d, per-card limit %d, max server RAM %s MB, available %s",
			g.ID, g.Name, g.TechTitle, g.CapacityVramMB, g.GPUCardCount,
			g.ServerAllocationLimit, derefInt(g.MaxServerRamMB), derefBool(g.IsAvailable))
	}

	section("GetVmwareGPUModelList — all locations")
	allGPUModels, err := client.GetVmwareGPUModelList(ctx, nil)
	if err != nil {
		log.Fatalf("GetVmwareGPUModelList(nil): %v", err)
	}
	step("GPU slicing profiles across all locations: %d", len(allGPUModels))
	// Demonstrate the id collision: count how many distinct IDs there are.
	distinct := map[int]int{}
	for _, g := range allGPUModels {
		distinct[g.ID]++
	}
	step("distinct ids: %d (an id repeats up to %d times — key by (id, vram_mb, card_count))",
		len(distinct), maxCount(distinct))

	// ---------- Tasks ----------
	//
	// VMware task ids live in their own space ("vmw{N}") and the SDK keeps them
	// apart from base ids both by type (*VmwareTaskID vs *TaskID) and at runtime.
	section("Task helpers")
	step("VmwareTaskIDPrefix=%q  IsVmwareTaskID(\"vmw123\")=%v  IsVmwareTaskID(\"l1t2\")=%v",
		sdk.VmwareTaskIDPrefix, sdk.IsVmwareTaskID("vmw123"), sdk.IsVmwareTaskID("l1t2"))
	step("WaitVmwareTask timeout floor: %v", sdk.VmwareTaskWaitDefaultTimeout)

	// Passing a base id to a VMware helper (or the other way round) is refused up
	// front rather than producing a confusing unmarshal error or a wait that hangs
	// on an already-finished task.
	if _, err := client.GetVmwareTask(ctx, "l1t2"); err != nil {
		step("GetVmwareTask(\"l1t2\") rejected: %v", err)
	}
	if _, err := client.GetTask(ctx, "vmw1"); err != nil {
		step("GetTask(\"vmw1\") rejected: %v", err)
	}

	// A nil task reference is the "answered synchronously, nothing to await" case
	// and every wait helper tolerates it.
	var noTask *sdk.VmwareTaskID
	step("(*VmwareTaskID)(nil).IsZero()=%v String()=%q", noTask.IsZero(), noTask.String())
	if task, err := client.WaitVmwareTaskRef(ctx, noTask); err != nil {
		log.Fatalf("WaitVmwareTaskRef(nil): %v", err)
	} else {
		step("WaitVmwareTaskRef(nil) -> task=%v err=nil", task)
	}

	// GetVmwareTask / WaitVmwareTask against a real id, if one was provided.
	// Any mutating VMware call hands one back; the server and network examples
	// below drive them end to end.
	section("GetVmwareTask / WaitVmwareTask")
	if _, err := client.GetVmwareTask(ctx, "vmw1"); err != nil {
		step("GetVmwareTask(\"vmw1\") -> %v (IsNotFound=%v)", err, sdk.IsNotFound(err))
	}
	if _, err := client.WaitVmwareTaskWithTimeout(ctx, "vmw1", 5*time.Second); err != nil {
		step("WaitVmwareTaskWithTimeout(\"vmw1\", 5s) -> %v", err)
	}
	step("WaitVmwareTask / WaitVmwareTaskRef on live tasks: see the vmware_server and vmware_network examples")

	fmt.Println("\n=== VMware metadata example completed ===")
}

// maxCount returns the largest value in the map, or 0 for an empty one.
func maxCount(m map[int]int) int {
	max := 0
	for _, v := range m {
		if v > max {
			max = v
		}
	}
	return max
}
