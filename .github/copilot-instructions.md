# Copilot Instructions for Inngest

This document provides guidance for GitHub Copilot when working with the Inngest codebase.

## Project Overview

Inngest is a durable functions platform that replaces queues, state management, and scheduling. The repository contains:
- **Go Backend**: The core Inngest server, CLI, and execution engine
- **TypeScript/Next.js UI**: Dashboard and Dev Server UI applications
- **Protocol Buffers**: API definitions using buf
- **Multi-language SDKs**: TypeScript, Python, Go, and Kotlin SDKs (separate repos)

### Architecture

The system consists of several key components:
- **Event API**: Receives events from SDKs via HTTP requests
- **Event Stream**: Buffers events between API and Runner
- **Runner**: Schedules function runs, manages state, and handles event-driven logic
- **Queue**: Multi-tenant queue with flow control (concurrency, throttling, rate limiting)
- **Executor**: Executes functions, manages steps, handles retries
- **State Store**: Persists function run data, step outputs, and errors
- **Database**: Stores apps, functions, events, and run history
- **API**: GraphQL and REST APIs for management
- **Dashboard UI**: Web interface for managing and monitoring functions

See `/docs/DEVSERVER_ARCHITECTURE.md` for detailed architecture information.

## Development Setup

### Prerequisites

- **Go 1.24+**: Main backend language
- **Node.js & pnpm**: UI development (pnpm@10.18.2)
- **GoReleaser**: For building the CLI
- **GolangCI-Lint**: Code linting
- **Protocol Buffers & buf**: API definitions

### Building

```bash
# Build the CLI
make dev

# Run the dev server
go run ./cmd dev --no-discovery

# Build UI
cd ui && pnpm install && pnpm build
```

### Testing

```bash
# Run Go unit tests
make test

# Run Go unit tests with race detection
go test $(shell go list ./... | grep -v tests) -race -count=1

# Run e2e tests
make e2e

# Run specific e2e test
./tests.sh TestNamePattern

# Run Go e2e tests only
make e2e-golang
```

### Linting

```bash
# Run Go linter
make lint
# or
golangci-lint run

# UI linting is handled by ESLint and Prettier
cd ui && pnpm lint
```

## Code Standards

### Go Code

- **Module**: `github.com/inngest/inngest`
- **Go Version**: 1.24+ (see go.mod)
- **Style**: Follow standard Go conventions
- **Linting**: Uses golangci-lint (see `.golangci.json`)
- **Testing**: Write tests in `*_test.go` files, use table-driven tests
- **Package Organization**:
  - `cmd/`: CLI commands and entry points
  - `pkg/`: Public packages (execution, config, API, etc.)
  - `tests/`: E2E integration tests
  - `proto/`: Protocol buffer definitions

### TypeScript/UI Code

- **Location**: `/ui` directory (monorepo with pnpm workspaces)
- **Apps**: 
  - `ui/apps/dashboard/`: Inngest Cloud dashboard
  - `ui/apps/dev-server-ui/`: Dev Server UI
- **Packages**: `ui/packages/components/`: Shared components
- **Style**: Use Tailwind CSS with color token system
- **Naming**: 
  - Use `ID` (not `Id`) for abbreviations: `environmentID`
  - Follow product nomenclature (e.g., "environment" not "workspace")
  - Use US English spelling (except "Cancelled" for legacy)
  - Use sentence case for UI text: "Click me" not "Click Me"
- **Color Tokens**: Always use color tokens, never hardcoded colors
  - Good: `className="bg-canvasBase"`
  - Bad: `className="bg-white"`
- **CSS**: Prefer Tailwind classes over inline styles

### GraphQL

- Generated using `github.com/99designs/gqlgen`
- Run `make gen` to regenerate after schema changes

### Protocol Buffers

- Definitions in `proto/` directory
- Uses `buf` for code generation
- Run `make protobuf` to regenerate

## Common Tasks

### Adding New API Endpoints

See `/docs/IMPLEMENTING_NEW_REST_API_V2_ENDPOINTS.md` for REST API v2 guidelines.

### Making Pull Requests

See `/docs/PULL_REQUEST_GUIDELINES.md`:
- Keep changes small and atomic
- Focus on one issue or feature per PR
- Write clear descriptions
- Ensure all checks pass before requesting review
- Test thoroughly before submitting

### Releasing

See `/docs/RELEASING.md` for release procedures.

## Important Patterns

### Error Handling

- Use `fmt.Errorf` with `%w` for error wrapping
- Return errors instead of panicking (except in initialization)
- Log errors with structured logging

### State Management

- Function runs maintain state in the State Store
- Steps are atomic units of work that can be retried
- State is persisted after each step completion

### Concurrency

- Use Go's built-in concurrency primitives (goroutines, channels)
- Be mindful of race conditions (use `-race` flag in tests)
- Functions support concurrency control via flow control configuration

### Testing

- Write unit tests for business logic
- Use table-driven tests for multiple scenarios
- E2E tests in `/tests` directory verify SDK integration
- Mock external dependencies
- Run tests with `-count=1` to avoid caching issues

## Key Files and Directories

- `/cmd`: CLI commands and main entry point
- `/pkg/execution`: Core execution engine
- `/pkg/config`: Configuration management
- `/pkg/coreapi`: Core API implementation
- `/pkg/devserver`: Dev server implementation
- `/ui`: Frontend applications and components
- `/tests`: E2E integration tests
- `/proto`: Protocol buffer definitions
- `/docs`: Architecture and development documentation
- `Makefile`: Common development tasks
- `TESTING.md`: Testing guidelines
- `.golangci.json`: Go linter configuration

## Code Owners

See `CODEOWNERS` file for team ownership:
- UI: @anafilipadealmeida @amh4r @djfarrelly @jacobheric
- Execution: @tonyhb @darwin67 @BrunoScheufler @KiKoS0 @jpwilliams
- State: @tonyhb @darwin67 @BrunoScheufler @KiKoS0
- Telemetry: @darwin67 @BrunoScheufler
- Run: @darwin67 @jpwilliams

## Helpful Commands

```bash
# Run dev server with custom tick rate
make run PARAMS="--tick=50"

# Run dev server in debug mode
make debug

# Generate code (GraphQL, protobufs, etc.)
make gen

# Vendor dependencies
make vendor

# Run specific Go test
go test ./pkg/execution -run TestFunctionName -v
```

## Security Considerations

- Never commit secrets or credentials
- Use environment variables for sensitive configuration
- UI uses pnpm with `minimum-release-age` setting for package security
- Follow secure coding practices for input validation and sanitization

## Additional Resources

- [Main README](../README.md): Project overview and getting started
- [Contributing Guide](../docs/CONTRIBUTING.md): How to contribute
- [Architecture](../docs/DEVSERVER_ARCHITECTURE.md): Detailed architecture
- [Testing Guide](../TESTING.md): E2E testing instructions
- [SDK Spec](../docs/SDK_SPEC.md): SDK implementation guidelines
- [Discord Community](https://www.inngest.com/discord): Get help and discuss

## Tips for AI-Assisted Development

- Always run tests after making changes
- Use `make lint` to ensure code quality
- Check existing patterns in the codebase before introducing new ones
- UI changes should follow the established design system and color tokens
- Backend changes should consider the distributed nature of the system
- Test with the dev server locally before submitting PRs
- Review the relevant docs in `/docs` for architecture-specific changes
