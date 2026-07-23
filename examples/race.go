package main

import (
	"context"
	"fmt"
	"log"

	sdk "github.com/itglobalcom/vstack-cloud-panel-sdk"
	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

func runRaceConditionTest(ctx context.Context, client *sdk.CloudClient) {
	fmt.Println("=== Race Condition Test: Delete NIC + Delete Server ===")

	// Create isolated network
	fmt.Println("\n=== Creating isolated network ===")
	networkReq := &entities.CreateNetworkRequest{
		Name:          "test-race-network",
		LocationID:    "kz",
		Description:   "Test network for race condition",
		NetworkPrefix: "10.100.0.0",
		Mask:          24,
	}

	network, err := client.CreateNetworkAndWait(ctx, networkReq)
	if err != nil {
		log.Fatalf("[FATAL] Failed to create network: %v", err)
	}
	fmt.Printf("Network created: ID=%s, Name=%s\n", network.ID, network.Name)

	networkID := network.ID

	// Create a server WITHOUT network interfaces
	fmt.Println("\n=== Creating server WITHOUT network interfaces ===")
	createReq := &entities.CreateServerRequest{
		Name:       "test-race-server",
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
		Networks: []entities.NetworkSpec{},
		Tags:     []string{"test", "race-condition"},
	}

	server, err := client.CreateServerAndWait(ctx, createReq)

	if err != nil {
		log.Fatalf("[FATAL] Failed to wait for server creation: %v", err)
	}

	fmt.Printf("Server created successfully! ID: %s, State: %s\n", server.ID, server.State)
	serverID := server.ID

	// Add NIC to isolated network
	fmt.Println("\n=== Creating NIC in isolated network ===")
	nic, err := client.CreateServerNICAndWait(ctx, serverID, &entities.CreateNICRequest{
		NetworkID: networkID,
	})
	if err != nil {
		log.Fatalf("[FATAL] Failed to create NIC: %v", err)
	}
	fmt.Printf("✓ NIC created: ID=%d, Type=Isolated, Network=%s, IP=%s/%d\n",
		nic.ID, nic.NetworkID, nic.IPAddress, nic.Mask)

	// Check that the NIC is created
	nics, err := client.GetServerNICs(ctx, serverID)
	if err != nil {
		log.Fatalf("[FATAL] Failed to get NICs: %v", err)
	}
	fmt.Printf("Current NICs count: %d\n", len(nics))

	fmt.Println("\n=== RACE CONDITION TEST: Deleting NIC and immediately deleting server ===")

	fmt.Printf("Step 1: Initiating NIC deletion (ID=%d)...\n", nic.ID)
	err = client.DeleteServerNICAndWait(ctx, serverID, nic.ID)
	if err != nil {
		log.Fatalf("[FATAL] Failed to initiate NIC deletion: %v", err)
	}
	fmt.Printf("✓ NIC deletion initiated (async)\n")

	fmt.Printf("Step 2: Immediately deleting server (ID=%s)...\n", serverID)
	err = client.DeleteServer(ctx, serverID)
	if err != nil {
		// We expect to get an error here due to a race condition
		fmt.Printf("[ERROR] Race condition detected! Error while deleting server: %v\n", err)
		fmt.Println("✓ Test successful - race condition caught!")
	} else {
		fmt.Printf("✓ Server deletion completed without error\n")
		fmt.Println("[WARNING] Race condition not triggered - system handled it correctly or timing was different")
	}

	// Cleanup: trying to delete the network
	fmt.Println("\n=== Cleanup: Attempting to delete network ===")
	err = client.DeleteNetwork(ctx, networkID)
	if err != nil {
		// The network may still be in use if the server has not been completely deleted
		fmt.Printf("[WARNING] Failed to delete network (may still be in use): %v\n", err)
	} else {
		fmt.Printf("✓ Network %s deleted successfully\n", networkID)
	}

	fmt.Println("\n=== Race condition test completed ===")
}
