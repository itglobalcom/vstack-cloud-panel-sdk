package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	sdk "github.com/itglobalcom/vstack-cloud-panel-sdk"
	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

// Shared scaffolding for the VMware Cloud examples.
//
// The three VMware examples (vmware_meta, vmware_server, vmware_network) between
// them call every exported Vmware* method of the SDK. They run against a live
// project, so each one provisions what it needs, exercises the methods and cleans
// up after itself.
//
// Environment (all optional):
//
//	VMWARE_LOCATION     location tech_title to work in           (default "minsk")
//	VMWARE_IMAGE_ID     image id for the servers being created   (default: first Linux image)
//	VMWARE_NETWORK_CIDR /24 base address of the test networks     (default "192.168.94.0")
//	VMWARE_KEEP         "1" to skip the teardown (leaves resources behind)
//	VMWARE_SKIP_LONG    "1" to skip copy and rebuild (~25 minutes of the run)

// defaultVmwareExampleLocation is the location the examples work in unless
// VMWARE_LOCATION says otherwise. Locations are matched against tech_title
// case-insensitively, so "minsk" picks "MinskBy".
const defaultVmwareExampleLocation = "minsk"

// defaultVmwareExampleCIDR is the /24 the example networks are carved out of.
const defaultVmwareExampleCIDR = "192.168.94.0"

// vmwareExampleServerPrefix is the name prefix every server the examples create
// carries. It is what lets the teardown recognize its own leftovers without ever
// touching a resource it did not create.
const vmwareExampleServerPrefix = "sdk-ex-"

// vmwareExampleLocation returns the tech_title the examples work with.
func vmwareExampleLocation() string {
	if loc := os.Getenv("VMWARE_LOCATION"); loc != "" {
		return loc
	}
	return defaultVmwareExampleLocation
}

// vmwareExampleKeepResources reports whether teardown should be skipped.
func vmwareExampleKeepResources() bool {
	return os.Getenv("VMWARE_KEEP") == "1"
}

// vmwareExampleSkipLong reports whether the multi-minute copy and rebuild steps
// should be skipped.
func vmwareExampleSkipLong() bool {
	return os.Getenv("VMWARE_SKIP_LONG") == "1"
}

// vmwareExampleNetworkAddress returns the /24 base address for the n-th example
// network, derived from VMWARE_NETWORK_CIDR by bumping the third octet.
func vmwareExampleNetworkAddress(n int) string {
	base := os.Getenv("VMWARE_NETWORK_CIDR")
	if base == "" {
		base = defaultVmwareExampleCIDR
	}
	octets := strings.Split(base, ".")
	if len(octets) != 4 {
		log.Fatalf("VMWARE_NETWORK_CIDR must be a dotted /24 base address like 192.168.94.0, got %q", base)
	}
	third, err := strconv.Atoi(octets[2])
	if err != nil {
		log.Fatalf("VMWARE_NETWORK_CIDR has a non-numeric third octet: %q", base)
	}
	return fmt.Sprintf("%s.%s.%d.0", octets[0], octets[1], third+n)
}

// section prints a titled step separator.
func section(format string, args ...any) {
	fmt.Printf("\n=== "+format+" ===\n", args...)
}

// announceBillableRun prints what the example is about to create and roughly how
// long it will take, before the first resource is ordered. These examples run
// against a live project and bill for what they provision, so the run should not
// come as a surprise to someone who typed `make example RESOURCE=...` to look
// around.
func announceBillableRun(what string, estimate string) {
	fmt.Printf(`
------------------------------------------------------------------
This example creates REAL, BILLABLE resources in your project:
  %s
Expected duration: %s.
It removes everything it created before exiting (VMWARE_KEEP=1 keeps it).
Use a dedicated test project.
------------------------------------------------------------------
`, what, estimate)
}

// step prints a single method call outcome.
func step(format string, args ...any) {
	fmt.Printf("  "+format+"\n", args...)
}

// intPtr / boolPtr / strPtr build pointers for the optional request fields. The
// VMware section uses pointers wherever the zero value is a meaningful input, so
// "not set" stays distinguishable from 0 / false / "".
func intPtr(v int) *int       { return &v }
func boolPtr(v bool) *bool    { return &v }
func strPtr(v string) *string { return &v }

// deref renders a *string for printing.
func deref(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

// derefInt renders a *int for printing.
func derefInt(i *int) string {
	if i == nil {
		return "<nil>"
	}
	return strconv.Itoa(*i)
}

// derefBool renders a *bool for printing.
func derefBool(b *bool) string {
	if b == nil {
		return "<nil>"
	}
	return strconv.FormatBool(*b)
}

// vmwareRun records the outcome of every method the example calls, so a single
// platform-level rejection (a power action the guest refuses, say) does not abort
// the whole walkthrough and the summary still shows what was and was not
// exercised.
type vmwareRun struct {
	ok      []string
	failed  []string
	skipped []string
}

// pass records a successful call.
func (r *vmwareRun) pass(method string, format string, args ...any) {
	r.ok = append(r.ok, method)
	fmt.Printf("  [ok]   %-42s %s\n", method, fmt.Sprintf(format, args...))
}

// fail records a failed call and keeps going.
func (r *vmwareRun) fail(method string, err error) {
	r.failed = append(r.failed, method)
	fmt.Printf("  [FAIL] %-42s %v\n", method, err)
}

// skip records a call that was deliberately not made.
func (r *vmwareRun) skip(method string, reason string) {
	r.skipped = append(r.skipped, method)
	fmt.Printf("  [skip] %-42s %s\n", method, reason)
}

// check records a call by its error, returning true when it succeeded.
func (r *vmwareRun) check(method string, err error, format string, args ...any) bool {
	if err != nil {
		r.fail(method, err)
		return false
	}
	r.pass(method, format, args...)
	return true
}

// summary prints the tally and returns the number of failures.
func (r *vmwareRun) summary(title string) int {
	fmt.Printf("\n=== %s: %d ok, %d failed, %d skipped ===\n",
		title, len(r.ok), len(r.failed), len(r.skipped))
	if len(r.failed) > 0 {
		fmt.Println("failed:")
		for _, m := range r.failed {
			fmt.Printf("  - %s\n", m)
		}
	}
	if len(r.skipped) > 0 {
		fmt.Println("skipped:")
		for _, m := range r.skipped {
			fmt.Printf("  - %s\n", m)
		}
	}
	return len(r.failed)
}

// findVmwareLocation looks a location up by tech_title, preferring an exact
// (case-insensitive) match and falling back to a substring match.
func findVmwareLocation(locations []*entities.VmwareLocation, techTitle string) *entities.VmwareLocation {
	for _, loc := range locations {
		if strings.EqualFold(loc.TechTitle, techTitle) {
			return loc
		}
	}
	needle := strings.ToLower(techTitle)
	for _, loc := range locations {
		if strings.Contains(strings.ToLower(loc.TechTitle), needle) {
			return loc
		}
	}
	return nil
}

// vmwareExampleEnv is the resolved catalog context the server and network
// examples build their requests from: which location to use, which disk type and
// which image.
type vmwareExampleEnv struct {
	Location *entities.VmwareLocation
	DiskType *entities.VmwareLocationDiskType
	Image    *entities.VmwareImage
}

// resolveVmwareExampleEnv reads the catalogs and picks a location, a system disk
// type and an image to build server orders from. Everything a create request
// needs comes from these two calls — there is no separate disk-type or
// storage-profile catalog any more.
func resolveVmwareExampleEnv(ctx context.Context, client *sdk.CloudClient) *vmwareExampleEnv {
	locations, err := client.GetVmwareLocationList(ctx)
	if err != nil {
		log.Fatalf("GetVmwareLocationList: %v", err)
	}
	wanted := vmwareExampleLocation()
	location := findVmwareLocation(locations, wanted)
	if location == nil {
		log.Fatalf("location %q not found among %d VMware locations (set VMWARE_LOCATION)", wanted, len(locations))
	}

	// Disk types travel inside the location; pick one usable for a system disk.
	var diskType *entities.VmwareLocationDiskType
	for _, dt := range location.DiskTypes {
		if !dt.IsAllowedForSystemDisk {
			continue
		}
		if diskType == nil || dt.IsDefault {
			diskType = dt
		}
	}
	if diskType == nil {
		log.Fatalf("location %s exposes no disk type allowed for a system disk", location.TechTitle)
	}

	// An explicit image wins; otherwise take the first non-GPU Linux image, which
	// is the cheapest thing to provision.
	images, err := client.GetVmwareImageList(ctx, &location.ID, strPtr(entities.VmwareImageGPUUnsupported))
	if err != nil {
		log.Fatalf("GetVmwareImageList: %v", err)
	}
	var image *entities.VmwareImage
	if raw := os.Getenv("VMWARE_IMAGE_ID"); raw != "" {
		id, err := strconv.Atoi(raw)
		if err != nil {
			log.Fatalf("VMWARE_IMAGE_ID must be numeric, got %q", raw)
		}
		for _, img := range images {
			if img.ID == id {
				image = img
				break
			}
		}
		if image == nil {
			log.Fatalf("image %d not offered in location %s", id, location.TechTitle)
		}
	} else {
		for _, img := range images {
			if img.OsFamily == "Linux" {
				image = img
				break
			}
		}
		if image == nil && len(images) > 0 {
			image = images[0]
		}
		if image == nil {
			log.Fatalf("location %s offers no images", location.TechTitle)
		}
	}

	fmt.Printf("Using location %s (id %d), disk type %q, image %q (id %d)\n",
		location.TechTitle, location.ID, diskType.Title, image.Name, image.ID)
	return &vmwareExampleEnv{Location: location, DiskType: diskType, Image: image}
}

// systemDiskSizeMB returns a system disk size that satisfies both the image and
// the disk type: at least the image's own footprint, rounded up to the disk
// type's step and never below its minimum.
func (e *vmwareExampleEnv) systemDiskSizeMB() int {
	size := e.DiskType.MinMB
	imageMB := e.Image.HddGB * 1024
	if imageMB > size {
		size = imageMB
	}
	if step := e.DiskType.StepMB; step > 0 && size%step != 0 {
		size = ((size / step) + 1) * step
	}
	return size
}

// createServerRequest builds a minimal valid order for the resolved catalog.
func (e *vmwareExampleEnv) createServerRequest(name string) *entities.VmwareCreateServerRequest {
	ramMB := 1024
	if e.Image.MinRamMB > ramMB {
		ramMB = e.Image.MinRamMB
	}
	return &entities.VmwareCreateServerRequest{
		LocationID:       e.Location.ID,
		Name:             name,
		ImageID:          e.Image.ID,
		CPUCount:         1,
		RamMB:            ramMB,
		SystemDiskSizeMB: e.systemDiskSizeMB(),
		SystemDiskType:   e.DiskType.Title,
	}
}
