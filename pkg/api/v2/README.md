# REST API v2 Spec (RC)

# Overview

This document defines the Inngest REST API v2. The API design prioritizes consistency, speed of development and ease of implementation while maintaining essential functionality. Once this document is finalized, it will be added to the monorepo and inngest projects in the /v2 directory as a [README.md](http://README.md) file and transformed into a [CLAUDE.md](http://CLAUDE.md) file so our friendly robot can help enforce the spec.

# URL Structure

## Base URL Structure

```
Production:    https://api.inngest.com/v2
Dev Server:    http://localhost:8288/api/v2
```

## Resource Endpoints (examples)

### Functions

```
GET    /v2/functions                      # List functions
GET    /v2/functions/{functionId}         # Get function details
PUT    /v2/functions/{functionId}         # Update function (full replace)
DELETE /v2/functions/{functionId}         # Delete function
POST   /v2/functions/{functionId}/invoke  # Trigger function execution
POST   /v2/functions/{functionId}/pause   # Pause function
POST   /v2/functions/{functionId}/resume  # Resume function
```

### Runs

```
GET    /v2/runs                          # List runs
GET    /v2/runs/{runId}                  # Get run details
DELETE /v2/runs/{runId}                  # Cancel run
POST   /v2/runs/{runId}/replay           # Replay run
```

### Events

```
GET    /v2/events                        # List events
GET    /v2/events/{eventId}              # Get event details
POST   /v2/events                        # Send event
GET    /v2/events/{eventId}/runs         # Get runs triggered by event
```

## Environment Header

Environment is passed via header (consistent with v1):

```bash
# Development - no header needed
curl http://localhost:8288/v2/functions

# Production - specify environment
curl https://api.inngest.com/v2/functions \
  -H "X-Inngest-Env: production" \
  -H "Authorization: Bearer signkey-prod-abc123..."
```

---

# Parameter Naming

## Consistent camelCase Strategy

All API fields use camelCase for consistency.

### Core Field Name Examples

```json
{
  "id": "01hp1zx8m3ng9vp6qn0xk7j4cy",
  "functionId": "user-signup",
  "name": "User Signup Handler",
  "description": "Handles user signups",
  "enabled": true,
  "createdAt": "2025-08-11T10:30:00Z",
  "updatedAt": "2025-08-11T10:35:00Z"
}
```

### Boolean Field Name Examples

```json
{
  "enabled": true,
  "isActive": true,
  "hasErrors": false,
  "canRetry": true
}
```

### Time and Duration Field Name Examples

NOTE: See SDK for duration and timeout examples below, matches behavior.

```json
{
	"createdAt": "2025-08-11T10:30:00Z",    # Creation timestamp
	"startedAt": "2025-08-11T10:30:00Z",    # Start timestamp
  "completedAt": "2025-08-11T10:30:05Z",  # Completion timestamp
  "duration": "5s",                       # Duration in seconds
  "timeout": "5m"                         # Timeout in minutes
}
```

### Benefits of camelCase

- **JavaScript / TypeScript Native**: Natural object property access
- **Modern APIs**: AWS, Azure, Google Cloud all use camelCase

---

# Response Envelope

## Consistent Response Structure with Metadata

All successful responses use a consistent envelope with data, metadata, and optional pagination:

```json
{
  "data": {...},                          # Actual response data
  "metadata": {
    "fetchedAt": "2025-08-11T10:30:00Z",  # Response generation timestamp
    "cachedUntil": null,                  # Cache expiration (null if not cached)
  },
  "page": {...}                           # Only for paginated collections
}
```

## Single Resource Response

```json
{
  "data": {
    "id": "01hp1zx8m3ng9vp6qn0xk7j4cy",
    "functionId": "user-signup",
    "name": "User Signup Handler",
    "enabled": true,
    "createdAt": "2025-08-11T10:30:00Z"
  },
  "metadata": {
    "fetchedAt": "2025-08-11T10:30:00Z",
    "cachedUntil": "2025-08-11T10:35:00Z"
  }
}
```

## Collection Response (Paginated)

```json
{
  "data": [
    {
      "id": "01hp1zx8m3ng9vp6qn0xk7j4cy",
      "functionId": "user-signup",
      "name": "User Signup Handler"
    },
    {
      "id": "01hp1zx8m3ng9vp6qn0xk7j4cz",
      "functionId": "send-email",
      "name": "Send Email Handler"
    }
  ],
  "metadata": {
    "fetchedAt": "2025-08-11T10:30:00Z",
    "cachedUntil": null
  },
  "page": {
    "cursor": "01hp1zx8m3ng9vp6qn0xk7j4cz",
    "hasMore": true,
    "limit": 50
  }
}
```

## Empty Collection Response

```json
{
  "data": [],
  "metadata": {
    "fetchedAt": "2025-08-11T10:30:00Z",
    "cachedUntil": "2025-08-11T10:35:00Z"
  },
  "page": {
    "hasMore": false,
    "limit": 50
  }
}
```

## Metadata Fields

### Required Fields

- **fetchedAt**: ISO 8601 timestamp when the response was generated
- **cachedUntil**: ISO 8601 timestamp when cached response expires (`null` if not cached)

## Benefits

- **V1 Compatibility**: Maintains familiar metadata structure from v1 API
- **Cache Transparency**: Clients know when data expires
- **Debugging**: Response timestamps aid troubleshooting
- **Predictable Structure**: Every response follows same pattern
- **Client Optimization**: Enables client-side cache management

---

# Resource Symmetry

## Round-Trip Compatibility

The API maintains **resource symmetry** wherever possible, meaning the shape of data returned from GET operations can be used directly as input for PUT/PATCH operations on the same resource. This round-trip compatibility simplifies client implementations and provides a more intuitive developer experience.

## Design Principle

**Best Effort Commitment**: The API strives to maintain identical or highly compatible schemas between read and write operations for the same resource type, while acknowledging that some fields may be read-only or computed.

### Symmetric Operations

```bash
# GET a function
GET /v2/functions/app-123:user-signup
{
  "data": {
    "id": "app-123:user-signup",
    "functionId": "user-signup",
    "name": "User Signup Handler",
    "description": "Handles new user registrations",
    "timeout": 30,
    "retries": 3,
    "enabled": true,
    "createdAt": "2025-08-11T10:30:00Z",
    "updatedAt": "2025-08-11T10:35:00Z"
  }
}

# PUT the same function (using same structure)
PUT /v2/functions/app-123:user-signup
{
  "functionId": "user-signup",
  "name": "User Signup Handler",
  "description": "Handles new user registrations",
  "timeout": 30,
  "retries": 3,
  "enabled": true
  // Read-only fields (id, createdAt, updatedAt) omitted but not required
}

```

## Read-Only Fields

Some fields are computed or system-managed and cannot be modified through write operations:

### System Fields

- **id**: Resource identifier (system-generated)
- **createdAt**: Creation timestamp (system-set)
- **updatedAt**: Last modification timestamp (system-managed)

### Computed Fields

- **status**: Derived from current resource state
- **metrics**: Calculated performance data
- **relationships**: Dynamic references to other resources

```json
{
  "data": {
    "id": "run_123", // Read-only: system identifier
    "functionId": "user-signup", // Writable: user-provided
    "status": "completed", // Read-only: computed state
    "output": { "success": true }, // Read-only: execution result
    "startedAt": "2025-08-11T10:30:00Z", // Read-only: system timestamp
    "completedAt": "2025-08-11T10:30:05Z", // Read-only: system timestamp
    "durationMs": 5000 // Read-only: computed value
  }
}
```

## Practical Benefits

### Client Implementation

```jsx
// Fetch, modify, and update pattern
const response = await fetch("/v2/functions/app-123:user-signup");
const functionData = response.data;

// Modify specific fields
functionData.timeout = 60;
functionData.description = "Updated description";

// Send back (read-only fields ignored)
await fetch("/v2/functions/app-123:user-signup", {
  method: "PUT",
  body: JSON.stringify(functionData),
});
```

### Tooling Advantages

- **Generic CRUD interfaces**: Build once, work with any resource
- **Configuration management**: Export/import resource definitions
- **Backup and restore**: Consistent data format across operations
- **Testing**: Simplified integration test patterns

## Limitations and Exceptions

### When Symmetry Breaks

Resource symmetry may not be maintained when:

- **Security concerns**: Sensitive fields not returned in responses
- **Performance optimization**: Large computed fields excluded from standard responses
- **API evolution**: New fields added to responses before write support

### Bulk Operations

Bulk endpoints may use different schemas optimized for their specific use case:

```bash
# Individual resource maintains symmetry
GET /v2/functions/app-123:user-signup  # Full resource shape
PUT /v2/functions/app-123:user-signup  # Same shape (minus read-only)

# Bulk operations may differ
POST /v2/functions/bulk-update         # Optimized bulk format

```

## Documentation Promise

When resource symmetry is **not** maintained, the API documentation will:

1. **Clearly indicate** which fields are read-only
2. **Explain differences** between input and output schemas
3. **Provide examples** showing compatible usage patterns
4. **Document workarounds** when round-trip compatibility is impossible

## Evolution Strategy

As the API evolves, resource symmetry will be maintained by:

- **Additive changes only**: New optional fields in responses
- **Graceful degradation**: Unknown fields in requests are ignored
- **Clear migration paths**: When symmetry must break, provide transition period
- **Version compatibility**: Symmetry maintained within each major version

This commitment to resource symmetry reduces the cognitive overhead for API consumers and enables more predictable, maintainable client implementations.

---

# Error Format

## HTTP Status + Error Codes + Consistent Array Format

Errors use HTTP status codes for categorization with human-readable error codes and descriptive messages. The API always returns errors as an array, whether there's one error or multiple errors. When possible, the API returns all validation errors in a single response rather than stopping at the first error. The HTTP status is set by the first error encountered and will correspond to the first error in the array.

## Standard Error Structure

### Always an Array

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

### Multiple Errors

```json
{
  "errors": [
    {
      "code": "function_name_required",
      "message": "Function name is required"
    },
    {
      "code": "function_timeout_out_of_range",
      "message": "Timeout must be between 1 and 3600 seconds"
    }
  ]
}
```

## Error Code Naming Convention

Error codes use `snake_case` with descriptive, human-readable names:

```
Examples:
function_name_required
function_timeout_out_of_range
function_not_found
run_already_completed
signing_key_invalid
rate_limit_exceeded

```

## Examples by Status Code

### 400 Bad Request

### Single Validation Error

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

### Multiple Validation Errors

```json
{
  "errors": [
    {
      "code": "function_name_required",
      "message": "Function name is required"
    },
    {
      "code": "function_timeout_out_of_range",
      "message": "Timeout must be between 1 and 3600 seconds"
    },
    {
      "code": "function_retries_invalid",
      "message": "Retries must be a non-negative integer"
    }
  ]
}
```

### 401 Unauthorized

```json
{
  "errors": [
    {
      "code": "authorization_header_missing",
      "message": "Authorization header is required"
    }
  ]
}

{
  "errors": [
    {
      "code": "signing_key_invalid",
      "message": "Invalid or expired signing key"
    }
  ]
}

```

### 403 Forbidden

```json
{
  "errors": [
    {
      "code": "function_access_denied",
      "message": "Access denied to function 'user-signup' in environment 'production'"
    }
  ]
}

{
  "errors": [
    {
      "code": "environment_access_denied",
      "message": "Signing key does not have access to environment 'production'"
    }
  ]
}

```

### 404 Not Found

```json
{
  "errors": [
    {
      "code": "function_not_found",
      "message": "Function 'user-signup' not found"
    }
  ]
}

{
  "errors": [
    {
      "code": "run_not_found",
      "message": "Run 'abc123' not found"
    }
  ]
}

```

### 409 Conflict

```json
{
  "errors": [
    {
      "code": "function_already_exists",
      "message": "Function 'user-signup' already exists"
    }
  ]
}

{
  "errors": [
    {
      "code": "run_already_completed",
      "message": "Run 'abc123' is already completed and cannot be cancelled"
    }
  ]
}

```

### 422 Unprocessable Entity

```json
{
  "errors": [
    {
      "code": "connect_too_many_apps_per_connection",
      "message": "Maximum of 10 apps allowed per connection"
    }
  ]
}

{
  "errors": [
    {
      "code": "function_circular_dependency",
      "message": "Function dependency would create a circular reference"
    }
  ]
}

```

### 429 Too Many Requests

```json
{
  "errors": [
    {
      "code": "rate_limit_exceeded",
      "message": "Rate limit exceeded. Try again in 5 minutes"
    }
  ]
}

{
  "errors": [
    {
      "code": "invoke_rate_limit_exceeded",
      "message": "Function invocation rate limit exceeded for 'user-signup'"
    }
  ]
}

```

### 500 Internal Server Error

```json
{
  "errors": [
    {
      "code": "internal_server_error",
      "message": "An unexpected error occurred"
    }
  ]
}
```

### 501 Not Implemented

```json
{
  "errors": [
    {
      "code": "feature_not_implemented",
      "message": "Environment creation is not supported in dev server"
    }
  ]
}
```

## Multiple Error Behavior

### When Multiple Errors Are Returned

The API makes a best effort to returns all errors when:

- **Request validation**: All field validation errors
- **Bulk operations**: Errors for each failed item
- **Complex business logic**: All constraint violations

```json
{
  "errors": [
    {
      "code": "function_not_found",
      "message": "Function 'invalid-func' not found",
      "context": { "functionId": "invalid-func" }
    },
    {
      "code": "function_name_too_long",
      "message": "Function name exceeds 100 character limit",
      "context": { "functionId": "very-long-function-name", "length": 150 }
    }
  ]
}
```

### When Single Error Is Returned

The API returns one error when:

- **Authentication failures**: Stop processing immediately
- **Authorization failures**: Security-sensitive operations
- **System errors**: Infrastructure or dependency failures
- **Rate limiting**: Request rejected before processing

## Error Message Guidelines

### Be Specific and Actionable

```json
// ✅ Good: Specific and actionable
{
  "errors": [
    {
      "code": "function_timeout_out_of_range",
      "message": "Function timeout must be between 1 and 3600 seconds"
    }
  ]
}

// ❌ Bad: Vague and unhelpful
{
  "errors": [
    {
      "code": "invalid_input",
      "message": "Invalid input"
    }
  ]
}

```

### Use Plain Language

```json
// ✅ Good: Clear and direct
{
  "errors": [
    {
      "code": "email_required",
      "message": "Email address is required"
    }
  ]
}

// ❌ Bad: Technical jargon
{
  "errors": [
    {
      "code": "email_validation_constraint_violation",
      "message": "Email field validation constraint violation"
    }
  ]
}

```

## Error Code Categories

### Common Prefixes

```
validation_*     # Input validation errors
function_*       # Function-specific errors
run_*            # Run-specific errors
event_*          # Event-specific errors
auth_*           # Authentication errors
rate_limit_*     # Rate limiting errors
```

### Common Suffixes

```
*_required       # Missing required field
*_invalid        # Invalid format or value
*_not_found      # Resource doesn't exist
*_already_exists # Resource conflict
*_out_of_range   # Value outside allowed range
*_too_long       # Exceeds length limit
*_exceeded       # Limit exceeded
```

## Benefits

- **Consistent Structure**: Always an array, whether one error or many
- **Machine Readable**: Error codes enable programmatic error handling
- **Human Readable**: Codes use descriptive names, not cryptic numbers
- **Complete Feedback**: Multiple errors help developers fix all issues at once
- **HTTP Standards**: Leverages existing status code semantics
- **Client-Friendly**: Simple array iteration for all error handling
- **Debuggable**: Rich error information aids troubleshooting
- **Predictable**: Same format regardless of error count

---

# Pagination

## Cursor-Based Pagination

The API uses a **surrogate cursor** approach for future flexibility. This will allow the cursor to evolve to include additional sorting or filtering criteria without breaking existing clients.

The `hasMore` boolean is calculated by querying `pageSize + 1` records and checking if more results exist

### Request Format

```bash
# First page
GET /v2/functions?limit=50

# Subsequent pages
GET /v2/functions?cursor=01hp1zx8m3ng9vp6qn0xk7j4cy&limit=50
```

### Query Parameters

```
cursor    # string  - Cursor from previous response (omit for first page)
limit     # integer - Items to return (default: 50, max: 250)
```

### Response Format

```json
{
  "data": [...],
  "page": {
    "cursor": "01hp1zx8m3ng9vp6qn0xk7j4cz",    # Last item from current page
    "hasMore": true,                                 # Whether more pages exist
    "limit": 50                                      # Items per page
  }
}
```

### Implementation with ULIDs

ULIDs naturally sort by creation time, making cursor pagination efficient:

```sql
-- First page
SELECT * FROM functions ORDER BY id LIMIT 50;

-- Next page
SELECT * FROM functions
WHERE id > '01hp1zx8m3ng9vp6qn0xk7j4cy'
ORDER BY id LIMIT 50;
```

### Benefits

- **Performance**: O(log n) queries regardless of dataset size
- **Consistency**: No duplicates
- **Simple**: Only need last item ID as cursor
- **ULID Compatible**: Natural ordering by creation time

---

# HTTP Methods and Status Codes

## RESTful Method Usage

### GET - Read Data (Safe, Idempotent)

_NOTE: JSON body is NOT allowed to be used in a GET request_

**CRUD Operation**: READ

**Data Mutation**: None

**Use Case**: Retrieve resources without side effects

```bash
GET /v2/functions/user-signup        # Read function → 200 OK, 404 Not Found
GET /v2/runs?status=completed        # Read runs → 200 OK
```

### POST - Create Resources or Execute Actions (Unsafe, Non-Idempotent)

**CRUD Operations**: CREATE

**Data Mutation**: Creates new resources, triggers actions

**Use Case**: Resource creation and action execution

```bash
POST /v2/functions                    # Create function → 201 Created, 409 Conflict
POST /v2/functions/user-signup/invoke # Execute action → 200 OK, 202 Accepted
POST /v2/events                       # Create event → 201 Created
POST /v2/runs/abc123/replay           # Execute action → 200 OK
```

### PUT - Update Entire Resource (Unsafe, Idempotent)

_NOTE: Docs need to include explicit instructions that this is a full replacement and omitted, optional fields will be unset or reset to default values_

**CRUD Operation**: UPDATE (full replacement)

**Data Mutation**: Replaces entire resource

**Use Case**: Complete resource replacement

```bash
PUT /v2/functions/user-signup        # Update function → 200 OK, 201 Created# Note: Full resource replacement - all fields required
```

### PATCH - Update Partial Resource (Unsafe, Non-Idempotent)

_NOTE: Even though this is a partial update, the entire resource will be returned in the response._

**CRUD Operation**: UPDATE (partial modification)

**Data Mutation**: Modifies specific fields only

**Use Case**: Partial resource updates

```bash
PATCH /v2/functions/user-signup      # Partial update → 200 OK, 404 Not Found
# Note: Only specified fields are updated, others remain unchanged
```

### DELETE - Remove Resources or Cancel Actions (Unsafe, Idempotent)

**CRUD Operation**: DELETE

**Data Mutation**: Removes resources or cancels operations

**Use Case**: Resource deletion and operation cancellation

```bash
DELETE /v2/functions/user-signup     *# Delete function → 200 OK, 204 No Content, 404 Not Found*
DELETE /v2/runs/run_abc123           *# Cancel run → 200 OK, 404 Not Found*
```

## Status Code Guidelines

### Success Responses (2xx)

```
200 OK          # Resource retrieved or action completed
201 Created     # Resource created successfully
202 Accepted    # Action accepted, processing asynchronously
204 No Content  # Action completed, no response body returned
```

### Cache Responses (3xx)

```
304 Not Modified       # Resource unchanged since last request (caching)
```

### Client Errors (4xx)

```
400 Bad Request          # Invalid request format or parameters
401 Unauthorized         # Authentication required or failed
403 Forbidden            # Authorization failed
404 Not Found            # Resource doesn't exist
409 Conflict             # Resource state conflict
422 Unprocessable Entity # Valid format but business logic error
429 Too Many Requests    # Rate limiting exceeded
```

### Server Errors (5xx)

```
500 Internal Server Error # Unexpected server error
501 Not Implemented       # Feature not implemented (dev server)
503 Service Unavailable   # Temporary service issue
```

---

# Query Parameters

## Query Parameter Naming Conventions

### snake_case for URL params

All query parameters use camelCase. This is consistent with the json body params:

```bash
# ✅ Correct: camelCase parameters
GET /v2/functions?functionId=user-signup&isEnabled=true&createdAfter=2025-08-01T00:00:00Z

# ❌ Avoid: snake_case or inconsistent naming
GET /v2/functions?function_id=user-signup&is_enabled=true&created_after=2025-08-01T00:00:00Z
```

## Standard Parameter Patterns

### Pagination Parameters

```bash
GET /v2/functions?cursor=abc123&limit=50
```

### Filtering Parameters

```bash
# Single value filters
GET /v2/functions?status=active
GET /v2/runs?functionId=user-signup

# Multiple values (comma-separated)
GET /v2/runs?status=completed,failed
```

### Time Range Parameters

```bash
# ISO 8601 timestamps
GET /v2/runs?startedAfter=2025-08-01T00:00:00Z
GET /v2/runs?completedBefore=2025-08-11T23:59:59Z

# Date ranges
GET /v2/functions?createdAfter=2025-08-01T00:00:00Z&createdBefore=2025-08-11T23:59:59Z
```

### Array Parameters

```bash
# Multiple statuses
GET /v2/runs?status=completed,failed
GET /v2/functions?status=active,paused,archived

# Multiple array parameters
GET /v2/runs?status=completed,failed&functionId=user-signup,send-email
GET /v2/functions?tags=auth,users&status=active,paused
```

### URL Encoding

```bash
# CEL example
function.name == "user-signup" && run.status in ["completed", "failed"] && run.duration > duration("30s")

# URL encoded request
GET /v2/runs?celFilter=function.name%20%3D%3D%20%22user-signup%22%20%26%26%20run.status%20in%20%5B%22completed%22%2C%20%22failed%22%5D%20%26%26%20run.duration%20%3E%20duration%28%2230s%22%29&limit=50
```

## URL Encoding Functions by Language

| Language       | Encode Function                     | Decode Function            | Query Builder                    |
| -------------- | ----------------------------------- | -------------------------- | -------------------------------- |
| **C#**         | `Uri.EscapeDataString()`            | `Uri.UnescapeDataString()` | `HttpUtility.ParseQueryString()` |
| **Go**         | `url.QueryEscape()`                 | `url.QueryUnescape()`      | `url.Values{}`                   |
| **Java**       | `URLEncoder.encode()`               | `URLDecoder.decode()`      | Manual building                  |
| **JavaScript** | `encodeURIComponent()`              | `decodeURIComponent()`     | `URLSearchParams`                |
| **PHP**        | `rawurlencode()`                    | `rawurldecode()`           | `http_build_query()`             |
| **Python**     | `urllib.parse.quote()`              | `urllib.parse.unquote()`   | `urllib.parse.urlencode()`       |
| **Ruby**       | `URI.encode_www_form_component()`   | -                          | `URI.encode_www_form()`          |
| **Rust**       | `form_urlencoded::byte_serialize()` | -                          | `url::Url`                       |

## Endpoint-Specific Parameters

### Functions Endpoint

```bash
GET /v2/functions?
    status=active&                       # Filter by status
    name=user&                           # Partial name match
    enabled=true&                        # Boolean filter
    createdAfter=2025-08-01T00:00:00Z&  # Time filter
    cursor=abc123&                       # Pagination cursor
    limit=50                             # Page size
```

### Runs Endpoint

```bash
GET /v2/runs?
    status=running,completed&            # Multiple statuses
    functionId=user-signup&             # Specific function
    startedAfter=2025-08-01T00:00:00Z&  # Time range
    hasErrors=false&                    # Boolean filter
    cursor=def456&                       # Pagination
    limit=100
```

### Events Endpoint

```bash
GET /v2/events?
    name=user.signup&                    # Event name
    receivedAfter=2025-08-01T00:00:00Z& # Time filter
    cursor=ghi789&                        # Pagination
    limit=200
```

## Parameter Validation

### Type Validation Errors

```bash
GET /v2/functions?limit=invalid

HTTP/1.1 400 Bad Request
{
  "error": "Invalid query parameter: limit must be an integer between 1 and 250"
}
```

### Date Format Errors

```bash
GET /v2/runs?started_after=invalid-date

HTTP/1.1 400 Bad Request
{
  "error": "Invalid date format: expected ISO 8601 format (YYYY-MM-DDTHH:MM:SSZ)"
}
```

---

# Content Types

## JSON-First Strategy

### Primary Content Type

```
Content-Type: application/json; charset=utf-8
Accept: application/json
```

### Request Examples

```bash
# Function creation
POST /v2/functions
Content-Type: application/json
{
  "functionId": "user-signup",
  "name": "User Signup Handler",
  "description": "Handles new user registrations"
}

# Function invocation
POST /v2/functions/user-signup/invoke
Content-Type: application/json
{
  "data": {
    "userId": "user_123",
    "email": "user@example.com"
  }
}
```

### Response Examples

```bash
# Successful single resource response
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
{
  "data": {
    "id": "abc123",
    "functionId": "user-signup",
    "name": "User Signup Handler"
  },
  "metadata": {
    "fetchedAt": "2025-08-11T10:30:00Z",
    "cachedUntil": "2025-08-11T10:35:00Z"
  }
}

# Error response
HTTP/1.1 404 Not Found
Content-Type: application/json; charset=utf-8
{
  "error": "Function 'user-signup' not found"
}
```

### Content Type Validation

```bash
# Missing Content-Type
POST /v2/functions
{
  "functionId": "user-signup"
}

HTTP/1.1 400 Bad Request
{
  "error": "Content-Type header is required for request body"
}

# Incorrect Content-Type
POST /v2/functions
Content-Type: text/plain
{
  "functionId": "user-signup"
}

HTTP/1.1 415 Unsupported Media Type
{
  "error": "Unsupported Content-Type: expected application/json"
}
```

## Character Encoding

All text content uses UTF-8:

```
Content-Type: application/json; charset=utf-8
```

Unicode support for international content:

```json
{
  "functionId": "user-signup",
  "name": "ダーウィン",
  "description": "Hândles signüps with émojis 🚀"
}
```

---

# ID Formats

## Resource-Specific ID Strategy

Different resource types use different ID formats optimized for their specific use cases and requirements.

## ID Format by Resource Type

### Events and Runs: ULIDs

Events and runs use ULIDs for sortable, time-ordered identifiers:

```json
{
  "eventId": "01HE8AM9DPK9N37V1RKY1DNQF5", // Event ULID
  "runId": "01HP1ZX8M3NG9VP6QN0XK7J4CZ" // Run ULID
}
```

### Functions: App ID + Function ID

Functions use a composite identifier combining the environment slug and user-provided function identifier:

```json
{
  "functionId": "app-slug-function-slug" // Composite ID
}
```

**Format**: `{app-slug}-{function-slug}`

### Everything Else: UUIDs

All other resources use standard UUIDs:

```json
{
  "{type}Id": "550e8400-e29b-41d4-a716-446655440000" // Standard UUID v4
}
```

## Benefits by Resource Type

### ULIDs (Events & Runs)

- Time-ordered queries perform well
- Natural sorting for chronological data
- Efficient pagination

### Composite IDs (Functions)

- Clear namespace boundaries
- Human-readable function names
- User generated / improved usability
- Easy app-level operations
- _NOTE: Need to be URL encoded when included in path_

### UUIDs (Everything Else)

- Simple, standard approach
- No special handling required
- Universal compatibility

---

# Date/Time Formats

## RFC 3339 Standard (UTC)

### Primary Format

```
YYYY-MM-DDTHH:MM:SSZ              # Standard format
YYYY-MM-DDTHH:MM:SS.sssZ          # With milliseconds (for precision)

```

### Examples

```json
{
  "createdAt": "2025-08-11T10:30:00Z",
  "updatedAt": "2025-08-11T10:35:00Z",
  "startedAt": "2025-08-11T10:30:00.123Z",     # With milliseconds
  "completedAt": "2025-08-11T10:30:05.456Z"
}

```

### Field Naming Conventions

```json
{
  "createdAt": "2025-08-11T10:30:00Z",        # Resource creation
  "updatedAt": "2025-08-11T10:35:00Z",        # Last modification
  "startedAt": "2025-08-11T10:30:00Z",        # Process start
  "completedAt": "2025-08-11T10:30:05Z",      # Process completion
  "receivedAt": "2025-08-11T10:30:00Z"        # Event reception
}

```

### Duration Fields

```json
{
  "durationMs": 5333,                         # Duration in milliseconds
  "timeoutSeconds": 300                       # Timeout in seconds
}

```

### Query Parameter Usage

```bash
# Time range filtering
GET /v2/runs?startedAfter=2025-08-01T00:00:00Z
GET /v2/runs?completedBefore=2025-08-11T23:59:59Z

```

### Timezone Requirements

**API Response Format**: All timestamps returned by the API are in UTC and use the `Z` suffix:

```json
// ✅ API responses always use UTC
{
  "createdAt": "2025-08-11T10:30:00Z",
  "updatedAt": "2025-08-11T10:35:00.123Z"
}
```

**Request Acceptance**: The API accepts timestamps in any valid RFC 3339 format and converts them to UTC internally:

```json
// ✅ Accepted: UTC timezone specified
{
  "scheduledAt": "2025-08-11T10:30:00Z"
}

// ✅ Accepted: Offset timezone specified (converted to UTC internally)
{
  "scheduledAt": "2025-08-11T10:30:00+05:00"
}

// ❌ Invalid: No timezone information
{
  "scheduledAt": "2025-08-11T10:30:00"
}

```

### Precision Support

The API supports varying levels of precision:

```json
{
  "startedAt": "2025-08-11T10:30:00Z",        # Second precision
  "completedAt": "2025-08-11T10:30:00.123Z",  # Millisecond precision
  "processedAt": "2025-08-11T10:30:00.123456Z" # Microsecond precision
}

```

## Benefits

- **Internet Standard**: RFC 3339 designed for network protocols
- **Developer Familiar**: Native support in all modern languages
- **Timezone Clear**: Explicit timezone eliminates confusion
- **Sortable**: Lexicographic sorting works correctly
- **Machine Readable**: Direct parsing without format specification
- **Interoperable**: Consistent format across all systems and APIs

---

# Boolean Representation

## Standard JSON Booleans

```json
{
  "enabled": true,
  "isActive": false,
  "hasErrors": true,
  "canRetry": false
}
```

### Query Parameters

```bash
# Boolean filters (case insensitive)
GET /v2/functions?enabled=true
GET /v2/runs?has_errors=false
```

### Naming Patterns

```json
{
  "enabled": true,           # Simple state
  "isActive": true,          # "is" prefix for state
  "hasErrors": false,        # "has" prefix for presence
  "canRetry": true           # "can" prefix for capability
}
```

---

# Nullable and Optional Fields

## Clear Request Semantics: Required, Nullable, and Optional

### Field Types and Behavior

### Required Fields

- **Must be present** for CREATE (POST) and full UPDATE (PUT)
- **Optional** for partial UPDATE (PATCH)
- **Never nullable** - always have a value

### Nullable Fields

- **Can accept `null`** as an explicit value
- **Get default value** when omitted from request
- **Only set to `null`** when explicitly provided (unless null is the default value)

### Optional Fields

- **Can be omitted** from requests
- **Get default value** when omitted
- **Only set** when explicitly provided in request

### Request Examples

### Function Creation (POST)

```json
*// ✅ Valid: All required fields present*
POST /v2/functions
{
  "functionId": "user-signup",     *// Required*
  "name": "User Signup Handler",   *// Required*
  "description": null,             *// Nullable: explicitly set to null*
  "timeout": 60                    *// Optional: explicitly set // retries omitted               // Optional: will use default (3)*
}

*// ❌ Invalid: Missing required field*
POST /v2/functions
{
  "functionId": "user-signup"
  *// name missing - required field*
}

HTTP/1.1 400 Bad Request
{
  "error": "Field 'name' is required"
}
```

### Function Full Update (PUT)

```json
*// ✅ Valid: All required fields present*
PUT /v2/functions/user-signup
{
  "functionId": "user-signup",      *// Required*
  "name": "Updated Handler",        *// Required*
  "description": "New description", *// Nullable: set to value*
  "timeout": 120                    *// Optional: set to value // retries omitted               // Optional: will reset to default*
}
```

### Function Partial Update (PATCH)

```json
*// ✅ Valid: Only update specific fields*
PATCH /v2/functions/user-signup
{
  "description": null,             *// Nullable: explicitly clear*
  "timeout": 90                    *// Optional: update value// name omitted                  // Required field can be omitted in PATCH// retries omitted               // Optional: keeps current value*
}
```

## Response Behavior

### Always Include in Responses

- **All configured fields** - whether explicitly set or using defaults
- **Field values reflect current state** - explicit values or computed defaults

### Example Response with Defaults

```json
{
  "id": "123",
  "functionId": "user-signup",
  "name": "User Signup Handler",
  "description": null,             *// User explicitly set to null*
  "timeout": 60,                   *// User configured*
  "retries": 3,                    *// Using default value*
  "concurrency": 10,               *// Using default value*
  "priority": "normal",            *// Using default value*
  "enabled": true,                 *// Using default value*
  "createdAt": "2025-08-11T10:30:00Z",
  "updatedAt": "2025-08-11T10:30:00Z"
}
```

### Benefits

- **Clear Intent**: Explicit `null` vs omitted have different meanings
- **Predictable Defaults**: Omitted fields always get defaults
- **Flexible Updates**: PATCH allows partial updates without affecting other fields
- **Type Safety**: Required fields can never be `null`
- **API Evolution**: Easy to add new optional/nullable fields without breaking changes

---

# Enum Formats

## Consistent SCREAMING_SNAKE_CASE Enums

### Format Rules

- SCREAMING_SNAKE_CASE for consistency
- **Human readable** (no magic numbers)
- **Case insensitive** acceptance
- **Error on unknown** values

### Examples

```json
{
  "status": "COMPLETED", // Preferred format
  "priority": "HIGH",
  "type": "USER_EVENT"
}
```

### Accepted Variations

```bash
# All accepted (normalized to uppercase)
"status": "COMPLETED"      # Preferred
"status": "Completed"      # Accepted
"status": "completed"      # Accepted
```

### Common Enums

### Run Status

```
RUNNING, COMPLETED, FAILED, CANCELLED, QUEUED
```

### Function Status

```
ACTIVE, PAUSED, ARCHIVED, DISABLED
```

### Event Types

```
USER_EVENT, SYSTEM_EVENT, WEBHOOK_EVENT
```

### Query Parameter Usage

```bash
# Multiple enum values
GET /v2/runs?status=COMPLETED,FAILED
GET /v2/functions?status=ACTIVE,PAUSED
```

### Validation Errors

```bash
POST /v2/functions
{
  "status": "INVALID_STATUS"
}

HTTP/1.1 400 Bad Request
{
  "error": "Invalid status: must be one of [ACTIVE, PAUSED, ARCHIVED, DISABLED]"
}
```

---

# Array Handling

## Always Pass Arrays (Never Null)

### Array Rules

- **Use empty array** for no items: `[]`
- **Never null** - always use empty array
- **Omitted** when not configured

### Examples

```json
// ✅ Correct
{
  "tags": [],                    # Empty array
  "runs": [{"id": "123"}],       # Array with items
  // labels omitted              # Not configured
  "steps": []                    # No steps yet
}

// ❌ Never do this
{
  "tags": null,                  # Invalid - use [] instead
  "runs": null                   # Invalid - use [] instead
}
```

### Response Patterns

### Empty Collections

```json
{
  "data": [],
  "page": {
    "hasMore": false,
    "limit": 50
  }
}
```

### Populated Collections

```json
{
  "data": [{ "id": "item_1" }, { "id": "item_2" }],
  "page": {
    "cursor": "item_2",
    "hasMore": true,
    "limit": 50
  }
}
```

---

# Number Formats

## Appropriate Precision and Clear Units

### Integer Usage

```json
{
  "timeout": 30,                 # Seconds (integer)
  "retries": 3,                  # Count (integer)
  "concurrency": 10,             # Limit (integer)
  "position": 42,                # Queue position (integer)
  "durationMs": 1247             # Milliseconds (integer)
}
```

### Float Usage

```json
{
  "progress": 0.75,              # 0.0 to 1.0 (percentage as decimal)
  "successRate": 0.995,          # 0.0 to 1.0 (rate as decimal)
  "averageLatency": 123.45       # Milliseconds with precision
}
```

### Time and Duration

```json
{
  "timeoutSeconds": 30,          # Explicit unit in field name
  "durationMs": 1247,            # Milliseconds
  "retryDelaySeconds": 5         # Seconds
}
```

### Size Fields (Always Bytes)

```json
{
  "memoryLimit": 536870912,      # 512 MB in bytes
  "fileSize": 1048576,           # 1 MB in bytes
  "maxPayloadSize": 1000000      # ~1 MB in bytes
}
```

### Benefits

- **Precision Control**: Appropriate precision for each use case
- **Unit Clarity**: Field names indicate units
- **JavaScript Safe**: Within safe integer limits
- **Consistency**: Same units across similar fields

---

# Versioning

## Simple URL-Based Versioning

### Strategy: Major Versions Only + Non-Breaking Evolution

The API uses simple URL-based major versioning with a commitment to non-breaking changes within each major version.

```bash
/v2/functions    # Current stable version
/v3/functions    # Future major version (breaking changes only)

```

## Versioning Philosophy

### Stability First

- **v2 endpoints never break**: All changes within v2 are strictly non-breaking
- **Additive only**: New fields, new optional parameters, new endpoints
- **No removals**: Fields, parameters, or behavior are never removed in v2
- **Breaking changes = new major version**: Any breaking change requires /v3/

### Non-Breaking Changes (Allowed in v2)

✅ **Safe to add without version bump:**

- New optional fields in responses
- New optional query parameters
- New optional request body fields
- New endpoints under /v2/
- New enum values (where backwards compatible)
- Performance improvements
- Bug fixes that don't change behavior

```json
// Original response
{
  "data": {
    "id": "123",
    "functionId": "user-signup",
    "name": "User Signup Handler"
  }
}

// Enhanced response (non-breaking addition)
{
  "data": {
    "id": "123",
    "functionId": "user-signup",
    "name": "User Signup Handler",
    "tags": ["auth", "users"],        // ✅ New optional field
    "createdAt": "2025-08-11T10:30:00Z" // ✅ New optional field
  }
}

```

### Breaking Changes (Require v3)

❌ **These changes require a new major version:**

- Removing fields from responses
- Removing query parameters
- Changing field types or formats
- Changing required/optional status
- Renaming fields
- Changing URL structure
- Changing HTTP methods
- Removing enum values
- Changing error response formats

```bash
# Breaking change example - requires v3
/v2/functions/{functionId}     # v2 uses functionId
/v3/functions/{id}             # v3 uses id (breaking change)

```

## Client Expectations

### Robust Client Design

Clients should be designed to handle non-breaking additions gracefully:

```jsx
// ✅ Good: Ignore unknown fields
const { id, functionId, name } = response.data;
// New fields like 'tags' are safely ignored

// ✅ Good: Handle new optional parameters
const params = {
  limit: 50,
  // Future optional parameters won't break this
};
```

## Benefits

- **Simple**: No complex versioning schemes or headers
- **Predictable**: URL clearly indicates version and capabilities
- **Stable**: v2 clients never break due to API changes
- **gRPC Compatible**: Works seamlessly with gRPC-gateway
- **Developer Friendly**: Easy to understand and implement
- **Future Proof**: Clear path for breaking changes via v3

---

# Authentication

## Signing Keys Only (Launch Strategy)

### Single Authentication Method

```bash
# All requests use signing keyAuthorization: Bearer signkey-prod-abc123def456...
X-Inngest-Env: production
```

### Key Format

Authentication middleware will accept signing keys with prefixes, unprefixed keys and hashed keys for compatibility with all existing signing keys.

```
signkey-{environment}-{key}
{key} *unprefixed
{hashed_key}*

Examples:
signkey-prod-abc123def456...
signkey-staging-def456ghi789...
signkey-dev-hij012klm345...
```

### Request Format

```bash
GET /v2/functions
Authorization: Bearer signkey-prod-abc123def456...
X-Inngest-Env: production
```

### Environment Scoping

Environment is embedded in the signing key and validated against the `X-Inngest-Env` header

### Future API Key Support

API keys can be added later without breaking changes:

```
# Future: Both supported
Authorization: Bearer signkey-prod-abc123...  # Existing
Authorization: Bearer apikey-prod-xyz789...   # Future
```

---

# Rate Limiting

## Leaky Bucket Algorithm Implementation

The API implements rate limiting using the **leaky bucket algorithm** via the [throttled/throttled](https://github.com/throttled/throttled) library. This approach provides smooth, consistent request processing while preventing traffic bursts from overwhelming the system.

## Rate Limit Enforcement

### Standard Rate Limits

### Per Key (Global)

- **Limit**: 1,000 requests per hour
- **Burst**: 50 additional requests
- **Window**: Rolling 1-hour period

### Per Endpoint Category

**Read Operations** (GET endpoints)

- **Limit**: 500 requests per hour per API key
- **Burst**: 25 additional requests

**Write Operations** (POST, PUT, PATCH, DELETE)

- **Limit**: 200 requests per hour per API key
- **Burst**: 10 additional requests

**High-Volume Endpoints** (Events, Invocations)

- **Limit**: 2,000 requests per hour per API key
- **Burst**: 100 additional requests

### Rate Limit Grouping

Rate limits are applied based on:

```
{key}:{endpoint_category}

```

Example groupings:

- `signkey-prod-abc123:read` - All GET operations
- `signkey-prod-abc123:write` - All POST/PUT/PATCH/DELETE operations
- `signkey-prod-abc123:events` - Event-related endpoints

## Response Headers

All API responses include rate limit information via standard headers:

### Successful Requests (200-299)

```
HTTP/1.1 200 OK
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 847
X-RateLimit-Reset: 1692187200

```

### Rate Limited Requests (429)

```
HTTP/1.1 429 Too Many Requests
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1692187200
Retry-After: 60

{
  "errors": [
    {
      "code": "rate_limit_exceeded",
      "message": "Rate limit exceeded. Try again in 60 seconds"
    }
  ]
}

```

### Header Definitions

| Header                  | Description                                   | Format                              |
| ----------------------- | --------------------------------------------- | ----------------------------------- |
| `X-RateLimit-Limit`     | Maximum requests allowed in current window    | Integer (e.g., `1000`)              |
| `X-RateLimit-Remaining` | Requests remaining in current window          | Integer (e.g., `847`)               |
| `X-RateLimit-Reset`     | Unix timestamp when current window resets     | Unix timestamp (e.g., `1692187200`) |
| `Retry-After`           | Seconds until next request allowed (429 only) | Integer seconds (e.g., `60`)        |

## Implementation Details

### Leaky Bucket Configuration

```go
// Example throttled configuration
quota := throttled.RateQuota{
    MaxRate:  throttled.PerHour(1000),  // 1000 requests per hour
    MaxBurst: 50,                       // Allow burst of 50 additional requests
}

rateLimiter, err := throttled.NewGCRARateLimiter(store, quota)

```

### Key Generation Strategy

Rate limiting keys are generated based on:

- **Key**: Extracted from `Authorization` header
- **Endpoint Category**: Determined by HTTP method and path pattern

```
Key Format: {api_key}:{category}
Examples:
- signkey-prod-abc123:read
- signkey-dev-xyz789:write
- signkey-prod-abc123:events
```

## Endpoint Categories

### Read Operations (`read`)

```bash
GET /v2/functions
GET /v2/runs/{runId}
GET /v2/events/{eventId}

```

### Write Operations (`write`)

```bash
POST /v2/functions
PUT /v2/functions/{functionId}
PATCH /v2/functions/{functionId}
DELETE /v2/functions/{functionId}

```

### High-Volume Operations (`events`)

```bash
POST /v2/events
POST /v2/functions/{functionId}/invoke
POST /v2/runs/{runId}/replay

```

## Error Responses

### Multiple Rate Limits Exceeded

```json
{
  "errors": [
    {
      "code": "global_rate_limit_exceeded",
      "message": "Global rate limit exceeded"
    },
    {
      "code": "endpoint_rate_limit_exceeded",
      "message": "Write operations rate limit exceeded"
    }
  ]
}
```

## Configuration Management

Rate limits are configurable via environment variables:

```bash
# Global limits
RATE_LIMIT_GLOBAL_HOURLY=1000
RATE_LIMIT_GLOBAL_BURST=50

# Category-specific limits
RATE_LIMIT_READ_HOURLY=500
RATE_LIMIT_WRITE_HOURLY=200
RATE_LIMIT_EVENTS_HOURLY=2000

# Storage backend
RATE_LIMIT_STORE_TYPE=redis
RATE_LIMIT_REDIS_URL=redis://localhost:6379

```

---

# Caching - in progress

_NOTES_

I think we can just write something like we have in `monorepo` for `DBCache`, except it is used for APIs.
this is currently backed by redis/valkey cluster right now, which seems like a common approach.

couple things we need to make sure it works that doesn’t right now:

- respect `Cache-Control` header
- writes will need to bust cache accordingly

- Ask @Riadh Daghmoura about current implementation

---

# Observability

## Essential Metrics for Launch

### Golden Signals

- Latency
- Traffic
- Errors
- Saturation

### Business Metrics

### API Adoption

- Daily active keys
- Requests per day
- V2 to V1 ratio
- New integrations

### Function Adoption

- invocations
- duration
- errors

### Rate Limit Monitoring

- **Requests per second** by endpoint category
- **Rate limit hit rate** (429 responses / total requests)
- **Average response time** when rate limited
- **Top rate-limited API keys**

### SLOs for Launch

- availability
- latency
- error rate

### Logging for Launch

**Requests**

```json
{
  "timestamp": "2025-08-11T10:30:00Z",
  "level": "info",
  "msg": "api_request",
  "method": "POST",
  "path": "/v2/functions/user-signup/invoke",
  "status": 200,
  "duration_ms": 145,
  "request_id": "abc123",
  "environment": "production",
  "function_id": "user-signup"
}
```

**Errors**

```json
{
  "timestamp": "2025-08-11T10:30:00Z",
  "level": "error",
  "msg": "function_invoke_failed",
  "error": "timeout waiting for function response",
  "request_id": "def456",
  "function_id": "slow-function",
  "environment": "production",
  "timeout_ms": 30000
}
```

**Authn**

```json
{
  "timestamp": "2025-08-11T10:30:00Z",
  "level": "warn",
  "msg": "auth_failed",
  "reason": "invalid_signing_key",
  "request_id": "ghi789",
  "ip_address": "192.168.1.100"
}
```
