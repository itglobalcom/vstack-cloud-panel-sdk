package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	sdk "github.com/itglobalcom/vstack-cloud-panel-sdk"
	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

// The one-click application example walks the whole scenario: it reads the
// application catalog of a location, orders a server with one of its entries,
// the values of that entry's parameters and a platform language model key where
// the entry offers one, waits for the create task and prints the installations
// the server answers with.
//
// Environment (both optional):
//
//	APPLICATION_LOCATION  location to order in    (default "kz")
//	APPLICATION_ID        catalog entry to order  (default: the first entry of the catalog)

// defaultApplicationExampleLocation is the location the example orders in unless
// APPLICATION_LOCATION says otherwise.
const defaultApplicationExampleLocation = "kz"

// applicationExampleServerName is the name the ordered server carries.
const applicationExampleServerName = "sdk-app-example"

// Resources the order falls back to when the catalog entry recommends none.
const (
	applicationExampleFallbackCPU       = 2
	applicationExampleFallbackRamMB     = 4096
	applicationExampleFallbackStorageMB = 25600
)

func runApplicationExample(ctx context.Context, client *sdk.CloudClient) {
	fmt.Println("=== One-Click Application Example ===")

	locationID := applicationExampleLocation()

	// Read the catalog of the location
	fmt.Printf("\n=== Getting the application catalog of location %s ===\n", locationID)
	applications, err := client.GetApplications(ctx, locationID)
	if err != nil {
		log.Fatalf("Failed to get applications: %v", err)
	}
	fmt.Printf("Applications offered: %d\n", len(applications))
	for _, app := range applications {
		fmt.Printf("  - %s (category: %s, parameters: %d, language model key: %v)\n",
			app.ID, app.Category, len(app.Parameters), app.LLMKeyEnabled)
	}
	if len(applications) == 0 {
		fmt.Println("\nThe project has no application catalog in this location — nothing to order")
		return
	}

	// Pick the entry to order
	app, ok := pickApplicationExampleEntry(applications)
	if !ok {
		log.Fatalf("Application %q is not offered in location %s", applicationExampleID(), locationID)
	}
	if len(app.Images) == 0 {
		log.Fatalf("Application %s lists no compatible image", app.ID)
	}

	fmt.Printf("\n=== Catalog entry %s ===\n", app.ID)
	printApplicationCatalogEntry(app)

	// Build the order: the parameter values, and the key request where the entry
	// offers one
	issueLLMKey := app.LLMKeyEnabled
	parameters := applicationExampleParameters(app, issueLLMKey)

	fmt.Println("\n=== Ordering the server ===")
	fmt.Printf("Values sent with the application:\n")
	for _, p := range app.Parameters {
		value, sent := parameters[p.Name]
		switch {
		case !sent:
			fmt.Printf("  %s: not sent\n", p.Name)
		case p.Secret:
			fmt.Printf("  %s = ****\n", p.Name)
		default:
			fmt.Printf("  %s = %s\n", p.Name, value)
		}
	}
	fmt.Printf("Language model key requested: %v\n", issueLLMKey)

	createReq := &entities.CreateServerRequest{
		Name:       applicationExampleServerName,
		LocationID: app.LocationID,
		ImageID:    app.Images[0],
		CPU:        applicationExampleCPU(app),
		RamMB:      applicationExampleRamMB(app),
		Volumes: []entities.VolumeSpec{
			{
				Name:   "boot",
				SizeMB: applicationExampleStorageMB(app),
			},
		},
		Networks: []entities.NetworkSpec{
			{
				BandwidthMbps: 100,
			},
		},
		Applications: []entities.ApplicationSpec{
			{
				ID:          app.ID,
				Parameters:  parameters,
				IssueLLMKey: issueLLMKey,
			},
		},
		Tags: []string{"example"},
	}

	// The create task completes only once every installation has reached a
	// terminal state, so this single wait covers the install as well.
	fmt.Printf("Waiting for the create task (up to %s)\n", sdk.DefaultPollingTimeout)
	server, err := client.CreateServerAndWait(ctx, createReq)
	if err != nil {
		reportApplicationOrderRefusal(err)
		log.Fatalf("Failed to order a server with %s: %v", app.ID, err)
	}

	fmt.Printf("Server created!\n")
	fmt.Printf("  ID: %s\n", server.ID)
	fmt.Printf("  Name: %s\n", server.Name)
	fmt.Printf("  State: %s\n", server.State)
	fmt.Printf("  Login: %s\n", server.Login)

	// The installations block of the server response carries the outcome of every
	// ordered application. A failed installation is an outcome of a successful
	// order, not a failure of it.
	fmt.Println("\n=== Application installations ===")
	fmt.Printf("Installations: %d\n", len(server.ApplicationInstallations))
	for _, installation := range server.ApplicationInstallations {
		printApplicationInstallation(installation)
	}

	// Clean up: the server is billable
	fmt.Println("\n=== Shutting down the server before deletion ===")
	if _, err := client.ShutdownServerAndWait(ctx, server.ID); err != nil {
		log.Fatalf("Failed to shutdown server: %v", err)
	}

	fmt.Println("\n=== Deleting the server ===")
	if err := client.DeleteServer(ctx, server.ID); err != nil {
		log.Fatalf("Failed to delete server: %v", err)
	}
	fmt.Printf("Server %s deleted successfully\n", server.ID)

	fmt.Println("\n=== One-click application example completed ===")
}

// applicationExampleLocation returns the location the example orders in.
func applicationExampleLocation() string {
	if locationID := os.Getenv("APPLICATION_LOCATION"); locationID != "" {
		return locationID
	}
	return defaultApplicationExampleLocation
}

// applicationExampleID returns the catalog entry the example orders, empty when
// the first entry of the catalog is to be taken.
func applicationExampleID() string {
	return os.Getenv("APPLICATION_ID")
}

// pickApplicationExampleEntry chooses the catalog entry to order and reports
// whether the catalog offers it.
func pickApplicationExampleEntry(applications []entities.Application) (entities.Application, bool) {
	wanted := applicationExampleID()
	if wanted == "" {
		return applications[0], true
	}
	for _, app := range applications {
		if app.ID == wanted {
			return app, true
		}
	}
	return entities.Application{}, false
}

// applicationExampleParameters builds the values the order carries for the entry.
// Every required parameter gets one — the catalog default where the entry
// publishes it. A parameter that belongs to the language model access is left out
// when the order asks for a platform key: the platform fills those itself.
func applicationExampleParameters(app entities.Application, issueLLMKey bool) map[string]string {
	values := make(map[string]string, len(app.Parameters))
	for _, parameter := range app.Parameters {
		if !parameter.Required {
			continue
		}
		if issueLLMKey && parameter.LLMKind != "" {
			continue
		}
		value := parameter.Default
		if value == "" {
			value = fmt.Sprintf("sdk-example-%s", strings.ToLower(parameter.Name))
		}
		values[parameter.Name] = value
	}
	return values
}

// applicationExampleCPU returns the CPU count the order asks for.
func applicationExampleCPU(app entities.Application) int {
	if app.RecommendedCPU > 0 {
		return app.RecommendedCPU
	}
	return applicationExampleFallbackCPU
}

// applicationExampleRamMB returns the RAM size the order asks for.
func applicationExampleRamMB(app entities.Application) int {
	if app.RecommendedRamMB > 0 {
		return app.RecommendedRamMB
	}
	return applicationExampleFallbackRamMB
}

// applicationExampleStorageMB returns the boot volume size the order asks for.
func applicationExampleStorageMB(app entities.Application) int {
	if app.RecommendedStorageMB > 0 {
		return app.RecommendedStorageMB
	}
	return applicationExampleFallbackStorageMB
}

// printApplicationCatalogEntry prints everything the catalog publishes about an
// entry, including the parameters the order has to carry.
func printApplicationCatalogEntry(app entities.Application) {
	fmt.Printf("  Location: %s\n", app.LocationID)
	fmt.Printf("  Category: %s\n", app.Category)
	fmt.Printf("  Documentation: %s\n", app.DocumentationURL)
	fmt.Printf("  Credentials: %s\n", app.CredentialsMode)
	fmt.Printf("  Language model key offered: %v\n", app.LLMKeyEnabled)
	fmt.Printf("  Recommended: %d CPU, %d MB RAM, %d MB storage\n",
		app.RecommendedCPU, app.RecommendedRamMB, app.RecommendedStorageMB)
	fmt.Printf("  Images: %s\n", strings.Join(app.Images, ", "))
	for _, parameter := range app.Parameters {
		fmt.Printf("  Parameter %s (required: %v, secret: %v, default: %q, language model role: %q)\n",
			parameter.Name, parameter.Required, parameter.Secret, parameter.Default, parameter.LLMKind)
	}
}

// printApplicationInstallation prints the outcome of one installation.
func printApplicationInstallation(installation entities.ApplicationInstallation) {
	fmt.Printf("  - %s: %s\n", installation.ID, installation.State)
	if installation.OutcomeReason != "" {
		fmt.Printf("      outcome reason: %s\n", installation.OutcomeReason)
	}
	if installation.InstalledAt != "" {
		fmt.Printf("      finished: %s\n", installation.InstalledAt)
	}
	for _, address := range installation.Addresses {
		fmt.Printf("      address of %s: %s\n", address.Service, address.Address)
	}
	for _, component := range installation.Components {
		fmt.Printf("      component %s (%s): status %q, address %s\n",
			component.Name, component.Kind, component.ObservedStatus, component.Address)
	}
	if installation.AppLogin != "" {
		fmt.Printf("      application login: %s / password: %s\n",
			installation.AppLogin, installation.AppPassword)
	}
	if installation.LLMKeyID != "" {
		fmt.Printf("      language model key: %s\n", installation.LLMKeyID)
	}
}

// reportApplicationOrderRefusal names the refusal an order with applications got.
// Each of them is a 400 with its own API code, so they are told apart by the
// predicates of the SDK rather than by the message.
func reportApplicationOrderRefusal(err error) {
	switch {
	case sdk.IsApplicationNotFound(err):
		fmt.Println("The project catalog does not offer the ordered application")
	case sdk.IsApplicationLLMKeyDisabled(err):
		fmt.Println("The application is offered without a language model key — order it with IssueLLMKey unset")
	case sdk.IsApplicationRequiredParameterNotSet(err):
		fmt.Printf("Required parameters left without a value: %s\n", applicationErrorParameterNames(err))
	case sdk.IsApplicationParameterNotDeclared(err):
		fmt.Printf("Parameters the application does not declare: %s\n", applicationErrorParameterNames(err))
	}
}

// applicationErrorParameterNames lists the parameter names the API attached to a
// refusal in its error_params block.
func applicationErrorParameterNames(err error) string {
	var reqErr *sdk.RequestError
	if !errors.As(err, &reqErr) {
		return "unknown"
	}
	names := make([]string, 0, len(reqErr.ErrorParams))
	for _, param := range reqErr.ErrorParams {
		if param.Name == "Parameter" {
			names = append(names, fmt.Sprint(param.Value))
		}
	}
	if len(names) == 0 {
		return "unknown"
	}
	return strings.Join(names, ", ")
}
