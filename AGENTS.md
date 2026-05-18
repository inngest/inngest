# Agent Guide

This file provides guidance to AI coding agents working in this repository.

## Repository Context

Prefer local documentation over broad architectural assumptions. Package-level
docs live alongside the code:

- `pkg/execution/realtime/docs/` - Realtime, broadcaster architecture
- `docs/durable-endpoints/` - Durable Endpoint streaming
- `docs/defer.md` - Defers (deferred-run scheduling)

Active implementation plans live in `docs/plans/`. Use
`scripts/check-plan-status.sh` to review plan status. For release automation,
changelog generation, PR release notes, migration notes, and prerelease
publishing, follow `docs/plans/004-automated-release-cycle.org`.

## Commit and PR Titles

Conventional Commit-style titles are required for any commit or PR title you
create. This repository's changelog configuration in `cliff.toml` enables
`conventional_commits = true` and `filter_unconventional = true`, so
non-conventional titles are easy to lose or misclassify.

Keep title types aligned with `cliff.toml` and `.github/workflows/commits.yml`.
The accepted types are:

- `feat`
- `fix`
- `doc`
- `perf`
- `refactor`
- `style`
- `test`
- `chore`
- `ci`
- `revert`
- `security`
- `cloud`
- `internal`
- `noop`

Use standard conventional formatting such as
`fix(promapi): handle empty query` or
`chore(ci): align changelog workflow permissions`.

Use `cloud`, `internal`, and `noop` only for changes that should not appear in
CLI/self-host release notes or changelogs.

## Release Metadata

When preparing PR text, preserve the release-note contract from plan 004:

- `Description` explains what changed and provides review context.
- `Release note` explains user-visible impact. Use `None` when there is no
  release note.
- `Migration note` covers upgrade, deployment, config, schema, compatibility,
  rollout, or rollback guidance. Use `None` when there is no migration note.

Do not add a second release-note convention. Keep release automation changes
compatible with the sections and placeholders described in
`docs/plans/004-automated-release-cycle.org`.

## Development Commands

- `go run . dev` runs development mode with hot reloading.
- `make dev` is an alias for development mode.
- `go run .` runs the main CLI application.
- `make test` or `go test -test.v ./...` runs the full test suite.
- `make lint` or `golangci-lint run --verbose` runs linting.

## Working Style

- Prefer minimal, targeted changes with tests to verify their behavior
- Commits should be in small logical chunks so each one is self reviewable
- If operating with a plan, update the checklist items as you go if there are any
