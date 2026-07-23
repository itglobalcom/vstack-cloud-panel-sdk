package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"

	sdk "github.com/itglobalcom/vstack-cloud-panel-sdk"
	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

// isPrivateIP checks if the IP address is private
func isPrivateIP(ip string) bool {
	// Remove possible CIDR
	if idx := strings.Index(ip, "/"); idx != -1 {
		ip = ip[:idx]
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// RFC 1918 private ranges
	privateBlocks := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}

	for _, block := range privateBlocks {
		_, subnet, _ := net.ParseCIDR(block)
		if subnet.Contains(parsedIP) {
			return true
		}
	}

	return false
}

// getNetworkPrefix returns the network prefix from an IP address
func getNetworkPrefix(ip string, maskBits int) string {
	// Remove possible CIDR
	if idx := strings.Index(ip, "/"); idx != -1 {
		ip = ip[:idx]
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return ip
	}

	// Create a mask
	mask := net.CIDRMask(maskBits, 32)

	// Apply the mask to get the network address
	network := parsedIP.Mask(mask)

	return fmt.Sprintf("%s/%d", network.String(), maskBits)
}

func runGatewayExample(ctx context.Context, client *sdk.CloudClient) {
	fmt.Println("=== Gateway Operations Example ===")

	// Get a list of all gateways
	fmt.Println("\n=== Getting gateway list ===")
	gateways, err := client.GetGatewayList(ctx)
	if err != nil {
		log.Fatalf("Failed to get gateway list: %v", err)
	}
	fmt.Printf("Found %d gateways\n", len(gateways))

	// Create a network for the gateway
	fmt.Println("\n=== Creating network for gateway ===")
	networkReq := &entities.CreateNetworkRequest{
		Name:          "gateway-test-network",
		LocationID:    "kz",
		Description:   "Test network for gateway demo",
		NetworkPrefix: "10.0.0.0",
		Mask:          24,
	}

	network, err := client.CreateNetworkAndWait(ctx, networkReq)
	if err != nil {
		log.Fatalf("[FATAL] Failed to create network: %v", err)
	}
	fmt.Printf("Network created: ID=%s, Name=%s, State=%s\n",
		network.ID, network.Name, network.State)

	networkID := network.ID

	// Create a new gateway
	fmt.Println("\n=== Creating new gateway ===")
	createReq := &entities.CreateGatewayRequest{
		Name:          "test-gateway",
		LocationID:    "kz",
		BandwidthMbps: 100,
		NetworkIDs:    []string{networkID},
	}

	gateway, err := client.CreateGatewayAndWait(ctx, createReq)
	if err != nil {
		log.Fatalf("[FATAL] Failed to create gateway: %v", err)
	}

	fmt.Printf("Gateway created successfully!\n")
	fmt.Printf("  ID: %s\n", gateway.ID)
	fmt.Printf("  Name: %s\n", gateway.Name)
	fmt.Printf("  State: %s\n", gateway.State)
	fmt.Printf("  Powered On: %v\n", gateway.IsPoweredOn())
	fmt.Printf("  NICs: %d\n", len(gateway.NICs))

	gatewayID := gateway.ID

	// Get gateway information
	fmt.Println("\n=== Getting gateway details ===")
	gateway, err = client.GetGateway(ctx, gatewayID)
	if err != nil {
		log.Fatalf("Failed to get gateway: %v", err)
	}
	fmt.Printf("Gateway: %s (%s)\n", gateway.Name, gateway.ID)
	fmt.Printf("Location: %s\n", gateway.LocationID)
	fmt.Printf("State: %s, Powered: %v\n", gateway.State, gateway.PoweredOn)
	fmt.Printf("NICs:\n")

	// Save IP addresses for use in NAT rules
	var publicIP, privateIP string
	for _, nic := range gateway.NICs {
		fmt.Printf("  - NIC %d: Network=%s, IP=%s, Bandwidth=%d Mbps\n",
			nic.ID, nic.NetworkID, nic.IPAddress, nic.BandwidthMbps)

		// Determine public and private IP
		if publicIP == "" && !isPrivateIP(nic.IPAddress) {
			publicIP = nic.IPAddress
		}
		if privateIP == "" && isPrivateIP(nic.IPAddress) {
			privateIP = nic.IPAddress
		}
	}

	if publicIP == "" {
		log.Fatalf("No public IP found on gateway NICs")
	}
	if privateIP == "" {
		log.Fatalf("No private IP found on gateway NICs")
	}

	fmt.Printf("\nDetected IPs:\n")
	fmt.Printf("  Public IP: %s\n", publicIP)
	fmt.Printf("  Private IP: %s\n", privateIP)

	// Update gateway name
	fmt.Println("\n=== Updating gateway name ===")
	updateReq := &entities.UpdateGatewayRequest{
		Name: "test-gateway-updated",
	}

	gateway, err = client.UpdateGateway(ctx, gatewayID, updateReq)
	if err != nil {
		log.Fatalf("Failed to update gateway: %v", err)
	}
	fmt.Printf("Gateway name updated: %s\n", gateway.Name)

	// Update bandwidth
	fmt.Println("\n=== Updating gateway bandwidth ===")
	bandwidthReq := &entities.UpdateGatewayBandwidthRequest{
		BandwidthMbps: 100,
	}

	gateway, err = client.UpdateGatewayBandwidthAndWait(ctx, gatewayID, bandwidthReq)
	if err != nil {
		log.Fatalf("Failed to update bandwidth: %v", err)
	}
	fmt.Printf("Bandwidth updated successfully\n")

	// Configure NAT rules with real IPs
	fmt.Println("\n=== Configuring NAT rules ===")

	privateNetwork := getNetworkPrefix(privateIP, 24)

	natReq := &entities.UpdateNATRulesRequest{
		NATRules: []entities.NATRule{
			{
				Type:            entities.NATTypeSNAT,
				Protocol:        entities.ProtocolTCP,
				Source:          privateNetwork,
				Destination:     "0.0.0.0/0",
				DestinationPort: 80,
				Translated:      publicIP,
				TranslatedPort:  80,
			},
			{
				Type:            entities.NATTypeDNAT,
				Protocol:        entities.ProtocolTCP,
				Source:          "0.0.0.0/0",
				Destination:     publicIP,
				DestinationPort: 443,
				Translated:      privateIP,
				TranslatedPort:  443,
			},
		},
	}

	fmt.Printf("NAT rules configuration:\n")
	fmt.Printf("  SNAT: %s -> %s (outbound traffic)\n", privateNetwork, publicIP)
	fmt.Printf("  DNAT: %s:443 -> %s:443 (inbound traffic)\n", publicIP, privateIP)

	err = client.UpdateNATRulesAndWait(ctx, gatewayID, natReq)
	if err != nil {
		log.Fatalf("Failed to update NAT rules: %v", err)
	}
	fmt.Printf("NAT rules configured successfully\n")

	// Get NAT rules
	fmt.Println("\n=== Getting NAT rules ===")
	natRules, err := client.GetNATRules(ctx, gatewayID)
	if err != nil {
		log.Fatalf("Failed to get NAT rules: %v", err)
	}
	fmt.Printf("Found %d NAT rules\n", len(natRules))
	for i, rule := range natRules {
		fmt.Printf("  %d. Type=%s, Protocol=%s, %s:%d -> %s:%d\n",
			i+1, rule.Type, rule.Protocol,
			rule.Destination, rule.DestinationPort,
			rule.Translated, rule.TranslatedPort)
	}

	// Configure Firewall rules
	fmt.Println("\n=== Configuring firewall rules ===")
	firewallReq := &entities.UpdateFirewallRulesRequest{
		FirewallRules: []entities.FirewallRule{
			{
				Action:          entities.FirewallActionAllow,
				Direction:       entities.FirewallDirectionIn,
				Protocol:        entities.ProtocolTCP,
				Source:          "0.0.0.0/0",
				Destination:     "10.0.0.0/24",
				DestinationPort: 443,
			},
			{
				Action:      entities.FirewallActionDeny,
				Direction:   entities.FirewallDirectionOut,
				Protocol:    entities.ProtocolTCP,
				Source:      "10.0.0.0/24",
				Destination: "0.0.0.0/0",
				SourcePort:  25,
			},
		},
	}

	err = client.UpdateFirewallRulesAndWait(ctx, gatewayID, firewallReq)
	if err != nil {
		log.Fatalf("Failed to update firewall rules: %v", err)
	}
	fmt.Printf("Firewall rules configured successfully\n")

	// Get Firewall rules
	fmt.Println("\n=== Getting firewall rules ===")
	firewallRules, err := client.GetFirewallRules(ctx, gatewayID)
	if err != nil {
		log.Fatalf("Failed to get firewall rules: %v", err)
	}
	fmt.Printf("Found %d firewall rules\n", len(firewallRules))
	for i, rule := range firewallRules {
		fmt.Printf("  %d. %s %s %s: %s -> %s\n",
			i+1, rule.Action, rule.Direction, rule.Protocol,
			rule.Source, rule.Destination)
	}

	// Add a tag
	fmt.Println("\n=== Adding tag to gateway ===")
	tagReq := &entities.CreateGatewayTagRequest{
		Value: "production",
	}
	err = client.CreateGatewayTag(ctx, gatewayID, tagReq)
	if err != nil {
		log.Fatalf("Failed to add tag: %v", err)
	}
	fmt.Println("Tag 'production' added successfully")

	// Create an additional isolated network
	fmt.Println("\n=== Creating additional network ===")
	additionalNetworkReq := &entities.CreateNetworkRequest{
		Name:          "gateway-additional-network",
		LocationID:    "kz",
		Description:   "Additional network for gateway connection demo",
		NetworkPrefix: "10.200.0.0",
		Mask:          24,
	}

	additionalNetwork, err := client.CreateNetworkAndWait(ctx, additionalNetworkReq)
	if err != nil {
		log.Fatalf("[FATAL] Failed to create additional network: %v", err)
	}
	fmt.Printf("Additional network created: ID=%s, Name=%s\n",
		additionalNetwork.ID, additionalNetwork.Name)

	additionalNetworkID := additionalNetwork.ID

	// Connect an additional network to the gateway
	fmt.Println("\n=== Connecting additional network to gateway ===")
	connectReq := &entities.ConnectNetworkRequest{
		NetworkID: additionalNetworkID,
	}

	err = client.ConnectNetworkAndWait(ctx, gatewayID, connectReq)
	if err != nil {
		log.Fatalf("Failed to connect network: %v", err)
	}
	fmt.Printf("Network connected successfully\n")

	// Show all connected networks
	gateway, err = client.GetGateway(ctx, gatewayID)
	if err != nil {
		log.Fatalf("Failed to get gateway: %v", err)
	}
	fmt.Printf("Total NICs: %d\n", len(gateway.NICs))

	fmt.Println("Connected networks:")
	var additionalNICID int
	for _, nic := range gateway.NICs {
		fmt.Printf("  - NIC %d: Network=%s, IP=%s\n", nic.ID, nic.NetworkID, nic.IPAddress)
		if nic.NetworkID == additionalNetworkID {
			additionalNICID = nic.ID
		}
	}

	// Disconnect the additional network from the gateway
	fmt.Println("\n=== Disconnecting additional network from gateway ===")
	if additionalNICID == 0 {
		log.Fatalf("Failed to find NIC ID for additional network")
	}

	err = client.DisconnectNetworkAndWait(ctx, gatewayID, additionalNICID)
	if err != nil {
		log.Fatalf("Failed to disconnect network: %v", err)
	}
	fmt.Printf("Network disconnected. NIC ID: %d\n", additionalNICID)

	// Check that the network is disconnected
	gateway, err = client.GetGateway(ctx, gatewayID)
	if err != nil {
		log.Fatalf("Failed to get gateway: %v", err)
	}
	fmt.Printf("Remaining NICs: %d\n", len(gateway.NICs))

	// Power management operations
	fmt.Println("\n=== Power management operations ===")

	// Stop the gateway
	fmt.Println("\n--- Stopping gateway ---")
	gateway, err = client.StopGatewayAndWait(ctx, gatewayID)
	if err != nil {
		log.Fatalf("Failed to stop gateway: %v", err)
	}
	fmt.Printf("Gateway stopped. Power status: %v\n", gateway.IsPoweredOn())

	// Start the gateway
	fmt.Println("\n--- Starting gateway ---")
	gateway, err = client.StartGatewayAndWait(ctx, gatewayID)
	if err != nil {
		log.Fatalf("Failed to start gateway: %v", err)
	}
	fmt.Printf("Gateway started. Power status: %v\n", gateway.IsPoweredOn())

	// Restart the gateway
	fmt.Println("\n--- Restarting gateway ---")
	gateway, err = client.RestartGatewayAndWait(ctx, gatewayID)
	if err != nil {
		log.Fatalf("Failed to restart gateway: %v", err)
	}
	fmt.Printf("Gateway restarted successfully. State: %s\n", gateway.State)

	// Remove the tag before deleting the gateway
	fmt.Println("\n=== Removing tag from gateway ===")
	err = client.DeleteGatewayTag(ctx, gatewayID, "production")
	if err != nil {
		log.Fatalf("Failed to delete tag: %v", err)
	}
	fmt.Println("Tag removed successfully")

	// Delete the gateway
	fmt.Println("\n=== Deleting gateway ===")
	err = client.DeleteGateway(ctx, gatewayID)
	if err != nil {
		log.Fatalf("Failed to delete gateway: %v", err)
	}
	fmt.Printf("Gateway %s deleted successfully\n", gatewayID)

	// Delete networks
	fmt.Println("\n=== Deleting networks ===")

	err = client.DeleteNetwork(ctx, networkID)
	if err != nil {
		log.Fatalf("Failed to delete network: %v", err)
	}
	fmt.Printf("Network %s deleted successfully\n", networkID)

	err = client.DeleteNetwork(ctx, additionalNetworkID)
	if err != nil {
		log.Fatalf("Failed to delete additional network: %v", err)
	}
	fmt.Printf("Additional network %s deleted successfully\n", additionalNetworkID)

	fmt.Println("\n=== Gateway operations completed ===")
}
