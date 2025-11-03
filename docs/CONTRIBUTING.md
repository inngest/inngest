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

1. Clone this repository:
   ```bash
   git clone https://github.com/inngest/inngest.git
   cd inngest
   ```
   
   **Note**: If you already have an existing clone, you'll need to initialize the submodules:
   ```bash
   git submodule update --init --recursive
   ```
   
   Alternatively, for new clones, you can clone with submodules automatically:
   ```bash
   git clone --recurse-submodules https://github.com/inngest/inngest.git
   cd inngest
   ```

2. Build the CLI by running `make dev` (this will initialize submodules automatically)
   - Your recently built CLI will be available at `./dist/inngest_[system_arch]/inngest` (e.g.
     `./dist/inngest_darwin_arm64/inngest`)

3. Run `./dist/inngest_[system_arch]/inngest` to see the CLI in action

### Troubleshooting

**Build errors about missing documentation files:**
If you encounter build errors related to missing files in `internal/embeddocs/website/`, you likely need to initialize the git submodules:

```bash
git submodule update --init --recursive
```

The documentation is stored as a git submodule pointing to the [`inngest/website`](https://github.com/inngest/website) repository.
