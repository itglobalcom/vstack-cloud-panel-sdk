package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	sdk "github.com/itglobalcom/vstack-cloud-panel-sdk"
)

func main() {
	// Define command-line flags
	resource := flag.String("resource", "", "Resource type to demonstrate")
	apiKey := flag.String("api-key", os.Getenv("API_KEY"), "API key for authentication")
	apiURL := flag.String("api-url", os.Getenv("API_URL"), "API base URL")

	flag.Parse()

	// Validate required resource argument
	if *resource == "" {
		fmt.Fprintf(os.Stderr, "Error: -resource flag is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Validate required API key
	if *apiKey == "" {
		log.Fatal("Error: API key is required. Set API_KEY environment variable or use -api-key flag")
	}

	// Set default API URL if not specified
	if *apiURL == "" {
		*apiURL = "https://api.example.com/"
	}

	// Create logger
	logger := log.New(os.Stdout, "[vstack-cloud-panel-sdk] ", log.LstdFlags)

	// Create client configuration
	config, err := sdk.NewConfig(
		*apiKey,
		*apiURL,
		sdk.WithTimeout(60*time.Second),
		sdk.WithPollingInterval(10*time.Second),
		sdk.WithLogger(logger),
		sdk.WithLogLevel(sdk.Info),
	)
	if err != nil {
		log.Fatalf("Failed to create config: %v", err)
	}

	// Create API client
	client, err := sdk.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Execute the requested example
	switch *resource {
	case "network":
		runNetworkExample(ctx, client)
	case "ssh":
		runSSHExample(ctx, client)
	case "affinity":
		runAffinityGroupExample(ctx, client)
	case "server":
		runServerExample(ctx, client)
	case "gateway":
		runGatewayExample(ctx, client)
	case "dns":
		runDNSExample(ctx, client)
	case "server_nic":
		runServerNICExample(ctx, client)
	case "volume":
		runVolumeExample(ctx, client)
	case "snapshot":
		runSnapshotExample(ctx, client)
	case "race":
		runRaceConditionTest(ctx, client)
	case "meta":
		runMetadataExample(ctx, client)
	case "vmware_server":
		runVmwareServerExample(ctx, client)
	case "vmware_network":
		runVmwareNetworkExample(ctx, client)
	case "vmware_meta":
		runVmwareMetainfoExample(ctx, client)
	default:
		fmt.Fprintf(os.Stderr, "Error: Unknown resource type '%s'\n\n", *resource)
		flag.Usage()
		os.Exit(1)
	}
}
