// Package sdk is a Go client library for the vStack Cloud Panel API.
//
// It provides a typed client for managing cloud resources — servers, networks,
// server NICs, volumes, snapshots, SSH keys, DNS zones, gateways, affinity
// groups, VMware Cloud (servers, networks, edge and metadata) and project
// metadata — exposed by the vStack Cloud Panel (for example
// https://api.example.com).
//
// # VMware Cloud
//
// C-13: the VMware section (types and methods prefixed with Vmware) differs from
// the rest of the SDK in two ways the API dictates. First, its objects are keyed
// by numeric int IDs, whereas the rest of the SDK uses string IDs. Second, its
// background tasks are awaited with WaitVmwareTask (task IDs of the form "vmw{N}"),
// not with the base task helpers — see the warning on GetTask. The create methods
// additionally offer "...AndWait" variants (see below).
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
// C-3: in the VMware section only the create methods provide such a variant
// (CreateVmwareServerAndWait and the CreateVmware*NetworkAndWait family). Every
// other VMware mutator returns a task reference that must be awaited explicitly
// with WaitVmwareTask or WaitVmwareTaskWithTimeout.
//
// # Retries and concurrency
//
// The client automatically retries transient failures (configurable HTTP
// statuses and API error codes) with exponential backoff. A CloudClient is safe
// for concurrent use by multiple goroutines and should be reused rather than
// created per request.
package sdk
