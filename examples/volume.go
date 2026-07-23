package main

import (
	"context"
	"fmt"
	"log"

	sdk "github.com/itglobalcom/vstack-cloud-panel-sdk"
	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

func runVolumeExample(ctx context.Context, client *sdk.CloudClient) {
	fmt.Println("=== Server Volume Operations Example ===")

	// Create a server with one boot volume
	fmt.Println("=== Creating server with boot volume ===")
	createReq := &entities.CreateServerRequest{
		Name:       "test-server-volumes",
		LocationID: "kz",
		ImageID:    "Debian-12-X64",
		CPU:        2,
		RamMB:      2048,
		Volumes: []entities.VolumeSpec{
			{
				Name:   "boot",
				SizeMB: 25600, // 25 GB
			},
		},
		Networks: []entities.NetworkSpec{
			{
				BandwidthMbps: 100,
			},
		},
		Tags: []string{"test", "volume-example"},
	}

	server, err := client.CreateServerAndWait(ctx, createReq)

	if err != nil {
		log.Fatalf("[FATAL] Failed to wait for server creation: %v", err)
	}

	fmt.Printf("Server created successfully!\n")
	fmt.Printf("  ID: %s\n", server.ID)
	fmt.Printf("  Name: %s\n", server.Name)
	fmt.Printf("  State: %s\n", server.State)

	serverID := server.ID

	// Get the list of initial volumes (should only be the boot volume)
	fmt.Println("\n=== Checking initial volumes ===")
	volumes, err := client.GetServerVolumes(ctx, serverID)
	if err != nil {
		log.Fatalf("Failed to get initial volumes: %v", err)
	}
	fmt.Printf("Initial volumes: %d\n", len(volumes))
	for _, vol := range volumes {
		fmt.Printf("  - ID=%d, Name=%s, Size=%d GB, Created=%s\n",
			vol.ID, vol.Name, vol.SizeMB/1024, vol.Created)
	}

	// Add 3 additional volumes
	fmt.Println("\n=== Adding 3 additional volumes ===")

	// Volume 1: 10 GB
	fmt.Println("\n--- Creating Volume 1: 10 GB ---")
	volume1, err := client.CreateServerVolumeAndWait(ctx, serverID, &entities.CreateVolumeRequest{
		Name:   "data-volume-1",
		SizeMB: 10240, // 10 GB
	})
	if err != nil {
		log.Fatalf("Failed to create volume 1: %v", err)
	}
	fmt.Printf("✓ Volume 1 created: ID=%d, Name=%s, Size=%d GB\n",
		volume1.ID, volume1.Name, volume1.SizeMB/1024)

	// Volume 2: 20 GB
	fmt.Println("\n--- Creating Volume 2: 20 GB ---")
	volume2, err := client.CreateServerVolumeAndWait(ctx, serverID, &entities.CreateVolumeRequest{
		Name:   "data-volume-2",
		SizeMB: 20480, // 20 GB
	})
	if err != nil {
		log.Fatalf("Failed to create volume 2: %v", err)
	}
	fmt.Printf("✓ Volume 2 created: ID=%d, Name=%s, Size=%d GB\n",
		volume2.ID, volume2.Name, volume2.SizeMB/1024)

	// Volume 3: 30 GB
	fmt.Println("\n--- Creating Volume 3: 30 GB ---")
	volume3, err := client.CreateServerVolumeAndWait(ctx, serverID, &entities.CreateVolumeRequest{
		Name:   "data-volume-3",
		SizeMB: 30720, // 30 GB
	})
	if err != nil {
		log.Fatalf("Failed to create volume 3: %v", err)
	}
	fmt.Printf("✓ Volume 3 created: ID=%d, Name=%s, Size=%d GB\n",
		volume3.ID, volume3.Name, volume3.SizeMB/1024)

	// Check all volumes
	fmt.Println("\n=== Checking all volumes after creation ===")
	volumes, err = client.GetServerVolumes(ctx, serverID)
	if err != nil {
		log.Fatalf("Failed to get volumes: %v", err)
	}
	fmt.Printf("Total volumes: %d\n", len(volumes))

	var totalSize int
	for i, vol := range volumes {
		totalSize += vol.SizeMB
		fmt.Printf("  %d. ID=%d, Name=%-20s Size=%d GB\n",
			i+1, vol.ID, vol.Name, vol.SizeMB/1024)
	}
	fmt.Printf("Total storage: %d GB\n", totalSize/1024)

	// Get details of a specific volume
	fmt.Println("\n=== Getting detailed info for Volume 2 ===")
	volDetails, err := client.GetServerVolume(ctx, serverID, volume2.ID)
	if err != nil {
		log.Fatalf("Failed to get volume details: %v", err)
	}
	fmt.Printf("Volume %d Details:\n", volDetails.ID)
	fmt.Printf("  Server ID: %s\n", volDetails.ServerID)
	fmt.Printf("  Name: %s\n", volDetails.Name)
	fmt.Printf("  Size: %d MB (%d GB)\n", volDetails.SizeMB, volDetails.SizeMB/1024)
	fmt.Printf("  Created: %s\n", volDetails.Created)

	// Edit 2 volumes
	fmt.Println("\n=== Editing 2 volumes ===")

	// Resize Volume 1: 10 GB -> 20 GB
	fmt.Println("\n--- Resizing Volume 1: 10 GB -> 20 GB ---")
	updatedVol1, err := client.UpdateServerVolumeAndWait(ctx, serverID, volume1.ID, &entities.UpdateVolumeRequest{
		SizeMB: 20480, // 20 GB
	})
	if err != nil {
		log.Fatalf("Failed to resize volume 1: %v", err)
	}
	fmt.Printf("✓ Volume 1 resized: ID=%d, Size=%d -> %d GB\n",
		updatedVol1.ID, volume1.SizeMB/1024, updatedVol1.SizeMB/1024)

	// Rename and resize Volume 2: 20 GB -> 40 GB
	fmt.Println("\n--- Updating Volume 2: rename + resize to 40 GB ---")
	updatedVol2, err := client.UpdateServerVolumeAndWait(ctx, serverID, volume2.ID, &entities.UpdateVolumeRequest{
		Name:   "renamed-data-volume-2",
		SizeMB: 40960, // 40 GB
	})
	if err != nil {
		log.Fatalf("Failed to update volume 2: %v", err)
	}
	fmt.Printf("✓ Volume 2 updated: ID=%d\n", updatedVol2.ID)
	fmt.Printf("  Name: %s -> %s\n", volume2.Name, updatedVol2.Name)
	fmt.Printf("  Size: %d -> %d GB\n", volume2.SizeMB/1024, updatedVol2.SizeMB/1024)

	// Check volumes after editing
	fmt.Println("\n=== Checking volumes after editing ===")
	volumes, err = client.GetServerVolumes(ctx, serverID)
	if err != nil {
		log.Fatalf("Failed to get volumes: %v", err)
	}

	totalSize = 0
	for i, vol := range volumes {
		totalSize += vol.SizeMB
		fmt.Printf("  %d. ID=%d, Name=%-25s Size=%d GB\n",
			i+1, vol.ID, vol.Name, vol.SizeMB/1024)
	}
	fmt.Printf("Total storage after updates: %d GB\n", totalSize/1024)

	// Delete 3 additional volumes (leaving only the boot volume)
	fmt.Println("\n=== Deleting 3 additional volumes ===")

	// Delete Volume 1
	fmt.Println("\n--- Deleting Volume 1 ---")
	err = client.DeleteServerVolumeAndWait(ctx, serverID, volume1.ID)
	if err != nil {
		log.Fatalf("Failed to delete volume 1: %v", err)
	}
	fmt.Printf("✓ Volume 1 (ID=%d) deleted\n", volume1.ID)

	// Delete Volume 2
	fmt.Println("\n--- Deleting Volume 2 ---")
	err = client.DeleteServerVolumeAndWait(ctx, serverID, volume2.ID)
	if err != nil {
		log.Fatalf("Failed to delete volume 2: %v", err)
	}
	fmt.Printf("✓ Volume 2 (ID=%d) deleted\n", volume2.ID)

	// Delete Volume 3
	fmt.Println("\n--- Deleting Volume 3 ---")
	err = client.DeleteServerVolumeAndWait(ctx, serverID, volume3.ID)
	if err != nil {
		log.Fatalf("Failed to delete volume 3: %v", err)
	}
	fmt.Printf("✓ Volume 3 (ID=%d) deleted\n", volume3.ID)

	// Verify remaining volumes
	fmt.Println("\n=== Verifying remaining volumes ===")
	remainingVolumes, err := client.GetServerVolumes(ctx, serverID)
	if err != nil {
		log.Fatalf("Failed to get remaining volumes: %v", err)
	}
	fmt.Printf("Remaining volumes: %d\n", len(remainingVolumes))
	for _, vol := range remainingVolumes {
		fmt.Printf("  - ID=%d, Name=%s, Size=%d GB\n",
			vol.ID, vol.Name, vol.SizeMB/1024)
	}

	// Delete the server
	fmt.Println("\n=== Deleting server ===")
	err = client.DeleteServer(ctx, serverID)
	if err != nil {
		log.Fatalf("Failed to delete server: %v", err)
	}
	fmt.Printf("✓ Server %s deleted successfully\n", serverID)

	fmt.Println("\n=== Volume operations completed ===")
}
