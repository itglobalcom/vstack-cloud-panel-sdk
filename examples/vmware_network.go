package main

import (
	"context"
	"fmt"
	"log"

	sdk "github.com/itglobalcom/vstack-cloud-panel-sdk"
	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

// C-14: end-to-end VMware network example. It creates a routed and an isolated
// network via the create-and-wait helpers, reads one back and lists them.
// Routed and isolated creation both run as background tasks, so the AndWait
// variants poll the task and return the ready network.
func runVmwareNetworkExample(ctx context.Context, client *sdk.CloudClient) {
	fmt.Println("=== VMware Network Operations Example ===")

	// Get a list of all VMware networks. A nil location filter returns every
	// network in the project.
	fmt.Println("\n=== Getting VMware network list ===")
	networks, err := client.GetVmwareNetworkList(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to get VMware network list: %v", err)
	}
	fmt.Printf("Found %d VMware networks\n", len(networks))

	// Create a routed network (it carries an edge gateway). Mask and DHCP are
	// optional pointer fields, so they are passed by address.
	fmt.Println("\n=== Creating routed VMware network ===")
	routedMask := 24
	routedDhcp := true
	routedBandwidth := 100
	routedReq := &entities.VmwareCreateRoutedNetworkRequest{
		LocationID:    1,
		Name:          "test-vmware-routed",
		Address:       "10.100.0.0",
		Mask:          &routedMask,
		EnableDhcp:    &routedDhcp,
		BandwidthMbps: &routedBandwidth,
	}

	routed, err := client.CreateVmwareRoutedNetworkAndWait(ctx, routedReq)
	if err != nil {
		log.Fatalf("[FATAL] Failed to create routed VMware network: %v", err)
	}
	fmt.Printf("Routed network created: ID=%d, Name=%s, Type=%s, State=%s\n",
		routed.ID, routed.Name, routed.Type, routed.State)
	// Type carries one of the VmwareNetworkType* constants.
	fmt.Printf("  Routed client: %v\n", routed.Type == entities.VmwareNetworkTypeRoutedClient)

	// Create an isolated (private) network.
	fmt.Println("\n=== Creating isolated VMware network ===")
	isolatedMask := 24
	isolatedDhcp := false
	isolatedReq := &entities.VmwareCreateIsolatedNetworkRequest{
		LocationID: 1,
		Name:       "test-vmware-isolated",
		Address:    "10.200.0.0",
		Mask:       &isolatedMask,
		EnableDhcp: &isolatedDhcp,
	}

	isolated, err := client.CreateVmwareIsolatedNetworkAndWait(ctx, isolatedReq)
	if err != nil {
		log.Fatalf("[FATAL] Failed to create isolated VMware network: %v", err)
	}
	fmt.Printf("Isolated network created: ID=%d, Name=%s, Type=%s, State=%s\n",
		isolated.ID, isolated.Name, isolated.Type, isolated.State)
	fmt.Printf("  Private client: %v\n", isolated.Type == entities.VmwareNetworkTypePrivateClient)

	// Get a network back by id.
	fmt.Println("\n=== Getting VMware network details ===")
	network, err := client.GetVmwareNetwork(ctx, routed.ID)
	if err != nil {
		log.Fatalf("Failed to get VMware network: %v", err)
	}
	fmt.Printf("VMware network: %s (%d)\n", network.Name, network.ID)
	fmt.Printf("  Location: %d\n", network.LocationID)
	fmt.Printf("  Type: %s, State: %s\n", network.Type, network.State)
	if network.Address != nil && network.Mask != nil {
		fmt.Printf("  Subnet: %s/%d\n", *network.Address, *network.Mask)
	}
	fmt.Printf("  Attached NICs: %d\n", network.NicsCount)

	fmt.Println("\n=== VMware network operations completed ===")
}
