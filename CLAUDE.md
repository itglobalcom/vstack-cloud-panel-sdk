# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Before proposing a change

```bash
make fmt && make vet && go build ./... && go test -race ./...
```

That, not `make test`: CI runs the tests with `-race`, and `go build ./...` also covers
`./examples` — a real package, and it breaks the build. `gofmt -l .` must print nothing, or CI
fails on its very first step.

## Repository

GitHub here is a read-only mirror of an internal GitLab: work on a branch, never commit to `main`.
`.env` is gitignored, holds live credentials and is picked up by the Makefile automatically —
mirror new variables into `.env.example`. Changes to the SDK's exported surface — types, methods,
their behaviour — are recorded in `CHANGELOG.md`.

## Layout

- **root `sdk`** — one file per resource, every call a method on `*CloudClient`. Request
  scaffolding and auth live in `client.go`, so the methods never touch headers;
- **`entities/`** — request and response structures. Requests have `Validate()`, and the client
  calls it before going to the network; when adding a mutating method, keep that pair;
- **`examples/`** — one program with a `-resource` flag, a new example is a new `case`.

Classify errors with the `Is*` helpers from `errors.go` rather than by comparing codes by hand.

## Documentation is the comments

There is no separate build: pkg.go.dev renders the doc comments straight from the published module
(to preview — `go doc -all .`). **gofmt owns their formatting**: it rewrites lists, indentation and
headings, so a comment that is off-canon lands in `gofmt -l` and fails CI. `make fmt` is needed
after documentation too, not only after code.

The style in the code is consistent: start with the name (`IsZero reports whether…`),
`reports whether` for booleans and `returns` for values, complete sentences, comment everything
exported. Above all — **explain why, not what**: restating the signature is worth nothing, the
valuable comment records the constraint behind the code. The package comment lives in `doc.go`
alone and describes contracts that signatures cannot express. Internal context has no place there:
ticket numbers, tracker links and notes to a colleague say nothing to a reader of the published
documentation, and godoc prints them verbatim.

## Tests

`go test` is offline and narrow — it reaches validation and stops. **There is no separate e2e
suite; `examples/` is it.** The VMware examples run against a live project, provision and clean up
everything themselves, record each call as pass/fail and exit non-zero on failure. So a new
exported method is not covered until an example calls it — and the runs are long and billable
(`VMWARE_SKIP_LONG=1`, `VMWARE_KEEP=1`, see the header of `examples/vmware_common.go`).
