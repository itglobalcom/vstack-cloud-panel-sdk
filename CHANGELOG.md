# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project follows
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.0]

Adds support for the **VMware Cloud** service and brings the base task model up to
the unified model the API publishes.

The release is additive: nothing published is removed, and the declarations that
describe routes the Public API does not serve are marked deprecated instead (see
Deprecated).

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
- `VmwareTask` follows the unified Task model: the status is in `State` (wire field
  `is_completed`, PascalCase enum), and the resources it touched are listed in
  `Resources` (`[]VmwareTaskResource`) instead of dedicated id fields. `NetworkID()`
  and `ServerID()` read the created resource id out of `Resources`; `ResourceID` is
  the general accessor.
- `VmwareTaskWaitDefaultTimeout` (30 minutes), the floor `WaitVmwareTask` applies
  instead of the 2-minute `PollingTimeout`. VMware operations are long and vary widely
  in duration — the same server order has been measured at 4.5 and 21.5 minutes, and
  two rebuilds of the same server at 4.5 and 24.5 minutes. `WithPollingTimeout` raises
  the wait above the floor but cannot lower it; use `WaitVmwareTaskWithTimeout` for
  that.
- `RequestError.ErrorParams`, the parsed `error_params` block the API attaches to an
  error to point at the field or list element that failed. This is the only way to tell
  which element of a batch operation was rejected.
- Error helpers and codes: `IsInvalidLocation`, `IsVmwareNoFreePublicNetwork`,
  `IsNetworkInUse`, `APICodeDCLocationDoesNotExist`, `APICodeVmwareNoFreePublicNetwork`,
  `APICodeVmwareInvalidPublicNetworkCapacity`.
- `entities.VmwareNetwork.EdgeExternalIP` (`edge_external_ip`), the external address of
  the network's edge gateway. It is what a DNAT rule's `original_ip` is normalized to,
  and the network read is the only operation of the contract that publishes it.
- Constants for the VMware enums: server and network states, network types, server and
  edge firewall actions and traffic directions, NAT rule types, VPN encryption types and
  Diffie-Hellman groups, image GPU filter values.
- Runnable examples covering every exported `Vmware*` method: `vmware_meta`,
  `vmware_server`, `vmware_network`. Each provisions what it needs, removes it
  afterwards, and prints a pass/fail tally.
- Unit tests for the VMware task-id handling, path building, input validation and all
  request `Validate()` methods.
- **VMware Cloud nested virtualization**: `EnableVmwareServerNestedHypervisor` and
  `DisableVmwareServerNestedHypervisor` with their `...AndWait` pairs, the
  `NestedHypervisor` field on `VmwareServer` and `VmwareCreateServerRequest`, and
  `VmwareLocation.NestedHypervisorSupported`, which says where the feature is offered.
  An idempotent call creates no task and answers with an empty `*VmwareTaskID`.
- The unified task model on `entities.TaskResponse`: `Type`, `ProgressPercent`,
  `Completed` and `Resources` (`[]TaskResource`). `Resources` is the complete list of
  what a task touched and the only place some resources appear at all — a DNS `ptr`
  record and a K8s cluster have no field of their own. `ResourceID` reads one out.
- `TaskState*` and `TaskResource*`, the single registry of task states and resource
  types. The `VmwareTaskState*` / `VmwareTaskResource*` names are defined as these
  constants, and `entities.VmwareTaskResource` now names `entities.TaskResource`.
- `TaskResponse.IsTerminal`, `IsSucceeded` and `IsFailed`. `is_completed` is a
  five-member enum, not a boolean.
- `AlreadyCompletedTaskID` and `IsAlreadyCompletedTaskID`: the synthetic
  `already_completed_task` id that a synchronous vStack delete returns with
  `?return_task=true`. The wait answers it without a request.

### Changed

- `GetTask` accepts a task id of **any** family, VMware (`vmw{N}`) included, and
  `entities.TaskResponse` decodes all of them: the unified part of the model is the same
  for every service, and the legacy per-resource fields a service does not send stay
  empty. The reason the VMware id used to be rejected does not hold on the current
  contract — a VMware task carries no numeric `server_id`/`network_id`, and
  `is_completed` is always a PascalCase string. `GetVmwareTask` still returns the
  VMware-typed model, whose `ServerID`/`NetworkID` hand back ints. This is a widening;
  no previously working call changes behaviour.
- **Waiting** for a VMware task through the base helpers is still refused, now by the
  wait itself rather than as a side effect of `GetTask`: VMware tasks routinely outrun
  the base `PollingTimeout`. Use `WaitVmwareTask`.
- The base wait leaves its loop on **any** terminal state. `TaskState` has five members
  and `Canceled` is terminal, so a canceled task used to be polled until the wait timed
  out instead of being reported. The error wraps `ErrTaskFailed` and names the state.
- `entities.TaskResponse.KubernetesClusterID` reads `k8s_cluster_id`. The tag was
  `cluster_id`, which no task DTO publishes, so the field was never filled.
- The per-resource `TaskResponse` id fields are marked deprecated, matching the API,
  which marks them `[Obsolete("use resources[]")]`. They are still filled and still
  published. `KubernetesNodeGroupID` is deprecated outright: no task of any service
  carries a node group id, so the field never had a wire counterpart.
- `GetVMwareImages` sends the real GPU filter, `gpu=required`. It sent
  `gpu_only=true`, which is not a declared parameter of the endpoint and was dropped:
  asking for GPU-only images answered with every image.
- `WaitVmwareTask` logs when a configured `PollingTimeout` below the VMware floor is
  raised, so the effective wait is discoverable from the log.
- Documented previously undocumented exported declarations (`RetryReason.String`,
  `DefaultRetryPolicy`, `ValidationError`, `NewValidationError`, `CreateNetworkRequest`,
  `UpdateNetworkRequest`, `CreateServerTagRequest`).

### Deprecated

Nothing is removed — every declaration below stays published and keeps compiling.

- `GetVMwareLocations`, `GetVMwareImages` and `GetVMwareGPUModels`, in favour of
  `GetVmwareLocationList`, `GetVmwareImageList` and `GetVmwareGPUModelList`.
- `GetVMwareDiskTypes` and `GetVMwareStorageProfiles`, with
  `ListVMwareDiskTypesResponse`, `ListVMwareStorageProfilesResponse` and
  `entities.VMwareStorageProfile`. `/vmware/disk-types` and `/vmware/storage-profiles`
  exist only under the AdminV2 prefix; through the Public API both answer 404. Disk
  types travel inside the location (`VmwareLocation.DiskTypes`); storage profiles are
  not published at all.
- `entities.VMwareDiskType`, in favour of `entities.VmwareLocationDiskType`. Its fields
  (`ID`, `MinGB`, `MaxGB`, `StepGB`, `StartValueGB`) do not match the wire — the
  contract publishes disk types inside a location, without an id and in megabytes — so
  they decode to zero.
- `entities.VMwareLocation` now names `VmwareLocation`, so it gains `DiskTypes` and
  `NestedHypervisorSupported`; its three previous fields are unchanged.

### Notes for VMware Cloud users

Behaviour of the service API that the SDK cannot paper over, and that a control loop
such as a Terraform provider has to account for:

- **Edge bandwidth is the network's bandwidth.** Set it with `EditVmwareNetwork` and
  read it from `VmwareNetwork.BandwidthMbps`. The SDK deliberately exposes no
  `PUT /edge/bandwidth` wrapper, and this is the one operation in its scope that is
  intentionally missing: the endpoint reports success without persisting anything.
  Measured against live production — a request for 30 Mbit/s completed its task with
  `Completed` while the network went on reporting `bandwidth_mbps: 20`.
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
