package main

import (
	"context"
	"fmt"
	"log"

	sdk "github.com/itglobalcom/vstack-cloud-panel-sdk"
	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

// C-14: end-to-end VMware server example (create-and-wait, read, list).
// It mirrors the style of examples/server.go but uses the VMware resource,
// whose ids are int-typed and whose create operation runs as a background task.
func runVmwareServerExample(ctx context.Context, client *sdk.CloudClient) {
	fmt.Println("=== VMware Server Operations Example ===")

	// Get a list of all VMware servers. A nil location filter returns every
	// server in the project.
	fmt.Println("\n=== Getting VMware server list ===")
	servers, err := client.GetVmwareServerList(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to get VMware server list: %v", err)
	}
	fmt.Printf("Found %d VMware servers\n", len(servers))

	// Create a new VMware server and wait for the background task to finish.
	// CreateVmwareServerAndWait polls the create task and returns the ready
	// server, so there is no need to handle VmwareServerOrder/task_id manually.
	fmt.Println("\n=== Creating new VMware server ===")
	createReq := &entities.VmwareCreateServerRequest{
		LocationID:       1,
		Name:             "test-vmware-server",
		ImageID:          1,
		CPUCount:         2,
		RamMB:            2048,
		SystemDiskSizeMB: 25600, // 25 GB
	}

	server, err := client.CreateVmwareServerAndWait(ctx, createReq)
	if err != nil {
		log.Fatalf("[FATAL] Failed to create VMware server: %v", err)
	}
	fmt.Printf("VMware server created successfully!\n")
	fmt.Printf("  ID: %d\n", server.ID)
	fmt.Printf("  Name: %s\n", server.Name)
	fmt.Printf("  State: %s\n", server.State)

	serverID := server.ID

	// Get server information by id.
	fmt.Println("\n=== Getting VMware server details ===")
	server, err = client.GetVmwareServer(ctx, serverID)
	if err != nil {
		log.Fatalf("Failed to get VMware server: %v", err)
	}
	fmt.Printf("VMware server details:\n")
	fmt.Printf("  Name: %s (%d)\n", server.Name, server.ID)
	fmt.Printf("  Location: %d\n", server.LocationID)
	fmt.Printf("  Resources: %d CPU, %d MB RAM, %d MB system disk\n",
		server.CPU, server.RamMB, server.SystemDiskMB)
	// The State field carries one of the VmwareServerState* constants.
	fmt.Printf("  State: %s, Powered on: %v\n", server.State, server.IsPowerOn)
	fmt.Printf("  Active: %v\n", server.State == entities.VmwareServerStateActive)

	// List the network interfaces attached to the server.
	fmt.Printf("  Network interfaces: %d\n", len(server.NICs))
	for _, nic := range server.NICs {
		ip := "<none>"
		if nic.IP != nil {
			ip = *nic.IP
		}
		fmt.Printf("    - NIC #%d (network %d): ip=%s, primary=%v\n",
			nic.Number, nic.NetworkID, ip, nic.IsPrimary)
	}

	fmt.Println("\n=== VMware server operations completed ===")
}
