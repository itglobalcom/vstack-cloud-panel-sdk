package main

import (
	"context"
	"fmt"
	"log"

	sdk "github.com/itglobalcom/vstack-cloud-panel-sdk"
)

// runVMwareExample reads the VMware catalog (api/v1/vmware). Every call is
// read-only, so the example is safe to run against a live project.
func runVMwareExample(ctx context.Context, client *sdk.CloudClient) {
	fmt.Println("=== VMware Catalog Example ===")

	// Get the VMware locations available to the project
	fmt.Println("\n=== Getting VMware locations ===")
	locations, err := client.GetVMwareLocations(ctx)
	if err != nil {
		log.Fatalf("Failed to get VMware locations: %v", err)
	}
	fmt.Printf("Available locations: %d\n", len(locations))
	for i, loc := range locations {
		fmt.Printf("%d. Location %d: %s (GPU supported: %v, nested hypervisor supported: %v)\n",
			i+1, loc.ID, loc.TechTitle, loc.GPUSupported, loc.NestedHypervisorSupported)
	}

	if len(locations) == 0 {
		fmt.Println("\nNo VMware locations available — nothing else to show")
		return
	}

	// The catalog filters are optional: 0 means "every location".
	locationID := locations[0].ID

	// Get the disk types allowed in the location
	fmt.Printf("\n=== Getting disk types of location %d ===\n", locationID)
	diskTypes, err := client.GetVMwareDiskTypes(ctx, locationID)
	if err != nil {
		// An unknown location_id is answered with 400, not an empty list.
		if sdk.IsInvalidLocation(err) {
			log.Fatalf("Unknown VMware location %d", locationID)
		}
		log.Fatalf("Failed to get VMware disk types: %v", err)
	}
	fmt.Printf("Disk types: %d\n", len(diskTypes))
	for _, dt := range diskTypes {
		fmt.Printf("  - %s (id %d): %d - %d GB, step %d, SSD: %v, system disk: %v\n",
			dt.Title, dt.ID, dt.MinGB, dt.MaxGB, dt.StepGB, dt.IsSSD, dt.IsAllowedForSystemDisk)
	}

	// Get the storage profiles of the location
	fmt.Printf("\n=== Getting storage profiles of location %d ===\n", locationID)
	profiles, err := client.GetVMwareStorageProfiles(ctx, locationID, 0)
	if err != nil {
		log.Fatalf("Failed to get VMware storage profiles: %v", err)
	}
	fmt.Printf("Storage profiles: %d\n", len(profiles))
	for _, p := range profiles {
		fmt.Printf("  - %s (id %d): disk type %d, default: %v, free %d GB\n",
			p.Name, p.ID, p.DiskTypeID, p.IsDefault, p.FreeSpaceGB)
	}

	// Get the GPU models available to the partner
	fmt.Printf("\n=== Getting GPU models of location %d ===\n", locationID)
	gpuModels, err := client.GetVMwareGPUModels(ctx, locationID)
	if err != nil {
		log.Fatalf("Failed to get VMware GPU models: %v", err)
	}
	fmt.Printf("GPU models: %d\n", len(gpuModels))
	for _, m := range gpuModels {
		fmt.Printf("  - %s (%s): %d MB VRAM, %d cards, up to %d per server, max RAM %d MB, available: %v\n",
			m.Name, m.TechTitle, m.CapacityVramMB, m.GPUCardCount,
			m.ServerAllocationLimit, m.MaxServerRamMB, m.IsAvailable)
	}

	// Get the OS images of the location
	fmt.Printf("\n=== Getting images of location %d ===\n", locationID)
	images, err := client.GetVMwareImages(ctx, locationID, false)
	if err != nil {
		log.Fatalf("Failed to get VMware images: %v", err)
	}
	fmt.Printf("Images: %d\n", len(images))
	displayCount := len(images)
	if displayCount > 5 {
		displayCount = 5
	}
	for i := 0; i < displayCount; i++ {
		img := images[i]
		fmt.Printf("  - %s (id %d): %s / %s, min RAM %d MB, system disk %d GB, SSH keys: %v\n",
			img.Name, img.ID, img.OSFamily, img.OSType, img.MinRamMB, img.HddGB, img.SSHKeySupported)
	}
	if len(images) > displayCount {
		fmt.Printf("  ... and %d more images\n", len(images)-displayCount)
	}

	// GPU-only images are requested with the gpu_only filter
	fmt.Printf("\n=== Getting GPU-only images of location %d ===\n", locationID)
	gpuImages, err := client.GetVMwareImages(ctx, locationID, true)
	if err != nil {
		log.Fatalf("Failed to get GPU-only VMware images: %v", err)
	}
	fmt.Printf("GPU-only images: %d\n", len(gpuImages))
	for _, img := range gpuImages {
		fmt.Printf("  - %s (id %d): supported GPU models %v\n",
			img.Name, img.ID, img.SupportedGPUModelIDs)
	}

	fmt.Println("\n=== VMware catalog operations completed ===")
}
