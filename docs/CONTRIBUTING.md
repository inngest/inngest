# Contributing Guide

Hi! Thanks for your interest in contributing to Inngest!

## Getting Started

The following instructions will help you build and run the Inngest CLI locally.

### Prerequisites

- [Git](https://git-scm.com/downloads)
- [Go](https://golang.org/doc/install) — version per the `go` directive in [`go.mod`](../go.mod) (currently 1.26.4)
- [GoReleaser](https://goreleaser.com/install/)
- [GolangCI-Lint](https://golangci-lint.run/welcome/install/#local-installation)

### Instructions

1. Clone this repository
2. Build the CLI by running `make dev`
   - Your recently built CLI will be available at `./dist/inngest_[system_arch]/inngest` (e.g.
     `./dist/inngest_darwin_arm64/inngest`)
3. Run `./dist/inngest_[system_arch]/inngest` to see the CLI in action
