# Agent Guide

This file provides guidance to AI coding agents working in this repository.

## Commit Titles

Use clear, scoped commit titles for any commit you create. Conventional commit
titles are preferred when they fit the change, for example `fix(step): preserve
run errors` or `test(connect): cover stale websocket writes`.

Release-visible changes are inferred from conventional commit titles by
`git-cliff`. Use one of the commit types configured in `cliff.toml` so release
PRs can compute the correct version bump and changelog entry.

## Development Commands

### Testing and Quality

- `make utest` runs unit tests for all packages except `./tests` with `-race`.
- `make itest` runs the integration tests in `./tests` with `-race`.
- `make lint` runs `golangci-lint run --verbose`.
- `go test ./...` is useful for a broad local pass when race detection is not
  needed.
- `go test ./path/to/package -run TestName -count=1 -v` is the fastest way to
  iterate on a focused package or test.

### Release Tooling

- `git cliff --bumped-version` computes the next tag from unreleased
  conventional commits.
- `git cliff --tag vX.Y.Z` regenerates `CHANGELOG.md` for a release tag.
- `./release/set_version.sh vX.Y.Z` updates the SDK version constants in
  `version.go` and `pkg/version/version.go`.

## Project Architecture

This repository is the Go SDK for Inngest. It lets users define durable
functions, serve them over HTTP, connect workers over websockets, and interact
with Inngest step primitives from Go code.

### Core Components

1. Root package files such as `client.go`, `funcs.go`, `handler.go`,
   `event.go`, and `options.go` define the public `github.com/inngest/inngestgo`
   API.
2. `step/` implements durable step primitives such as `Run`, `Sleep`, `Send`,
   `Invoke`, `Fetch`, `WaitForEvent`, and `WaitForSignal`.
3. `connect/` implements the websocket-based connect flow, worker API,
   handshake, buffering, invocation, and worker pool behavior.
4. `stephttp/` provides HTTP client/provider support for step-level HTTP
   behavior and websocket integration tests.
5. `realtime/`, `errors/`, `group/`, and `experimental/` expose smaller public
   package surfaces.
6. `internal/` contains shared SDK internals for checkpointing, event and
   function modeling, middleware, request management, opcodes, logging, and
   utility helpers.
7. `pkg/` contains support packages that are public but not part of the root
   SDK surface, such as environment, interval, version, checkpoint, and HTTP
   utilities.
8. `tests/` contains integration tests; package-local `*_test.go` files cover
   unit behavior.
9. `examples/` contains small HTTP, connect, and stephttp applications.

### Key Files

- `go.mod` pins the Go version and module dependencies.
- `Makefile` defines the CI-aligned unit, integration, and lint commands.
- `.github/workflows/go.yml` runs lint, unit tests, and integration tests.
- `.github/workflows/release-pr.yml` opens or updates `release/next` with the
  generated changelog and SDK version bump.
- `.github/workflows/release-tag.yml` tags merged release PRs.
- `.github/workflows/release.yml` publishes GitHub release notes for pushed
  release tags.
- `cliff.toml` defines release-note grouping and version-bump behavior.

## Working Style

- Prefer minimal, targeted changes that preserve the existing Go style.
- Run `gofmt` on touched Go files.
- Run relevant tests for the area you touch when practical.
- Prefer table tests when adding coverage for multiple cases.
- Keep public API changes deliberate; this repository is an SDK and exported
  names are user-facing.
- Keep unit tests outside `./tests`; use `./tests` only for integration behavior
  that needs the test harness.
- Do not hand-edit `vendor/` for ordinary code changes. If dependencies change,
  update module files through the Go toolchain and keep vendored contents
  consistent.
- Commit in small logical chunks so each commit is self-reviewable.
- If working with a plan, update checklist statuses as you go.
- Add comments to clarify intent
