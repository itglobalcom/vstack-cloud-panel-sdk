# vstack-cloud-panel-sdk

[![Go Reference](https://pkg.go.dev/badge/github.com/itglobalcom/vstack-cloud-panel-sdk.svg)](https://pkg.go.dev/github.com/itglobalcom/vstack-cloud-panel-sdk)
![Go Version](https://img.shields.io/github/go-mod/go-version/itglobalcom/vstack-cloud-panel-sdk)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

`vstack-cloud-panel-sdk` is a Go client library for the [vStack Cloud Panel] API.
It provides a typed, concurrency-safe client for managing cloud resources such as servers,
networks, volumes, snapshots, SSH keys, DNS zones, gateways and more.

## Features

- Typed client for the vStack Cloud Panel REST API (`/api/v1`).
- Functional options for timeouts, logging, polling and retries.
- Automatic retries with exponential backoff for transient failures.
- `...AndWait` helpers that poll long-running tasks until completion.
- Safe for concurrent use by multiple goroutines.

## Installation

```bash
go get github.com/itglobalcom/vstack-cloud-panel-sdk
```

Requires Go 1.25 or newer.

## Quick start

```go
package main

import (
	"context"
	"fmt"
	"log"

	sdk "github.com/itglobalcom/vstack-cloud-panel-sdk"
)

func main() {
	config, err := sdk.NewConfig(
		"your-api-token",           // API token for your project
		"https://api.example.com", // API base URL
	)
	if err != nil {
		log.Fatal(err)
	}

	client, err := sdk.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}

	project, err := client.GetProject(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Project %d — balance %.2f %s\n", project.ID, project.Balance, project.Currency)
}
```

## Authentication

Requests are authenticated with an API token sent in the `X-API-KEY` header.
Create the token in the vStack Cloud Panel for a specific project and pass it to `NewConfig`.

## Configuration

`NewConfig(apiKey, baseURL string, opts ...Option)` applies sensible defaults that can be
overridden with functional options:

| Option | Default | Description |
| --- | --- | --- |
| `WithTimeout` | `30s` | HTTP request timeout |
| `WithPollingInterval` | `5s` | Poll interval for `...AndWait` operations |
| `WithPollingTimeout` | `2m` | Maximum time to wait for a task |
| `WithUserAgent` | `vstack-cloud-panel-go-sdk/…` | Custom `User-Agent` |
| `WithHTTPClient` | — | Provide a custom `*http.Client` |
| `WithLogger` | no-op | Logger implementing `Printf(format, ...any)` |
| `WithLogLevel` | `Info` | `Debug`, `Info`, `Warn` or `Error` |
| `WithContext` | `context.Background()` | Base context |
| `WithMaxRetries` | `7` | Maximum retry attempts |
| `WithRetryWaitMinMax` | `3s` / `30s` | Backoff bounds |
| `WithRetryableStatus` | `408,429,500,502,503,504` | Retryable HTTP statuses |
| `WithRetryableCodes` | conflict codes | Retryable API error codes |

```go
logger := log.New(os.Stdout, "[vstack] ", log.LstdFlags)

config, err := sdk.NewConfig(token, url,
	sdk.WithTimeout(60*time.Second),
	sdk.WithLogger(logger),
	sdk.WithLogLevel(sdk.Debug),
	sdk.WithMaxRetries(5),
)
```

## Working with resources

The client exposes methods for the following resources:

- **Servers** — create, resize (PUT/PATCH), power operations, delete
- **Networks** & **server NICs**
- **Volumes** & **snapshots**
- **SSH keys**
- **DNS** zones and records
- **Gateways**
- **Affinity groups**
- **VMware Cloud** — servers (power, resize, copy, rebuild, snapshot, volumes, NICs, firewall, nested virtualization), networks (isolated/routed/public), edge (firewall/NAT/VPN) and metadata; tasks via `GetVmwareTask` / `WaitVmwareTask`
- **Project metadata** — locations, images, applications, tasks
- **VMware catalog** (read-only) — locations with their disk types, OS images, GPU slicing profiles

### Public API coverage

The SDK's scope is every Public API operation except the Kubernetes section:
**131 of those 132 operations** are implemented.

The single gap is deliberate. `PUT /api/v1/vmware/networks/{id}/edge/bandwidth`
is **not implemented, on purpose**: the endpoint does not work. Measured against
live production — a request for 30 Mbit/s drove its task to `Completed` while the
network kept reporting `bandwidth_mbps: 20`. Edge bandwidth and network bandwidth
are the same field, so set it with `EditVmwareNetwork`
(`VmwareEditNetworkRequest.BandwidthMbps`) and read it from
`VmwareNetwork.BandwidthMbps`.

Most mutating operations that trigger a background task provide an `...AndWait` variant
(for example `CreateServerAndWait`) that polls the task until it finishes:

```go
server, err := client.CreateServerAndWait(ctx, &entities.CreateServerRequest{ /* ... */ })
```

### VMware catalog

`GetVmwareLocationList`, `GetVmwareImageList` and `GetVmwareGPUModelList` read
the three lookups the VMware section publishes (`/api/v1/vmware`). They are
separate from the vStack lookups (`GetLocations`, `GetImages`), which describe a
different platform and use string identifiers.

Disk types are **not** a catalog of their own: each location carries the disk
types offered in it (`VmwareLocation.DiskTypes`), with the sizes in megabytes to
match `system_disk_size_mb` / `size_mb`, and `Title` as the value the create and
verify requests take. Storage profiles are not published at all.

An unknown (but positive) location id is rejected by the API with a 400 that
`sdk.IsInvalidLocation` recognises:

```go
locations, err := client.GetVmwareLocationList(ctx)
images, err := client.GetVmwareImageList(ctx, &locations[0].ID, nil)
if sdk.IsInvalidLocation(err) {
	log.Fatalf("unknown VMware location")
}
```

`GetVMwareLocations`, `GetVMwareImages` and `GetVMwareGPUModels` are the older
form of the same three reads and are **deprecated** — they answer with a lossier
model (no three-state GPU filter, no way to tell an absent GPU limit from a zero
one). Use the `GetVmware*List` family. `GetVMwareDiskTypes` and
`GetVMwareStorageProfiles` are gone: `/vmware/disk-types` and
`/vmware/storage-profiles` are not part of the Public API and always answered 404.

## Error handling

API errors are returned as `*sdk.RequestError`, which carries the HTTP status,
a parsed message and the API error codes:

```go
if _, err := client.GetServer(ctx, id); err != nil {
	var reqErr *sdk.RequestError
	if errors.As(err, &reqErr) {
		fmt.Println(reqErr.StatusCode, reqErr.Message, reqErr.Codes)
	}
}
```

## Examples

Runnable examples live in [`examples/`](examples/). They call the live API and need a token:

```bash
cp .env.example .env        # then fill in API_KEY and API_URL
make example RESOURCE=meta  # read-only, safe to run first
```

Available `RESOURCE` values: `meta`, `vmware`, `server`, `network`, `ssh`, `affinity`,
`dns`, `gateway`, `volume`, `snapshot`, `server_nic`, `race`.

> **Note:** examples other than `meta` and `vmware` create and delete real (billable) resources.
> Use a dedicated test project.

`vmware_meta`, `vmware_server` and `vmware_network` between them call every exported `Vmware*`
method, each in both forms (raw call plus explicit wait, and `...AndWait`). Each provisions what
it needs, removes it afterwards, and prints an ok/failed/skipped tally. They accept:

| Variable | Default | Purpose |
|---|---|---|
| `VMWARE_LOCATION` | `minsk` | location `tech_title` to work in (matched case-insensitively) |
| `VMWARE_IMAGE_ID` | first Linux image | image for the servers being created |
| `VMWARE_NETWORK_CIDR` | `192.168.94.0` | `/24` base address; the examples bump the third octet |
| `VMWARE_KEEP` | unset | `1` leaves the created resources in place |
| `VMWARE_SKIP_LONG` | unset | `1` skips the copy and rebuild steps (~55 min of `vmware_server`) |

`vmware_server` takes roughly 100 minutes end to end because rebuild alone runs ~20 minutes and
both of its forms are exercised; `VMWARE_SKIP_LONG=1` brings it down to about 40.

## Documentation
- Go package docs: <https://pkg.go.dev/github.com/itglobalcom/vstack-cloud-panel-sdk>
- Release notes: [CHANGELOG.md](CHANGELOG.md)

## Contributing

Contributions are welcome — see [CONTRIBUTING.md](CONTRIBUTING.md).
For security issues, please follow [SECURITY.md](SECURITY.md).

## License

Licensed under the [Apache License 2.0](LICENSE).
