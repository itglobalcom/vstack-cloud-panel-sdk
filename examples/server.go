package main

import (
	"context"
	"fmt"
	"log"

	sdk "github.com/itglobalcom/vstack-cloud-panel-sdk"
	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

func runServerExample(ctx context.Context, client *sdk.CloudClient) {
	fmt.Println("=== Server Operations Example ===")

	// Get a list of all servers
	fmt.Println("\n=== Getting server list ===")
	servers, err := client.GetServerList(ctx)
	if err != nil {
		log.Fatalf("Failed to get server list: %v", err)
	}
	fmt.Printf("Found %d servers\n", len(servers))

	// Create a temporary SSH key to attach to the server
	fmt.Println("\n=== Creating temporary SSH key ===")
	sshKey, err := client.CreateSSHKey(ctx, &entities.CreateSSHKeyRequest{
		Name:      "test-server-key",
		PublicKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDKGui+SXS8SYLDm6sGvH1c6JFhFi5M9HdDBCHgu3k/A user@example.com",
	})
	if err != nil {
		log.Fatalf("Failed to create SSH key: %v", err)
	}
	fmt.Printf("SSH key created: ID=%d\n", sshKey.ID)

	// Create a new server
	fmt.Println("\n=== Creating new server ===")
	createReq := &entities.CreateServerRequest{
		Name:       "test-server",
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
		Tags:             []string{"test", "example"},
		SSHKeyIDs:        []int{sshKey.ID},
		ServerInitScript: "mkdir -p /data; touch /data/test; echo test > /data/test",
	}

	server, err := client.CreateServerAndWait(ctx, createReq)
	if err != nil {
		log.Fatalf("[FATAL] Failed to create server: %v", err)
	}

	fmt.Printf("Server created successfully!\n")
	fmt.Printf("  ID: %s\n", server.ID)
	fmt.Printf("  Name: %s\n", server.Name)
	fmt.Printf("  State: %s\n", server.State)
	fmt.Printf("  Login: %s\n", server.Login)
	fmt.Printf("  Password: %s\n", server.Password)

	serverID := server.ID

	// Get server information
	fmt.Println("\n=== Getting server details ===")
	server, err = client.GetServer(ctx, serverID)
	if err != nil {
		log.Fatalf("Failed to get server: %v", err)
	}
	fmt.Printf("Server details:\n")
	fmt.Printf("  Name: %s (%s)\n", server.Name, server.ID)
	fmt.Printf("  Location: %s\n", server.LocationID)
	fmt.Printf("  Resources: %d CPU, %d MB RAM\n", server.CPU, server.RamMB)
	fmt.Printf("  State: %s, Power: %v\n", server.State, server.IsPoweredOn())

	// Full resource update (PUT)
	fmt.Println("\n=== Updating server resources (PUT - full update) ===")
	updateReq := &entities.UpdateServerRequest{
		CPU:   4,
		RamMB: 4096,
	}
	server, err = client.UpdateServerAndWait(ctx, serverID, updateReq)
	if err != nil {
		log.Fatalf("Failed to update server: %v", err)
	}
	fmt.Printf("Update completed. CPU: %d, RAM: %d MB\n", server.CPU, server.RamMB)

	// Partially update server resources (PATCH)
	fmt.Println("\n=== Updating server resources (PATCH - CPU only) ===")
	newCPU := 3
	patchReq := &entities.PatchServerRequest{
		CPU: &newCPU,
	}
	server, err = client.PatchServerAndWait(ctx, serverID, patchReq)
	if err != nil {
		log.Fatalf("Failed to patch server: %v", err)
	}
	fmt.Printf("Patch completed. CPU: %d\n", server.CPU)

	// Power management operations
	fmt.Println("\n=== Power management operations ===")

	// 1. Graceful shutdown
	fmt.Println("\n--- Powering off server (graceful) ---")
	server, err = client.PowerOffServerAndWait(ctx, serverID)
	if err != nil {
		log.Fatalf("Failed to power off server: %v", err)
	}
	fmt.Printf("Server powered off. Power status: %v\n", server.IsPoweredOn())

	// 2. Powering on again
	fmt.Println("\n--- Powering on server ---")
	server, err = client.PowerOnServerAndWait(ctx, serverID)
	if err != nil {
		log.Fatalf("Failed to power on server: %v", err)
	}
	fmt.Printf("Server powered on. Power status: %v\n", server.IsPoweredOn())

	// 3. Soft reboot
	fmt.Println("\n--- Rebooting server (soft) ---")
	server, err = client.RebootServerAndWait(ctx, serverID)
	if err != nil {
		log.Fatalf("Failed to reboot server: %v", err)
	}
	fmt.Printf("Server rebooted. State: %s\n", server.State)

	// 4. Hard reset
	fmt.Println("\n--- Resetting server (hard) ---")
	server, err = client.ResetServerAndWait(ctx, serverID)
	if err != nil {
		log.Fatalf("Failed to reset server: %v", err)
	}
	fmt.Printf("Server reset. State: %s\n", server.State)

	// 5. Shutting down before deletion
	fmt.Println("\n--- Shutting down server before deletion ---")
	server, err = client.ShutdownServerAndWait(ctx, serverID)
	if err != nil {
		log.Fatalf("Failed to shutdown server: %v", err)
	}
	fmt.Printf("Server shutdown. Power status: %v\n", server.IsPoweredOn())

	// Delete the server
	fmt.Println("\n=== Deleting server ===")
	err = client.DeleteServer(ctx, serverID)
	if err != nil {
		log.Fatalf("Failed to delete server: %v", err)
	}
	fmt.Printf("Server %s deleted successfully\n", serverID)

	// Delete the temporary SSH key
	fmt.Println("\n=== Deleting temporary SSH key ===")
	if err := client.DeleteSSHKey(ctx, sshKey.ID); err != nil {
		log.Fatalf("Failed to delete SSH key: %v", err)
	}
	fmt.Printf("SSH key %d deleted successfully\n", sshKey.ID)

	fmt.Println("\n=== Server operations completed ===")
}
