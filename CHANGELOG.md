# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project follows
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.0] - Unreleased

Adds support for the **VMware Cloud** service. This is new surface only — no
previously published type or method was removed or renamed, so existing code keeps
compiling.

### Added

- **VMware Cloud servers**: order (with a dry-run `VerifyVmwareServer`), read, resize,
  rename, change the guest hostname, copy, rebuild, delete, and the five power actions
  (on, off, graceful shutdown, reboot, reset).
- **VMware Cloud server sub-resources**: data volumes, the per-server snapshot, network
  interfaces, and the server firewall.
- **VMware Cloud networks**: isolated, routed and public networks, editing, deletion,
  and attaching servers in a batch.
- **VMware Cloud edge gateway** (routed networks): firewall, NAT and IPsec VPN.
- **VMware Cloud catalogs**: locations (each carrying its disk types and their size
  limits), OS images with a GPU filter, and GPU slicing profiles.
- `...AndWait` variants for every VMware mutator that starts a background task. They
  return the resulting entity where the API makes it identifiable and a plain error
  otherwise. `DeleteVmwareServerAndWait` and `DeleteVmwareNetworkAndWait` additionally
  wait for the object to disappear, because the task alone can complete while it is
  still readable.
- Resource-state waiters for callers that need the resource itself to be ready rather
  than just the task: `WaitVmwareServerState`, `WaitVmwareServerActive`,
  `WaitVmwareServerGone`, `WaitVmwareNetworkState`, `WaitVmwareNetworkActive`,
  `WaitVmwareNetworkGone`.
- `VmwareTaskID`, a task reference distinct from the base `TaskID`, plus
  `WaitVmwareTaskRef`, `IsVmwareTaskID` and `VmwareTaskIDPrefix`. VMware tasks and base
  tasks share the `GET /tasks/{id}` endpoint but answer with structurally different
  bodies, so the two families are kept apart by type.
- `VmwareTaskWaitDefaultTimeout` (30 minutes), the floor `WaitVmwareTask` applies
  instead of the 2-minute `PollingTimeout`. VMware operations are long and vary widely
  in duration — the same server order has been measured at 4.5 and 21.5 minutes, and
  two rebuilds of the same server at 4.5 and 24.5 minutes. `WithPollingTimeout` raises
  the wait above the floor but cannot lower it; use `WaitVmwareTaskWithTimeout` for
  that.
- `RequestError.ErrorParams`, the parsed `error_params` block the API attaches to an
  error to point at the field or list element that failed. This is the only way to tell
  which element of a batch operation was rejected.
- Error helpers and codes: `IsVmwareLocationNotFound`, `IsVmwareNoFreePublicNetwork`,
  `IsNetworkInUse`, `APICodeVmwareLocationNotFound`, `APICodeVmwareNoFreePublicNetwork`,
  `APICodeVmwareInvalidPublicNetworkCapacity`.
- Constants for the VMware enums: server and network states, network types, server and
  edge firewall actions and traffic directions, NAT rule types, VPN encryption types and
  Diffie-Hellman groups, image GPU filter values.
- Runnable examples covering every exported `Vmware*` method: `vmware_meta`,
  `vmware_server`, `vmware_network`. Each provisions what it needs, removes it
  afterwards, and prints a pass/fail tally.
- Unit tests for the VMware task-id handling, path building, input validation and all
  request `Validate()` methods.

### Changed

- `GetTask` now rejects a VMware task id (`vmw{N}`) up front instead of issuing the
  request. Previously such a call either failed with an unmarshal error that looked like
  a broken API, or — for a task carrying no `server_id`/`network_id` — decoded into an
  empty `IsCompleted` and hung the wait on an already-finished task. Use
  `GetVmwareTask` for those ids. Correct callers are unaffected.
- `WaitVmwareTask` logs when a configured `PollingTimeout` below the VMware floor is
  raised, so the effective wait is discoverable from the log.
- Documented previously undocumented exported declarations (`RetryReason.String`,
  `DefaultRetryPolicy`, `ValidationError`, `NewValidationError`, `CreateNetworkRequest`,
  `UpdateNetworkRequest`, `CreateServerTagRequest`).

### Notes for VMware Cloud users

Behaviour of the service API that the SDK cannot paper over, and that a control loop
such as a Terraform provider has to account for:

- **Edge bandwidth is the network's bandwidth.** Set it with `EditVmwareNetwork` and
  read it from `VmwareNetwork.BandwidthMbps`. The SDK deliberately exposes no
  `PUT /edge/bandwidth` wrapper: that endpoint reports success without persisting the
  value, so the API keeps serving the old bandwidth afterwards.
- **A completed task is not a settled resource.** After a rebuild the replaced server
  stays readable in state `deleting` for minutes. Use the resource-state waiters.
- **A create has happened even if the wait fails.** `CreateVmwareServerAndWait`,
  `CopyVmwareServerAndWait` and `RebuildVmwareServerAndWait` name the new server in
  their error for that reason. When losing track of a server would matter, use the
  two-step form, which returns the id directly.
- **Some mutations are answered synchronously**, with no task — editing an isolated
  network, for one. Those methods return `(nil, nil)`; a nil `*VmwareTaskID` means
  "done, nothing to await", not an error.
- **Round-trip asymmetries.** `computer_name` is stored upper-cased;
  `diffie_hellman_group` is written lower case and read back upper case; a DNAT rule's
  `original_ip` is replaced with the edge external address; `peer_network` is written as
  one subnet and read back as the `peer_subnets` list. Compare these
  case-insensitively, or do not treat the written value as the expected state.
- **`bandwidth_mbps` on a NIC** can only be changed for a NIC on a shared/public
  network; on a client-network NIC the field is refused. There, `ip` is the editable
  field, and only when the network has DHCP disabled.
- **The API serializes changes per object.** A second mutation on the same server or
  network while the first is still running is rejected with `APICodeConflict` (-4000),
  which the client retries automatically.

## [1.0.1]

See the repository history for releases before this changelog was introduced.
