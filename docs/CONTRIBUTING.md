# Contributing Guide

Hi! Thanks for your interest in contributing to Inngest!

## Getting Started

The following instructions will help you build and run the Inngest CLI locally.

### Prerequisites

- [Git](https://git-scm.com/downloads)
- [Go 1.18 or higher](https://golang.org/doc/install)
- [GoReleaser](https://goreleaser.com/install/)
- [GolangCI-Lint](https://golangci-lint.run/welcome/install/#local-installation)

### Instructions

1. Clone this repository **with submodules**:

   ```sh
   git clone --recurse-submodules https://github.com/inngest/inngest.git
   ```

   If you have already cloned without `--recurse-submodules`, run the
   following from the repo root before building:

   ```sh
   git submodule update --init --recursive
   ```

   The CLI embeds documentation from the
   [`internal/embeddocs/website`](../internal/embeddocs) submodule, so
   `make dev` will fail with `pattern website/pages/docs/*: no matching files
   found` if the submodule has not been initialized.
2. Build the CLI by running `make dev`
   - Your recently built CLI will be available at `./dist/inngest_[system_arch]/inngest` (e.g.
     `./dist/inngest_darwin_arm64/inngest`)
3. Run `./dist/inngest_[system_arch]/inngest` to see the CLI in action
