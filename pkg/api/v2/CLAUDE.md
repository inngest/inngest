# Inngest REST API v2 - Development Context

This directory contains the implementation of Inngest's REST API v2. This CLAUDE.md provides development context based on the API specification.

## API Design Principles

The v2 API follows these core principles:

### 1. Consistency & Predictability

- **camelCase everywhere**: All JSON fields use camelCase (e.g., `functionId`, `createdAt`, `isEnabled`)
- **Consistent response envelope**: All responses wrap data in `{data, metadata, page?}` structure
- **Resource symmetry**: GET responses can be used as PUT/PATCH input (minus read-only fields)

### 2. Developer Experience

- **Human-readable error codes**: `function_not_found` instead of magic numbers
- **Multiple validation errors**: Return all errors at once when possible
- **Predictable pagination**: Cursor-based with `hasMore` boolean

### 3. HTTP Standards Compliance

- **RESTful methods**: GET (read), POST (create/actions), PUT (full replace), PATCH (partial), DELETE
- **Proper status codes**: 200/201/204 for success, 400/401/403/404/409/422/429 for client errors
- **Standard headers**: Use `Authorization: Bearer`, `Content-Type: application/json`

## Key Implementation Details

### URL Structure

```
Base: /v2
Functions: /v2/functions/{functionId}
Runs: /v2/runs/{runId}
Events: /v2/events/{eventId}
```

### Authentication

- **Signing keys only** for initial launch
- Format: `Authorization: Bearer signkey-{env}-{key}`
- Environment validation via `X-Inngest-Env` header

### Response Format

All successful responses use this envelope:

```json
{
  "data": {...},
  "metadata": {
    "fetchedAt": "2025-08-11T10:30:00Z",
    "cachedUntil": null,
  },
  "page": {...}  // Only for paginated responses
}
```

### Error Format

Errors are always arrays, even for single errors:

```json
{
  "errors": [
    {
      "code": "function_name_required",
      "message": "Function name is required"
    }
  ]
}
```

### ID Formats by Resource Type

- **Events & Runs**: ULIDs (time-sortable)
- **Functions**: Composite `{app-slug}-{function-slug}`
- **Everything else**: UUIDs

### Field Conventions

- **Timestamps**: RFC 3339 UTC format (`2025-08-11T10:30:00Z`)
- **Booleans**: Standard JSON (`true`/`false`)
- **Enums**: SCREAMING_SNAKE_CASE (`COMPLETED`, `RUNNING`)
- **Arrays**: Never null, use empty array `[]`
- **Durations**: Milliseconds as integers (`durationMs`)

### Pagination

Cursor-based using ULIDs/IDs:

```bash
GET /v2/functions?cursor=01hp1zx8m3ng9vp6qn0xk7j4cy&limit=50
```

### Rate Limiting

- Leaky bucket algorithm via throttled/throttled library
- Per-key limits: 1000/hour global, 500/hour reads, 200/hour writes
- Headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`

## Development Guidelines

### When Adding New Endpoints

1. Follow existing URL patterns (`/v2/{resource}` or `/v2/{resource}/{id}`)
2. Use appropriate HTTP methods per resource operation
3. Maintain response envelope consistency
4. Add proper error handling with descriptive error codes
5. Include rate limiting configuration

### When Adding New Fields

- Use camelCase naming
- Include in response metadata
- Make new fields optional to maintain backwards compatibility
- Follow nullable vs optional field semantics from spec
- Always include a short description and summary

### When Modifying Existing Endpoints

- **Never break backwards compatibility within v2**
- Only additive changes allowed (new optional fields, parameters)
- Breaking changes require v3

### Error Handling Best Practices

- Use snake_case error codes with descriptive names
- Provide actionable error messages
- Return multiple validation errors when possible
- Set HTTP status based on first error in array

### Testing Considerations

- Test all HTTP methods for each endpoint
- Verify response envelope structure
- Test pagination edge cases (empty results, single page)
- Validate error response formats
- Test rate limiting behavior

## Common Patterns

### Resource CRUD

```go
// GET /v2/functions/{id} - retrieve
// POST /v2/functions - create
// PUT /v2/functions/{id} - full replace
// PATCH /v2/functions/{id} - partial update
// DELETE /v2/functions/{id} - remove
```

### Action Endpoints

```go
// POST /v2/functions/{id}/invoke - trigger action
// POST /v2/functions/{id}/pause - state change
// POST /v2/runs/{id}/replay - action on resource
```

### Filtering & Querying

```bash
# Use camelCase query parameters
GET /v2/functions?status=active&createdAfter=2025-08-01T00:00:00Z
```

## Technical Implementation

### Architecture

- **REST API Generation**: Uses grpc-gateway to generate REST endpoints from protobuf definitions
- **Auth Middlewar injected**: Auth middleware is optionally injected
- **Proto Files**: Protobuf definitions are located in `proto/api/v2/` directory
- **URL Definitions**: REST URLs are defined using annotations in the proto files defined by the following pattern

```
service V2 {
  rpc Hello(HelloWorldRequest) returns (HelloWorldResponse) {
    option (google.api.http) = {
      get : "/api/v2/helloworld"
    };
  }
}
```

- **Service Implementation**: Core service logic is implemented in `pkg/api/v2/service.go`
- **Transport Layer**: Uses grpc-gateway for gRPC/HTTP protocol handling
- **Port Configuration**: Service mounts to the same port as main application (port 8288)
- **Integration Point**: Service attaches to main application in `pkg/devserver/devserver.go`

### Project Conventions

- Follow existing patterns and conventions used throughout the Inngest project
- Maintain consistency with current codebase architecture and styling
- Use established error handling and logging patterns
- Follow existing authentication and authorization patterns

### Development Workflow

1. Define protobuf service definitions with REST annotations
2. Generate REST endpoints using grpc-gateway
3. Implement service logic in the service file
4. Mount service to main application server
5. Test endpoints follow project testing conventions

## Implementation Status

This specification is **Release Candidate** status. Core patterns are stable, but specific endpoint details may evolve before final v2 release.

## Related Files

- `README.md` - Full API specification
- `service.go` - Core service implementation
- `pkg/devserver/devserver.go` - Service integration point
- Implementation files in this directory follow these patterns
- Test files should validate spec compliance
