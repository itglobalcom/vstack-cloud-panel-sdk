package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	sdk "github.com/itglobalcom/vstack-cloud-panel-sdk"
	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

func runLLMExample(ctx context.Context, client *sdk.CloudClient) {
	fmt.Println("=== LLM Operations Example ===")
	suffix := time.Now().Format("20060102-150405")

	fmt.Println("=== Getting the LLM connection ===")
	connection, err := client.GetLLMConnection(ctx)
	if err != nil {
		log.Fatalf("Failed to get the LLM connection: %v", err)
	}
	fmt.Printf("Connection state: %s\n\n", connection.State)

	if connection.State == entities.LLMConnectionStateNotConnected {
		fmt.Println("=== Connecting the LLM service ===")
		connection, err = client.ConnectLLMAndWait(ctx, &entities.ConnectLLMRequest{
			GroupName:    "sdk-example-first-" + suffix,
			FirstKeyName: "sdk-example-first-key",
		})
		if err != nil {
			log.Fatalf("Failed to connect the LLM service: %v", err)
		}
		fmt.Printf("Connection state: %s\n\n", connection.State)
	}

	fmt.Println("=== Reading the model catalog ===")
	catalog, err := client.GetLLMModelList(ctx)
	if err != nil {
		log.Fatalf("Failed to list LLM models: %v", err)
	}
	fmt.Printf("Found %d models\n", len(catalog.Models))
	var modelIDs []entities.LLMModelID
	for _, m := range catalog.Models {
		fmt.Printf("  - %d %s (%s, %s) %s\n", m.ID, m.TechTitle, m.Kind, m.Contour, m.Endpoint)
		if m.IsAvailableToGroups && len(modelIDs) == 0 {
			modelIDs = append(modelIDs, m.ID)
		}
	}
	fmt.Println()

	fmt.Println("=== Creating a group ===")
	group, err := client.CreateLLMGroupAndWait(ctx, &entities.CreateLLMGroupRequest{Name: "sdk-example-" + suffix})
	if err != nil {
		log.Fatalf("Failed to create an LLM group: %v", err)
	}
	groupID := *group.ID
	fmt.Printf("Group created: ID=%d, Name=%s, Deployment=%s\n\n", groupID, group.Name, group.DeploymentState)

	defer func() {
		fmt.Println("=== Disabling the group ===")
		if _, err := client.DisableLLMGroupAndWait(ctx, groupID); err != nil {
			log.Printf("Failed to disable LLM group %d: %v", groupID, err)
			return
		}
		fmt.Printf("Group %d disabled\n", groupID)
	}()

	if len(modelIDs) > 0 {
		fmt.Println("=== Granting a model to the group ===")
		group, err = client.SetLLMGroupModelsAndWait(ctx, groupID, &entities.SetLLMGroupModelsRequest{ModelIDs: modelIDs})
		if err != nil {
			log.Fatalf("Failed to set the models of LLM group %d: %v", groupID, err)
		}
		fmt.Printf("Group models: %d\n\n", len(group.Models))
	}

	if err := client.SetLLMGroupTags(ctx, groupID, &entities.SetLLMTagsRequest{Tags: []string{"sdk-example"}}); err != nil {
		log.Fatalf("Failed to set the tags of LLM group %d: %v", groupID, err)
	}

	fmt.Println("=== Issuing an access key ===")
	key, err := client.IssueLLMAccessKeyAndWait(ctx, groupID, &entities.IssueLLMAccessKeyRequest{Name: "sdk-example-key"})
	if err != nil {
		log.Fatalf("Failed to issue an LLM access key: %v", err)
	}
	keyID := *key.ID
	fmt.Printf("Access key issued: ID=%d, Name=%s\n\n", keyID, key.Name)

	defer func() {
		fmt.Println("=== Deleting the access key ===")
		if err := client.DeleteLLMAccessKeyAndWait(ctx, groupID, keyID); err != nil {
			log.Printf("Failed to delete LLM access key %d: %v", keyID, err)
			return
		}
		fmt.Printf("Access key %d deleted\n", keyID)
	}()

	fmt.Println("=== Revealing the secret (once) ===")
	secret, err := client.RevealLLMAccessKeySecret(ctx, groupID, keyID)
	if err != nil {
		log.Fatalf("Failed to reveal the secret of LLM access key %d: %v", keyID, err)
	}
	fmt.Printf("Secret received: %d characters\n", len(secret))
	if _, err := client.RevealLLMAccessKeySecret(ctx, groupID, keyID); err != nil {
		fmt.Printf("Second reveal refused (codes %v)\n\n", llmAPICodes(err))
	}

	fmt.Println("=== Locking and unlocking the access key ===")
	key, err = client.LockLLMAccessKeyAndWait(ctx, groupID, keyID, &entities.LockLLMAccessKeyRequest{Reason: "sdk example"})
	if err != nil {
		log.Fatalf("Failed to lock LLM access key %d: %v", keyID, err)
	}
	if key.Locks != nil {
		fmt.Printf("Locked by client: %v\n", key.Locks.ByClient)
	}
	if _, err := client.UnlockLLMAccessKeyAndWait(ctx, groupID, keyID); err != nil {
		log.Fatalf("Failed to unlock LLM access key %d: %v", keyID, err)
	}
	fmt.Println()

	fmt.Println("=== Reading the usage of the month ===")
	now := time.Now()
	usage, err := client.GetLLMUsage(ctx, &entities.LLMUsageRequest{
		From:    now.AddDate(0, 0, 1-now.Day()).Format(entities.LLMDateLayout),
		To:      now.Format(entities.LLMDateLayout),
		GroupBy: entities.LLMUsageGroupByModel,
	})
	if err != nil {
		log.Fatalf("Failed to get the LLM usage: %v", err)
	}
	fmt.Printf("Usage %s..%s: %.2f credits in %d rows, collection %s\n\n",
		usage.From, usage.To, usage.Totals.Credits, len(usage.Items), usage.Collection.State)
}

// llmAPICodes returns the API error codes err carries, if any.
func llmAPICodes(err error) []int {
	var reqErr *sdk.RequestError
	if errors.As(err, &reqErr) {
		return reqErr.Codes
	}
	return nil
}
