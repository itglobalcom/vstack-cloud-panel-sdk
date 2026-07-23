package main

import (
	"context"
	"fmt"
	"log"

	sdk "github.com/itglobalcom/vstack-cloud-panel-sdk"
	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

func runAffinityGroupExample(ctx context.Context, client *sdk.CloudClient) {
	fmt.Println("=== Affinity Groups Example ===")

	// 1. Get a list of all affinity groups
	fmt.Println("Getting affinity groups list...")
	groups, err := client.GetAffinityGroupList(ctx)
	if err != nil {
		log.Fatalf("Failed to get affinity groups: %v", err)
	}
	fmt.Printf("Found %d groups\n\n", len(groups))

	// 2. Create an affinity group
	fmt.Println("Creating affinity group...")
	affinityGroup, err := client.CreateAffinityGroup(ctx, &entities.CreateAffinityGroupRequest{
		Name:       "demo-affinity-group",
		LocationID: "kz",
		Affinity:   true,
	})
	if err != nil {
		log.Fatalf("Failed to create affinity group: %v", err)
	}
	fmt.Printf("Created: %s (ID: %s)\n\n", affinityGroup.Name, affinityGroup.ID)

	// 3. Create an anti-affinity group
	fmt.Println("Creating anti-affinity group...")
	antiAffinityGroup, err := client.CreateAffinityGroup(ctx, &entities.CreateAffinityGroupRequest{
		Name:       "demo-anti-affinity-group",
		LocationID: "kz",
		Affinity:   false,
	})
	if err != nil {
		log.Fatalf("Failed to create anti-affinity group: %v", err)
	}
	fmt.Printf("Created: %s (ID: %s)\n\n", antiAffinityGroup.Name, antiAffinityGroup.ID)

	// 4. Get details of a specific group
	fmt.Println("Getting group details...")
	group, err := client.GetAffinityGroup(ctx, affinityGroup.ID)
	if err != nil {
		log.Fatalf("Failed to get group: %v", err)
	}
	fmt.Printf("Group: %s\n", group.Name)
	fmt.Printf("Type: %s\n", getGroupType(group.Affinity))
	fmt.Printf("Location: %s\n", group.LocationID)
	fmt.Printf("Servers: %d\n\n", len(group.ServerIDs))

	// 5. Get the updated list
	fmt.Println("Getting updated groups list...")
	updatedGroups, err := client.GetAffinityGroupList(ctx)
	if err != nil {
		log.Fatalf("Failed to get groups: %v", err)
	}
	fmt.Printf("Total groups: %d\n", len(updatedGroups))
	for _, g := range updatedGroups {
		fmt.Printf("  - %s (%s, Location: %s)\n", g.Name, getGroupType(g.Affinity), g.LocationID)
	}
	fmt.Println()

	// 6. Delete the created groups
	fmt.Println("Deleting affinity group...")
	if err := client.DeleteAffinityGroup(ctx, affinityGroup.ID); err != nil {
		log.Fatalf("Failed to delete group: %v", err)
	}
	fmt.Printf("Deleted: %s\n\n", affinityGroup.ID)

	fmt.Println("Deleting anti-affinity group...")
	if err := client.DeleteAffinityGroup(ctx, antiAffinityGroup.ID); err != nil {
		log.Fatalf("Failed to delete group: %v", err)
	}
	fmt.Printf("Deleted: %s\n\n", antiAffinityGroup.ID)

	// 7. Verify that the groups are deleted
	fmt.Println("Verifying deletion...")
	finalGroups, err := client.GetAffinityGroupList(ctx)
	if err != nil {
		log.Fatalf("Failed to get groups: %v", err)
	}
	fmt.Printf("Final count: %d groups\n\n", len(finalGroups))

	fmt.Println("Example completed successfully!")
}

func getGroupType(affinity bool) string {
	if affinity {
		return "Affinity"
	}
	return "Anti-Affinity"
}
