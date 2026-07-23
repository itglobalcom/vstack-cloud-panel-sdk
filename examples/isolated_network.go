package main

import (
	"context"
	"fmt"
	"log"

	sdk "github.com/itglobalcom/vstack-cloud-panel-sdk"
	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

func runNetworkExample(ctx context.Context, client *sdk.CloudClient) {
	fmt.Println("=== Network Operations Example ===")

	// Get a list of all networks
	fmt.Println("=== Getting network list ===")
	networks, err := client.GetNetworkList(ctx)
	if err != nil {
		log.Fatalf("Failed to get network list: %v", err)
	}
	fmt.Printf("Found %d networks\n\n", len(networks))

	// Create a new network
	fmt.Println("=== Creating new network ===")
	createReq := &entities.CreateNetworkRequest{
		Name:          "test-network",
		LocationID:    "kz",
		Description:   "Test network for demo",
		NetworkPrefix: "10.100.0.0",
		Mask:          24,
	}

	network, err := client.CreateNetworkAndWait(ctx, createReq)
	if err != nil {
		log.Fatalf("[FATAL] %v", err)
	}
	fmt.Printf("Network created: ID=%s, Name=%s, State=%s\n\n", network.ID, network.Name, network.State)

	networkID := network.ID

	// Get network information
	fmt.Println("=== Getting network details ===")
	network, err = client.GetNetwork(ctx, networkID)
	if err != nil {
		log.Fatalf("Failed to get network: %v", err)
	}
	fmt.Printf("Network: %s (%s)\n", network.Name, network.ID)
	fmt.Printf("Location: %s\n", network.LocationID)
	fmt.Printf("Network: %s/%d\n\n", network.NetworkPrefix, network.Mask)

	// Add a tag
	fmt.Println("=== Adding tag to network ===")
	tagReq := &sdk.AddNetworkTagRequest{
		Tag: "production",
	}
	err = client.AddNetworkTag(ctx, networkID, tagReq)
	if err != nil {
		log.Fatalf("Failed to add tag: %v", err)
	}
	fmt.Println("Tag 'production' added successfully")

	// Update the network
	fmt.Println("=== Updating network ===")
	updateReq := &entities.UpdateNetworkRequest{
		Name:        "test-network-updated",
		Description: "Updated description",
	}

	network, err = client.UpdateNetwork(ctx, networkID, updateReq)
	if err != nil {
		log.Fatalf("Failed to update network: %v", err)
	}
	fmt.Printf("Network updated: Name=%s\n\n", network.Name)

	// Delete the network
	fmt.Println("=== Deleting network ===")
	err = client.DeleteNetwork(ctx, networkID)
	if err != nil {
		log.Fatalf("Failed to delete network: %v", err)
	}
	fmt.Printf("Network %s deleted successfully\n", networkID)
}
