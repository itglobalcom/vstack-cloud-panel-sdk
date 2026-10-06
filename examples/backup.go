package main

import (
	"context"
	"fmt"
	"log"
	"os"

	sdk "github.com/itglobalcom/vstack-cloud-panel-sdk"
	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

func backupLocation() string {
	if loc := os.Getenv("BACKUP_LOCATION"); loc != "" {
		return loc
	}
	return "kz"
}

func backupImageID() string {
	if image := os.Getenv("BACKUP_IMAGE_ID"); image != "" {
		return image
	}
	return "Debian-12-X64"
}

func backupRestoreEnabled() bool {
	return os.Getenv("BACKUP_RESTORE") == "1"
}

// backupDailyKeep returns the number of daily copies to keep within the
// partner's limits
func backupDailyKeep(limits *entities.BackupLimits) int {
	if limits != nil && limits.Daily != nil && limits.Daily.DefaultKeep > 0 {
		return limits.Daily.DefaultKeep
	}
	return 1
}

// backupScheduleHour returns the first hour of the partner's schedule window
func backupScheduleHour(limits *entities.BackupLimits) int {
	if limits != nil {
		return limits.ScheduleWindowFromHour
	}
	return 0
}

func runBackupExample(ctx context.Context, client *sdk.CloudClient) {
	fmt.Println("=== Server Backup Example ===")

	location := backupLocation()
	imageID := backupImageID()

	var serverID string
	var nearbyServerIDs []string
	backupEnabled := false

	cleanup := func() {
		fmt.Println("\n=== Cleanup ===")
		for _, id := range nearbyServerIDs {
			if err := client.DeleteServer(ctx, id); err != nil {
				log.Printf("⚠ Failed to delete restored server %s: %v", id, err)
			} else {
				fmt.Printf("✓ Restored server %s deleted\n", id)
			}
		}
		if serverID == "" {
			return
		}
		if backupEnabled {
			if err := client.DisableServerBackupAndWait(ctx, serverID); err != nil {
				log.Printf("⚠ Failed to disable backup of server %s: %v", serverID, err)
			} else {
				fmt.Printf("✓ Backup of server %s disabled\n", serverID)
			}
		}
		if err := client.DeleteServer(ctx, serverID); err != nil {
			log.Printf("⚠ Failed to delete server %s: %v", serverID, err)
		} else {
			fmt.Printf("✓ Server %s deleted\n", serverID)
		}
	}

	fail := func(format string, args ...any) {
		log.Printf("[FATAL] "+format, args...)
		cleanup()
		os.Exit(1)
	}

	// 1. Backup storages available to a new server in the location
	fmt.Printf("\n=== Step 1: Backup storages for an order in %s (image %s) ===\n", location, imageID)
	orderCatalog, err := client.GetBackupStorageList(ctx, location, imageID)
	if err != nil {
		fail("Failed to get backup storages for an order: %v", err)
	}
	fmt.Printf("Found %d storage(s)\n", len(orderCatalog.Storages))
	for _, storage := range orderCatalog.Storages {
		fmt.Printf("  - ID=%d, Name=%s, PricePerGB=%.4f, RestoreLocations=%v\n",
			storage.ID, storage.Name, storage.PricePerGB, storage.RestoreLocationIDs)
	}

	// 2. Create a server
	fmt.Println("\n=== Step 2: Creating server ===")
	server, err := client.CreateServerAndWait(ctx, &entities.CreateServerRequest{
		Name:       "test-backup",
		LocationID: location,
		ImageID:    imageID,
		CPU:        1,
		RamMB:      1024,
		Volumes: []entities.VolumeSpec{
			{
				Name:   "boot",
				SizeMB: 25600,
			},
		},
		Networks: []entities.NetworkSpec{
			{
				BandwidthMbps: 100,
			},
		},
		Tags: []string{"test", "backup-demo"},
	})
	if err != nil {
		fail("Failed to create server: %v", err)
	}
	serverID = server.ID
	fmt.Printf("✓ Server created: ID=%s, Name=%s\n", server.ID, server.Name)

	// 3. Backup storages available to the server
	fmt.Println("\n=== Step 3: Backup storages for the server ===")
	catalog, err := client.GetServerBackupStorages(ctx, serverID)
	if err != nil {
		fail("Failed to get backup storages of server %s: %v", serverID, err)
	}
	if len(catalog.Storages) == 0 {
		fail("No backup storages available to server %s", serverID)
	}
	storage := catalog.Storages[0]
	fmt.Printf("✓ Using storage ID=%d, Name=%s\n", storage.ID, storage.Name)

	// 4. Enable the backup service
	fmt.Println("\n=== Step 4: Enabling backup service ===")
	schedule := &entities.BackupSchedule{
		Hour:   backupScheduleHour(catalog.Limits),
		Minute: 0,
		Daily: &entities.BackupRule{
			Keep:            backupDailyKeep(catalog.Limits),
			BackupStorageID: storage.ID,
		},
	}
	backup, err := client.EnableServerBackupAndWait(ctx, serverID, schedule)
	if err != nil {
		fail("Failed to enable backup of server %s: %v", serverID, err)
	}
	backupEnabled = true
	fmt.Printf("✓ Backup enabled: Enabled=%v\n", backup.Enabled)

	// 5. Update the schedule
	fmt.Println("\n=== Step 5: Updating backup schedule ===")
	schedule.Minute = 30
	backup, err = client.UpdateServerBackupAndWait(ctx, serverID, schedule)
	if err != nil {
		fail("Failed to update backup of server %s: %v", serverID, err)
	}
	if backup.Schedule != nil {
		fmt.Printf("✓ Schedule updated: %02d:%02d\n", backup.Schedule.Hour, backup.Schedule.Minute)
	}

	// 6. Take a manual copy
	fmt.Println("\n=== Step 6: Taking manual copy ===")
	point, err := client.CreateServerBackupRestorePointAndWait(ctx, serverID, &entities.CreateBackupRestorePointRequest{
		Name: "manual-copy",
	})
	if sdk.IsBackupRestorePointFailed(err) {
		fail("The platform failed to take the copy: %v", err)
	}
	if err != nil {
		fail("Failed to take manual copy of server %s: %v", serverID, err)
	}
	fmt.Printf("✓ Copy taken: ID=%d, Name=%s, Size=%d MB, State=%s\n", point.ID, point.Name, point.SizeMB, point.State)

	// 7. List restore points
	fmt.Println("\n=== Step 7: Listing restore points ===")
	points, err := client.GetServerBackupRestorePoints(ctx, serverID)
	if err != nil {
		fail("Failed to list restore points of server %s: %v", serverID, err)
	}
	fmt.Printf("Total restore points: %d\n", len(points))
	for i, p := range points {
		fmt.Printf("  %d. ID=%d, Name=%s, State=%s, Manual=%v, RetainedBy=%v\n",
			i+1, p.ID, p.Name, p.State, p.IsManual, p.RetainedBy)
	}

	// 8. Restore over the server and nearby
	if backupRestoreEnabled() {
		fmt.Println("\n=== Step 8: Restoring server over itself ===")
		restored, err := client.RestoreServerBackupRestorePointAndWait(ctx, serverID, point.ID)
		if err != nil {
			fail("Failed to restore server %s from point %d: %v", serverID, point.ID, err)
		}
		fmt.Printf("✓ Server restored: State=%s\n", restored.State)

		nearbyLocation := location
		if len(storage.RestoreLocationIDs) > 0 {
			nearbyLocation = storage.RestoreLocationIDs[0]
		}
		fmt.Printf("\n=== Step 8a: Restoring copy nearby into %s ===\n", nearbyLocation)
		before, err := client.GetServerList(ctx)
		if err != nil {
			fail("Failed to get server list: %v", err)
		}
		known := make(map[string]bool, len(before))
		for _, s := range before {
			known[s.ID] = true
		}
		err = client.RestoreServerBackupRestorePointNearbyAndWait(ctx, serverID, point.ID, &entities.RestoreBackupRestorePointNearbyRequest{
			LocationID: nearbyLocation,
		})
		if err != nil {
			fail("Failed to restore point %d nearby: %v", point.ID, err)
		}
		after, err := client.GetServerList(ctx)
		if err != nil {
			fail("Failed to get server list: %v", err)
		}
		for _, s := range after {
			if !known[s.ID] && s.LocationID == nearbyLocation {
				nearbyServerIDs = append(nearbyServerIDs, s.ID)
				fmt.Printf("✓ Restored server: ID=%s, Name=%s\n", s.ID, s.Name)
			}
		}
	} else {
		fmt.Println("\n=== Step 8: Restore skipped (set BACKUP_RESTORE=1 to run it) ===")
	}

	// 9. Delete the manual copy
	fmt.Println("\n=== Step 9: Deleting manual copy ===")
	if err := client.DeleteServerBackupRestorePointAndWait(ctx, serverID, point.ID); err != nil {
		fail("Failed to delete restore point %d: %v", point.ID, err)
	}
	fmt.Printf("✓ Restore point %d deleted\n", point.ID)

	// 10. Disable the backup service
	fmt.Println("\n=== Step 10: Disabling backup service ===")
	if err := client.DisableServerBackupAndWait(ctx, serverID); err != nil {
		fail("Failed to disable backup of server %s: %v", serverID, err)
	}
	backupEnabled = false
	backup, err = client.GetServerBackup(ctx, serverID)
	if err != nil {
		fail("Failed to get backup of server %s: %v", serverID, err)
	}
	fmt.Printf("✓ Backup disabled: Enabled=%v\n", backup.Enabled)

	cleanup()
	fmt.Println("\nBackup demo completed successfully!")
}
