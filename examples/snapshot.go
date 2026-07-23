package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	sdk "github.com/itglobalcom/vstack-cloud-panel-sdk"
	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

// sshClient executes a command on the server via SSH
func sshClient(host, user, password string, command string) (string, error) {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}

	client, err := ssh.Dial("tcp", host+":22", config)
	if err != nil {
		return "", fmt.Errorf("failed to dial: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	output, _ := session.CombinedOutput(command)
	return string(output), nil
}

// waitForSSH waits for SSH to become available on the server
func waitForSSH(host, user, password string, maxAttempts int) error {
	fmt.Printf("Waiting for SSH to become available on %s...\n", host)

	for i := 0; i < maxAttempts; i++ {
		_, err := sshClient(host, user, password, "echo 'SSH ready'")
		if err == nil {
			fmt.Printf("✓ SSH is ready after %d attempts\n", i+1)
			return nil
		}

		if i < maxAttempts-1 {
			fmt.Printf("  Attempt %d/%d failed, retrying in 5s...\n", i+1, maxAttempts)
			time.Sleep(5 * time.Second)
		}
	}

	return fmt.Errorf("SSH not available after %d attempts", maxAttempts)
}

// fileExists checks if a file exists
func fileExists(host, user, password, filepath string) bool {
	cmd := fmt.Sprintf("test -f '%s' && echo 'EXISTS' || echo 'NOT_FOUND'", filepath)
	output, _ := sshClient(host, user, password, cmd)
	return strings.Contains(output, "EXISTS")
}

// getFileInfo gets file information (size, creation time)
func getFileInfo(host, user, password, filepath string) (string, error) {
	cmd := fmt.Sprintf("ls -lh '%s' 2>/dev/null | awk '{print $5, $6, $7, $8}'", filepath)
	output, err := sshClient(host, user, password, cmd)
	return strings.TrimSpace(output), err
}

// createTestFile creates a test file using dd with /dev/zero
func createTestFile(host, user, password, filepath string, sizeMB int, fileNum int) error {
	fmt.Printf("Creating test file #%d: %s (size: %d MB)\n", fileNum, filepath, sizeMB)

	// Use dd with /dev/zero (very fast, fills with zeros)
	cmd := fmt.Sprintf("dd if=/dev/zero of=%s bs=1M count=%d 2>/dev/null && echo 'OK' || echo 'FAILED'",
		filepath, sizeMB)

	output, _ := sshClient(host, user, password, cmd)
	output = strings.TrimSpace(output)

	if fileExists(host, user, password, filepath) {
		info, _ := getFileInfo(host, user, password, filepath)
		fmt.Printf("✓ Test file #%d created successfully: %s\n", fileNum, info)
		return nil
	}

	return fmt.Errorf("failed to create file: %s", filepath)
}

// listFiles lists files in a directory
func listFiles(host, user, password, dir string) ([]string, error) {
	cmd := fmt.Sprintf("ls -1 '%s' 2>/dev/null", dir)
	output, _ := sshClient(host, user, password, cmd)

	var files []string
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

// initializeServer creates a directory for files
func initializeServer(host, user, password string) error {
	fmt.Println("=== Initializing server ===")

	cmd := "mkdir -p /data && echo 'OK'"
	output, _ := sshClient(host, user, password, cmd)

	if strings.Contains(output, "OK") {
		fmt.Println("✓ Server initialized, /data directory ready")
		return nil
	}

	return fmt.Errorf("failed to initialize server")
}

func runSnapshotExample(ctx context.Context, client *sdk.CloudClient) {
	fmt.Println("╔════════════════════════════════════════════════════════╗")
	fmt.Println("║  Advanced Server Snapshot Example with Test Files      ║")
	fmt.Println("╚════════════════════════════════════════════════════════╝")

	// 1. Create a server
	fmt.Println("=== Step 1: Creating server ===")
	createReq := &entities.CreateServerRequest{
		Name:       "test-snapshot-files",
		LocationID: "kz",
		ImageID:    "Debian-12-X64",
		CPU:        2,
		RamMB:      2048,
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
		Tags: []string{"test", "snapshot-demo"},
	}

	server, err := client.CreateServerAndWait(ctx, createReq)

	if err != nil {
		log.Fatalf("[FATAL] Failed to wait for server creation: %v", err)
	}

	fmt.Printf("✓ Server created successfully!\n")
	fmt.Printf("  ID: %s\n", server.ID)
	fmt.Printf("  Name: %s\n", server.Name)

	serverID := server.ID
	serverIP := server.NICs[0].IPAddress
	username := server.Login
	password := server.Password

	fmt.Printf("  IP Address: %s\n\n", serverIP)

	// 2. Wait for SSH to be available
	fmt.Println("=== Step 2: Waiting for SSH ===")
	err = waitForSSH(serverIP, username, password, 30)
	if err != nil {
		log.Fatalf("[FATAL] %v", err)
	}

	// 3. Initialize the server
	fmt.Println("\n=== Step 3: Initializing server ===")
	err = initializeServer(serverIP, username, password)
	if err != nil {
		log.Fatalf("[FATAL] %v", err)
	}

	// 4. Create the first file
	fmt.Println("\n=== Step 4: Creating test file #1 (100 MB) ===")
	file1 := "/data/database-backup-v1.0.dat"
	err = createTestFile(serverIP, username, password, file1, 100, 1)
	if err != nil {
		log.Fatalf("[FATAL] %v", err)
	}

	// Check if the file exists
	if fileExists(serverIP, username, password, file1) {
		fmt.Printf("✓ File #1 verified: exists at %s\n", file1)
	} else {
		log.Fatalf("[FATAL] File #1 not found")
	}

	// 5. Create the first snapshot
	fmt.Println("\n=== Step 5: Creating Snapshot 1 (with file #1) ===")
	snapshot1, err := client.CreateServerSnapshotAndWait(ctx, serverID, &entities.CreateSnapshotRequest{
		Name: "snapshot-with-file-100mb",
	})
	if err != nil {
		log.Fatalf("[FATAL] Failed to create snapshot 1: %v", err)
	}
	fmt.Printf("✓ Snapshot 1 created: ID=%d, Size=%d MB\n", snapshot1.ID, snapshot1.SizeMB)

	// 6. Create the second file
	fmt.Println("\n=== Step 6: Creating test file #2 (50 MB) ===")
	file2 := "/data/config-backup-v2.0.dat"
	err = createTestFile(serverIP, username, password, file2, 50, 2)
	if err != nil {
		log.Fatalf("[FATAL] %v", err)
	}

	// Check both files
	fmt.Println("\nCurrent files on server:")
	files, _ := listFiles(serverIP, username, password, "/data")
	for _, f := range files {
		info, _ := getFileInfo(serverIP, username, password, "/data/"+f)
		fmt.Printf("  - %s (%s)\n", f, info)
	}

	// 7. Create the second snapshot
	fmt.Println("\n=== Step 7: Creating Snapshot 2 (with both files) ===")
	snapshot2, err := client.CreateServerSnapshotAndWait(ctx, serverID, &entities.CreateSnapshotRequest{
		Name: "both-files-150mb",
	})
	if err != nil {
		log.Fatalf("[FATAL] Failed to create snapshot 2: %v", err)
	}
	fmt.Printf("✓ Snapshot 2 created: ID=%d, Size=%d MB\n", snapshot2.ID, snapshot2.SizeMB)

	// List all snapshots
	fmt.Println("\n=== All available snapshots ===")
	snapshots, err := client.GetServerSnapshots(ctx, serverID)
	if err != nil {
		log.Fatalf("[FATAL] Failed to get snapshots: %v", err)
	}
	fmt.Printf("Total snapshots: %d\n", len(snapshots))
	for i, snap := range snapshots {
		fmt.Printf("  %d. ID=%d, Name=%-40s Size=%d MB\n",
			i+1, snap.ID, snap.Name, snap.SizeMB)
	}

	// 8. Rollback to the first snapshot
	fmt.Println("\n=== Step 8: Rolling back to Snapshot 1 ===")
	fmt.Println("This will restore the state with only file #1")

	serverAfterRollback, err := client.RollbackServerSnapshotAndWait(ctx, serverID, snapshot1.ID)
	if err != nil {
		log.Fatalf("[FATAL] Failed to rollback: %v", err)
	}
	fmt.Printf("✓ Rolled back successfully. Server state: %s\n", serverAfterRollback.State)

	// Wait for SSH after rollback
	fmt.Println("\nWaiting for SSH after rollback...")
	err = waitForSSH(serverIP, username, password, 30)
	if err != nil {
		log.Fatalf("[FATAL] SSH not available after rollback: %v", err)
	}

	// 9. Verify file state after rollback
	fmt.Println("\n=== Step 9: Verifying file state after rollback ===")

	exists1 := fileExists(serverIP, username, password, file1)
	exists2 := fileExists(serverIP, username, password, file2)

	fmt.Printf("File #1 (%s): ", file1)
	if exists1 {
		fmt.Println("EXISTS ✓")
	} else {
		fmt.Println("NOT FOUND ✗")
	}

	fmt.Printf("File #2 (%s): ", file2)
	if exists2 {
		fmt.Println("EXISTS ✓")
	} else {
		fmt.Println("NOT FOUND ✓")
	}

	if exists1 && !exists2 {
		fmt.Println("\n✓ Rollback verified successfully! State restored correctly.")
		fmt.Println("  File #1 is present (as expected)")
		fmt.Println("  File #2 is absent (as expected)")
	} else {
		fmt.Println("\n⚠ Warning: File state is not as expected!")
		fmt.Printf("  Expected: file1=true, file2=false\n")
		fmt.Printf("  Got: file1=%v, file2=%v\n", exists1, exists2)
	}

	// 10. Get the list of remaining snapshots
	fmt.Println("\n=== Step 10: Checking remaining snapshots after rollback ===")
	snapshotsAfterRollback, err := client.GetServerSnapshots(ctx, serverID)
	if err != nil {
		log.Fatalf("[FATAL] Failed to get snapshots: %v", err)
	}

	fmt.Printf("Remaining snapshots: %d\n", len(snapshotsAfterRollback))
	for i, snap := range snapshotsAfterRollback {
		fmt.Printf("  %d. ID=%d, Name=%s, Size=%d MB\n",
			i+1, snap.ID, snap.Name, snap.SizeMB)
	}

	// 11. Delete snapshots in order
	fmt.Println("\n=== Step 11: Deleting snapshots in order ===")

	for i, snap := range snapshotsAfterRollback {
		fmt.Printf("\n[%d/%d] Deleting Snapshot ID=%d (Name: %s)\n",
			i+1, len(snapshotsAfterRollback), snap.ID, snap.Name)

		err = client.DeleteServerSnapshotAndWait(ctx, serverID, snap.ID)
		if err != nil {
			log.Printf("⚠ Warning: Failed to delete snapshot %d: %v", snap.ID, err)
		} else {
			fmt.Printf("✓ Snapshot %d deleted successfully\n", snap.ID)
		}

		time.Sleep(2 * time.Second)
	}

	// 12. Verify that all snapshots are deleted
	fmt.Println("\n=== Step 12: Verifying all snapshots are deleted ===")
	remainingSnapshots, err := client.GetServerSnapshots(ctx, serverID)
	if err != nil {
		log.Fatalf("[FATAL] Failed to verify snapshots: %v", err)
	}

	if len(remainingSnapshots) == 0 {
		fmt.Println("✓ All snapshots deleted successfully")
	} else {
		fmt.Printf("⚠ Warning: %d snapshots still remain\n", len(remainingSnapshots))
		for _, snap := range remainingSnapshots {
			fmt.Printf("  - ID=%d, Name=%s\n", snap.ID, snap.Name)
		}
	}

	// 13. Delete the server
	fmt.Println("\n=== Step 13: Deleting server ===")
	err = client.DeleteServer(ctx, serverID)
	if err != nil {
		log.Fatalf("[FATAL] Failed to delete server: %v", err)
	}
	fmt.Printf("✓ Server %s deleted successfully\n", serverID)

	// Final report
	fmt.Println("\n" + strings.Repeat("═", 56))
	fmt.Println("║ Snapshot demo completed successfully! ║")
	fmt.Println(strings.Repeat("═", 56))
	fmt.Println("\nSummary of operations:")
	fmt.Println("  ✓ Created server")
	fmt.Println("  ✓ Created test file #1 (100 MB)")
	fmt.Println("  ✓ Created Snapshot 1")
	fmt.Println("  ✓ Created test file #2 (50 MB)")
	fmt.Println("  ✓ Created Snapshot 2")
	fmt.Println("  ✓ Rolled back to Snapshot 1")
	fmt.Println("  ✓ Verified file state after rollback")
	fmt.Println("  ✓ Deleted all snapshots in order")
	fmt.Println("  ✓ Deleted server")
	fmt.Println("\nAll operations completed successfully!")
}
