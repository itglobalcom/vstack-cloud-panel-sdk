package main

import (
	"context"
	"fmt"
	"log"

	sdk "github.com/itglobalcom/vstack-cloud-panel-sdk"
	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

func runServerNICExample(ctx context.Context, client *sdk.CloudClient) {
	fmt.Println("=== Server NIC Operations Example ===")

	// Create isolated network
	fmt.Println("\n=== Creating isolated network ===")
	networkReq := &entities.CreateNetworkRequest{
		Name:          "test-nic-network",
		LocationID:    "kz",
		Description:   "Test network for NIC example",
		NetworkPrefix: "10.100.0.0",
		Mask:          24,
	}

	network, err := client.CreateNetworkAndWait(ctx, networkReq)
	if err != nil {
		log.Fatalf("[FATAL] Failed to create network: %v", err)
	}
	fmt.Printf("Network created: ID=%s, Name=%s, Prefix=%s/%d\n",
		network.ID, network.Name, network.NetworkPrefix, network.Mask)

	networkID := network.ID

	// Create server WITHOUT network interfaces
	fmt.Println("\n=== Creating server WITHOUT network interfaces ===")
	createReq := &entities.CreateServerRequest{
		Name:       "test-server-nic",
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
		Networks: []entities.NetworkSpec{},
		Tags:     []string{"test", "nic-example"},
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

	// Check that there are no NICs
	fmt.Println("\n=== Checking initial NIC list ===")
	nics, err := client.GetServerNICs(ctx, serverID)
	if err != nil {
		log.Fatalf("Failed to get initial NICs: %v", err)
	}
	fmt.Printf("Initial NICs count: %d\n", len(nics))

	// Add 3 network interfaces (2 public + 1 isolated)
	fmt.Println("\n=== Adding 3 network interfaces (2 public + 1 isolated) ===")

	// NIC 1: Public network 100 Mbps
	fmt.Println("\n--- Creating NIC 1: Public 100 Mbps ---")
	nic1, err := client.CreateServerNICAndWait(ctx, serverID, &entities.CreateNICRequest{
		BandwidthMbps: 100,
	})
	if err != nil {
		log.Fatalf("Failed to create NIC 1: %v", err)
	}
	fmt.Printf("✓ NIC 1 created: ID=%d, Type=Public, IP=%s, Bandwidth=%d Mbps\n",
		nic1.ID, nic1.IPAddress, nic1.BandwidthMbps)

	// NIC 2: Public network 50 Mbps
	fmt.Println("\n--- Creating NIC 2: Public 50 Mbps ---")
	nic2, err := client.CreateServerNICAndWait(ctx, serverID, &entities.CreateNICRequest{
		BandwidthMbps: 50,
	})
	if err != nil {
		log.Fatalf("Failed to create NIC 2: %v", err)
	}
	fmt.Printf("✓ NIC 2 created: ID=%d, Type=Public, IP=%s, Bandwidth=%d Mbps\n",
		nic2.ID, nic2.IPAddress, nic2.BandwidthMbps)

	// NIC 3: Isolated network (auto IP)
	fmt.Println("\n--- Creating NIC 3: Isolated network (auto IP) ---")
	nic3, err := client.CreateServerNICAndWait(ctx, serverID, &entities.CreateNICRequest{
		NetworkID: networkID,
	})
	if err != nil {
		log.Fatalf("Failed to create NIC 3: %v", err)
	}
	fmt.Printf("✓ NIC 3 created: ID=%d, Type=Isolated, Network=%s, IP=%s/%d\n",
		nic3.ID, nic3.NetworkID, nic3.IPAddress, nic3.Mask)

	// Check all NICs
	fmt.Println("\n=== Checking all NICs after creation ===")
	nics, err = client.GetServerNICs(ctx, serverID)
	if err != nil {
		log.Fatalf("Failed to get NICs: %v", err)
	}
	fmt.Printf("Total NICs: %d\n", len(nics))

	var totalBandwidth int
	for i, nic := range nics {
		nicType := "Public"
		extra := fmt.Sprintf("Bandwidth=%d Mbps", nic.BandwidthMbps)
		if nic.NetworkID == networkID {
			nicType = "Isolated"
			extra = fmt.Sprintf("Network=%s", nic.NetworkID)
		} else {
			totalBandwidth += nic.BandwidthMbps
		}
		fmt.Printf("  %d. [%s] ID=%d, IP=%s/%d, MAC=%s, %s\n",
			i+1, nicType, nic.ID, nic.IPAddress, nic.Mask, nic.MAC, extra)
	}
	fmt.Printf("Total public bandwidth: %d Mbps\n", totalBandwidth)

	// Get details of isolated NIC
	fmt.Println("\n=== Getting detailed info for NIC 3 (Isolated) ===")
	nicDetails, err := client.GetServerNIC(ctx, serverID, nic3.ID)
	if err != nil {
		log.Fatalf("Failed to get NIC details: %v", err)
	}
	fmt.Printf("NIC %d Details:\n", nicDetails.ID)
	fmt.Printf("  Type: Isolated Network\n")
	fmt.Printf("  Server ID: %s\n", nicDetails.ServerID)
	fmt.Printf("  Network ID: %s\n", nicDetails.NetworkID)
	fmt.Printf("  MAC Address: %s\n", nicDetails.MAC)
	fmt.Printf("  IP Address: %s/%d\n", nicDetails.IPAddress, nicDetails.Mask)
	fmt.Printf("  Gateway: %s\n", nicDetails.Gateway)

	// Update bandwidth for 2 public NICs
	fmt.Println("\n=== Updating bandwidth for 2 public NICs ===")

	// Update NIC 1: 100 -> 150 Mbps
	fmt.Println("\n--- Updating NIC 1 (Public): 100 -> 150 Mbps ---")
	updatedNIC1, err := client.UpdateServerNICAndWait(ctx, serverID, nic1.ID, &entities.UpdateNICRequest{
		BandwidthMbps: 150,
	})
	if err != nil {
		log.Fatalf("Failed to update NIC 1: %v", err)
	}
	fmt.Printf("✓ NIC 1 updated: ID=%d, Bandwidth=%d -> %d Mbps\n",
		updatedNIC1.ID, nic1.BandwidthMbps, updatedNIC1.BandwidthMbps)

	// Update NIC 2: 50 -> 100 Mbps
	fmt.Println("\n--- Updating NIC 2 (Public): 50 -> 100 Mbps ---")
	updatedNIC2, err := client.UpdateServerNICAndWait(ctx, serverID, nic2.ID, &entities.UpdateNICRequest{
		BandwidthMbps: 100,
	})
	if err != nil {
		log.Fatalf("Failed to update NIC 2: %v", err)
	}
	fmt.Printf("✓ NIC 2 updated: ID=%d, Bandwidth=%d -> %d Mbps\n",
		updatedNIC2.ID, nic2.BandwidthMbps, updatedNIC2.BandwidthMbps)

	// Note: Isolated network NIC cannot have bandwidth updated
	fmt.Println("\n--- Note: NIC 3 (Isolated) doesn't support bandwidth updates ---")

	// Check updated NICs
	fmt.Println("\n=== Checking NICs after bandwidth updates ===")
	nics, err = client.GetServerNICs(ctx, serverID)
	if err != nil {
		log.Fatalf("Failed to get NICs: %v", err)
	}

	totalBandwidth = 0
	for i, nic := range nics {
		nicType := "Public"
		info := ""
		if nic.NetworkID == networkID {
			nicType = "Isolated"
			info = fmt.Sprintf("Network=%s", nic.NetworkID)
		} else {
			totalBandwidth += nic.BandwidthMbps
			info = fmt.Sprintf("Bandwidth=%d Mbps", nic.BandwidthMbps)
		}
		fmt.Printf("  %d. [%s] ID=%d, IP=%s, %s\n",
			i+1, nicType, nic.ID, nic.IPAddress, info)
	}
	fmt.Printf("Total public bandwidth after updates: %d Mbps\n", totalBandwidth)

	// Delete all 3 NICs
	fmt.Println("\n=== Deleting all 3 NICs ===")

	// Delete NIC 1 (Public)
	fmt.Println("\n--- Deleting NIC 1 (Public) ---")
	err = client.DeleteServerNICAndWait(ctx, serverID, nic1.ID)
	if err != nil {
		log.Fatalf("Failed to delete NIC 1: %v", err)
	}
	fmt.Printf("✓ NIC 1 (ID=%d) deletion initiated\n", nic1.ID)

	// Delete NIC 2 (Public)
	fmt.Println("\n--- Deleting NIC 2 (Public) ---")
	err = client.DeleteServerNICAndWait(ctx, serverID, nic2.ID)
	if err != nil {
		log.Fatalf("Failed to delete NIC 2: %v", err)
	}
	fmt.Printf("✓ NIC 2 (ID=%d) deletion initiated\n", nic2.ID)

	// Delete NIC 3 (Isolated)
	fmt.Println("\n--- Deleting NIC 3 (Isolated) ---")
	err = client.DeleteServerNICAndWait(ctx, serverID, nic3.ID)
	if err != nil {
		log.Fatalf("Failed to delete NIC 3: %v", err)
	}
	fmt.Printf("✓ NIC 3 (ID=%d) deletion initiated\n", nic3.ID)

	// Verify that there are no more NICs
	fmt.Println("\n=== Verifying NICs deletion ===")
	remainingNICs, err := client.GetServerNICs(ctx, serverID)
	if err != nil {
		log.Fatalf("Failed to get remaining NICs: %v", err)
	}
	fmt.Printf("Remaining NICs: %d\n", len(remainingNICs))

	if len(remainingNICs) > 0 {
		fmt.Println("Warning: Some NICs are still present:")
		for _, nic := range remainingNICs {
			nicType := "Public"
			if nic.NetworkID == networkID {
				nicType = "Isolated"
			}
			fmt.Printf("  - [%s] ID=%d, IP=%s\n", nicType, nic.ID, nic.IPAddress)
		}
	} else {
		fmt.Println("✓ All NICs successfully deleted")
	}

	// Delete server
	fmt.Println("\n=== Deleting server ===")
	err = client.DeleteServer(ctx, serverID)
	if err != nil {
		log.Fatalf("Failed to delete server: %v", err)
	}
	fmt.Printf("✓ Server %s deleted successfully\n", serverID)

	// Delete network
	fmt.Println("\n=== Deleting isolated network ===")
	err = client.DeleteNetwork(ctx, networkID)
	if err != nil {
		log.Fatalf("Failed to delete network: %v", err)
	}
	fmt.Printf("✓ Network %s deleted successfully\n", networkID)

	fmt.Println("\n=== Server NIC operations completed ===")
}
