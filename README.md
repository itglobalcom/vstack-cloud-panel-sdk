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
- **Project metadata** — locations, images, applications, tasks

Mutating operations that trigger a background task provide an `...AndWait` variant
(for example `CreateServerAndWait`) that polls the task until it finishes:

```go
server, err := client.CreateServerAndWait(ctx, &entities.CreateServerRequest{ /* ... */ })
```

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

Available `RESOURCE` values: `meta`, `server`, `network`, `ssh`, `affinity`,
`dns`, `gateway`, `volume`, `snapshot`, `server_nic`, `race`.

> **Note:** examples other than `meta` create and delete real (billable) resources.
> Use a dedicated test project.

## Documentation
- Go package docs: <https://pkg.go.dev/github.com/itglobalcom/vstack-cloud-panel-sdk>

## Contributing

Contributions are welcome — see [CONTRIBUTING.md](CONTRIBUTING.md).
For security issues, please follow [SECURITY.md](SECURITY.md).

## License

Licensed under the [Apache License 2.0](LICENSE).
