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
// The VMware section (types and methods prefixed with Vmware) differs from
// the rest of the SDK in two ways the API dictates. First, its objects are keyed
// by numeric int IDs, whereas the rest of the SDK uses string IDs. Second, its
// background tasks live in their own ID space ("vmw{N}") and are awaited with
// WaitVmwareTask, not with the base task helpers.
//
// Task references are typed: VMware mutators return *VmwareTaskID while base ones
// return *TaskID, so the two families cannot be mixed up by accident. Where only
// the bare string travels, GetTask and GetVmwareTask reject a foreign ID outright
// instead of misdecoding its body — the two share the GET /tasks/{id} endpoint but
// answer with structurally different payloads.
//
// Four things about VMware tasks are worth knowing before writing a control loop
// (a Terraform provider, say):
//
//   - They run long. WaitVmwareTask applies its own floor,
//     VmwareTaskWaitDefaultTimeout, instead of the 2m base PollingTimeout. The
//     floor is a minimum, not a value: a larger WithPollingTimeout raises the
//     wait, a smaller one does not lower it. To wait for less, call
//     WaitVmwareTaskWithTimeout. Rebuild has been measured at up to ~26m, which
//     fits the floor with little headroom, so give it a larger timeout.
//
//   - A mutator that creates something has already created it by the time it
//     returns, even if the wait then fails. The create/copy/rebuild ...AndWait
//     methods name the new server in their error for that reason; the two-step
//     form (CreateVmwareServer and friends) hands the id back directly and is the
//     safer choice when losing track of a server would matter.
//
//   - A completed task does not mean a settled resource. After a rebuild the task
//     reports completed while the replaced server is still "deleting". Use
//     WaitVmwareServerState, WaitVmwareServerGone, WaitVmwareNetworkState or
//     WaitVmwareNetworkGone when the resource itself has to be ready.
//
//   - Some mutations are answered synchronously, with no task at all — editing an
//     isolated network, for one. Those methods return (nil, nil): a nil
//     *VmwareTaskID means "done, nothing to await", not an error.
//     VmwareTaskID.IsZero is nil-safe.
//
// The API also serializes changes per object: a second mutation on the same server
// or network while the first is still running is rejected with APICodeConflict
// (-4000), which the client retries automatically. See IsConflict.
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
