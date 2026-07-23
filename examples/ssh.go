package main

import (
	"context"
	"fmt"
	"log"

	sdk "github.com/itglobalcom/vstack-cloud-panel-sdk"
	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

func runSSHExample(ctx context.Context, client *sdk.CloudClient) {
	fmt.Println("=== SSH Keys Operations Example ===")

	// Get a list of all SSH keys
	fmt.Println("=== Getting SSH keys list ===")
	sshKeys, err := client.GetSSHKeyList(ctx)
	if err != nil {
		log.Fatalf("Failed to get SSH keys list: %v", err)
	}
	fmt.Printf("Found %d SSH keys\n\n", len(sshKeys))

	// Create a new SSH key
	fmt.Println("=== Creating new SSH key ===")
	createReq := &entities.CreateSSHKeyRequest{
		Name:      "demo-key",
		PublicKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDKGui+SXS8SYLDm6sGvH1c6JFhFi5M9HdDBCHgu3k/A user@example.com",
	}

	sshKey, err := client.CreateSSHKey(ctx, createReq)
	if err != nil {
		log.Fatalf("Failed to create SSH key: %v", err)
	}
	fmt.Printf("SSH key created: ID=%d, Name=%s\n\n", sshKey.ID, sshKey.Name)

	keyID := sshKey.ID

	// Get SSH key information
	fmt.Println("=== Getting SSH key details ===")
	sshKey, err = client.GetSSHKey(ctx, keyID)
	if err != nil {
		log.Fatalf("Failed to get SSH key: %v", err)
	}
	fmt.Printf("SSH Key: %s (ID: %d)\n", sshKey.Name, sshKey.ID)
	fmt.Printf("Public Key: %s...\n\n", sshKey.PublicKey[:50])

	// Get updated list
	fmt.Println("=== Getting updated SSH keys list ===")
	sshKeys, err = client.GetSSHKeyList(ctx)
	if err != nil {
		log.Fatalf("Failed to get SSH keys list: %v", err)
	}
	fmt.Printf("Total SSH keys: %d\n", len(sshKeys))
	for _, key := range sshKeys {
		fmt.Printf("  - %s (ID: %d)\n", key.Name, key.ID)
	}
	fmt.Println()

	// Delete SSH key
	fmt.Println("=== Deleting SSH key ===")
	err = client.DeleteSSHKey(ctx, keyID)
	if err != nil {
		log.Fatalf("Failed to delete SSH key: %v", err)
	}
	fmt.Printf("SSH key %d deleted successfully\n", keyID)
}
