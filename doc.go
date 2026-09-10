// Package sdk is a Go client library for the vStack Cloud Panel API.
//
// It provides a typed client for managing cloud resources — servers, networks,
// server NICs, volumes, snapshots, SSH keys, DNS zones, gateways, affinity
// groups, project metadata and the read-only VMware catalog (locations with
// their disk types, OS images, GPU slicing profiles) — exposed by the vStack
// Cloud Panel (for example https://api.example.com).
//
// # Getting started
//
// Build a configuration with your API token and the API base URL, then create
// a client:
//
//	config, err := sdk.NewConfig(apiKey, "https://api.example.com")
//	if err != nil {
//		log.Fatal(err)
//	}
//	client, err := sdk.NewClient(config)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	project, err := client.GetProject(context.Background())
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(project.ID, project.Balance, project.Currency)
//
// # Authentication
//
// Requests are authenticated with the API token sent in the X-API-KEY header.
// The token is created in the vStack Cloud Panel for a specific project.
//
// # Configuration
//
// NewConfig accepts functional options such as WithTimeout, WithLogger,
// WithLogLevel and WithPollingInterval, plus the retry options WithMaxRetries,
// WithRetryWaitMinMax, WithRetryableStatus and WithRetryableCodes. Sensible
// defaults are applied when options are omitted.
//
// # Long-running operations
//
// Many mutating calls have an "AndWait" variant (for example CreateServerAndWait)
// that polls the associated task until it completes, so callers need not
// implement polling themselves.
//
// The VMware section provides the same variant for every mutator that starts
// a task — creates, edits, deletes, power actions, volumes, snapshots, NICs,
// firewalls, NAT and VPN. Each returns the resulting entity where the API makes it
// identifiable, and a plain error otherwise. The delete variants
// (DeleteVmwareServerAndWait, DeleteVmwareNetworkAndWait) additionally wait for the
// object to actually disappear, because the task alone completes too early.
//
// The non-waiting form of each method is still available and returns a
// *VmwareTaskID; await it with WaitVmwareTaskRef, or with WaitVmwareTask /
// WaitVmwareTaskWithTimeout for a bare ID.
//
// # Retries and concurrency
//
// The client automatically retries transient failures (configurable HTTP
// statuses and API error codes) with exponential backoff. A CloudClient is safe
// for concurrent use by multiple goroutines and should be reused rather than
// created per request.
package sdk
