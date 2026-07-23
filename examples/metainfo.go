package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	sdk "github.com/itglobalcom/vstack-cloud-panel-sdk"
)

func runMetadataExample(ctx context.Context, client *sdk.CloudClient) {
	fmt.Println("=== Metadata Operations Example ===")

	// Get project information
	fmt.Println("\n=== Getting project information ===")
	project, err := client.GetProject(ctx)
	if err != nil {
		log.Fatalf("Failed to get project: %v", err)
	}
	fmt.Printf("Project Details:\n")
	fmt.Printf("  ID: %d\n", project.ID)
	fmt.Printf("  Balance: %.2f %s\n", project.Balance, project.Currency)
	fmt.Printf("  State: %s\n", project.State)
	fmt.Printf("  Created: %s\n", project.Created)

	// Get a list of available locations
	fmt.Println("\n=== Getting available locations ===")
	locations, err := client.GetLocations(ctx)
	if err != nil {
		log.Fatalf("Failed to get locations: %v", err)
	}
	fmt.Printf("Available locations: %d\n\n", len(locations))

	for i, loc := range locations {
		fmt.Printf("%d. Location: %s\n", i+1, loc.ID)
		fmt.Printf("   System volume: %d - %d MB (%d - %d GB)\n",
			loc.SystemVolumeMin, loc.VolumeMax,
			loc.SystemVolumeMin/1024, loc.VolumeMax/1024)
		fmt.Printf("   Additional volume: min %d MB (%d GB)\n",
			loc.AdditionalVolumeMin, loc.AdditionalVolumeMin/1024)
		fmt.Printf("   Windows system volume: min %d MB (%d GB)\n",
			loc.WindowsSystemVolumeMin, loc.WindowsSystemVolumeMin/1024)
		fmt.Printf("   Bandwidth: %d - %d Mbps\n",
			loc.BandwidthMin, loc.BandwidthMax)
		fmt.Printf("   CPU options: %v\n", loc.CPUQuantityOptions)
		fmt.Printf("   RAM options (MB): %v\n", loc.RAMSizeOptions)
		fmt.Println()
	}

	// Get a list of available images
	fmt.Println("=== Getting available images ===")
	images, err := client.GetImages(ctx)
	if err != nil {
		log.Fatalf("Failed to get images: %v", err)
	}
	fmt.Printf("Total images: %d\n\n", len(images))

	// Group images by type and location
	imagesByTypeAndLocation := make(map[string]map[string][]string)
	for _, img := range images {
		if imagesByTypeAndLocation[img.Type] == nil {
			imagesByTypeAndLocation[img.Type] = make(map[string][]string)
		}
		imagesByTypeAndLocation[img.Type][img.LocationID] = append(
			imagesByTypeAndLocation[img.Type][img.LocationID],
			fmt.Sprintf("%s (%s, SSH: %v)", img.ID, img.OSVersion, img.AllowSSHKeys),
		)
	}

	// Show images by type
	for osType, locationMap := range imagesByTypeAndLocation {
		fmt.Printf("%s images:\n", osType)
		totalCount := 0
		for _, imgs := range locationMap {
			totalCount += len(imgs)
		}
		fmt.Printf("  Total: %d images\n", totalCount)

		// Show the first few from each location
		for locID, imgs := range locationMap {
			fmt.Printf("  Location %s: %d images\n", locID, len(imgs))
			displayCount := len(imgs)
			if displayCount > 3 {
				displayCount = 3
			}
			for i := 0; i < displayCount; i++ {
				fmt.Printf("    - %s\n", imgs[i])
			}
			if len(imgs) > 3 {
				fmt.Printf("    ... and %d more\n", len(imgs)-3)
			}
		}
		fmt.Println()
	}

	// Get a list of all applications
	fmt.Println("=== Getting all available applications ===")
	allApps, err := client.GetApplications(ctx, "")
	if err != nil {
		log.Fatalf("Failed to get applications: %v", err)
	}

	// Group by location
	appsByLocation := make(map[string][]string)
	for _, app := range allApps {
		appsByLocation[app.LocationID] = append(appsByLocation[app.LocationID], app.ID)
	}

	fmt.Printf("Total applications: %d\n\n", len(allApps))
	for locID, apps := range appsByLocation {
		fmt.Printf("Location %s: %d applications\n", locID, len(apps))
		fmt.Printf("  %s\n", strings.Join(apps, ", "))
		fmt.Println()
	}

	// Show details of several applications
	fmt.Println("=== Application details (first 5) ===")
	displayCount := len(allApps)
	if displayCount > 5 {
		displayCount = 5
	}
	for i := 0; i < displayCount; i++ {
		app := allApps[i]
		fmt.Printf("\n%d. Application: %s\n", i+1, app.ID)
		fmt.Printf("   Location: %s\n", app.LocationID)
		fmt.Printf("   Compatible images (%d): %s\n",
			len(app.Images), strings.Join(app.Images[:min(3, len(app.Images))], ", "))
		if len(app.Images) > 3 {
			fmt.Printf("   ... and %d more images\n", len(app.Images)-3)
		}
	}

	// Get applications for a specific location
	if len(locations) > 0 {
		locationID := locations[0].ID
		fmt.Printf("\n=== Applications for location %s ===\n", locationID)
		locationApps, err := client.GetApplications(ctx, locationID)
		if err != nil {
			log.Fatalf("Failed to get applications for location: %v", err)
		}
		fmt.Printf("Available: %d applications\n\n", len(locationApps))
		for i, app := range locationApps {
			fmt.Printf("  %d. %s (supports %d images)\n",
				i+1, app.ID, len(app.Images))
		}
	}

	fmt.Println("\n=== Metadata operations completed ===")
}
