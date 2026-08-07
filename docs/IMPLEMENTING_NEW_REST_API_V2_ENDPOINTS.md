# Implementing New REST API v2 Endpoints

This document provides a comprehensive guide for implementing new endpoints in Inngest's REST API v2.

## Overview

The REST API v2 uses gRPC-Gateway to generate REST endpoints from Protocol Buffer definitions. The implementation follows a schema-first approach where the API specification is defined in protobuf files, and the actual service logic is implemented in Go.

## Implementation Steps

### 1. Define the Endpoint in Protobuf

**File:** `proto/api/v2/service.proto`

Add your new RPC method to the `V2` service definition with the appropriate HTTP annotations:

```protobuf
service V2 {
  // ... existing endpoints ...

  rpc YourNewEndpoint(YourRequest) returns (YourResponse) {
    option (google.api.http) = {
      // For GET endpoints
      get: "/your-resource"

      // For POST endpoints with body
      post: "/your-resource"
      body: "*"

      // For endpoints with path parameters
      get: "/your-resource/{id}"
    };

    // Required: the API key permission this endpoint needs. The grant must be
    // declared once via `option (grant_definition)` at the top of
    // service.proto; reuse an existing one unless the resource is genuinely
    // new. Action is READ for anything that only fetches, WRITE for anything
    // that mutates or has side effects.
    option (authz) = {
      grant: "api:your-resource"
      action: AUTHZ_ACTION_READ
    };
    // ...or, for a deliberately public endpoint, exempt it and say why:
    //   // Exempt: <reason>
    //   option (authz) = { exempt: true };

    // OpenAPI documentation
    option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_operation) = {
      summary: "Brief description of endpoint"
      description: "Detailed description of what this endpoint does"

      // Security requirement (for authenticated endpoints)
      security: {
        security_requirement: {
          key: "BearerAuth"
          value: {}
        }
      }

      // Define response schemas
      responses: {
        key: "200"
        value: {
          description: "Success response description"
          schema: {
            json_schema: {
              ref: "#/definitions/v2YourResponse"
            }
          }
        }
      }
      responses: {
        key: "400"
        value: {
          description: "Bad Request - validation errors"
          schema: {
            json_schema: {
              ref: "#/definitions/v2ErrorResponse"
            }
          }
        }
      }
      responses: {
        key: "401"
        value: {
          description: "Unauthorized - authentication required"
          schema: {
            json_schema: {
              ref: "#/definitions/v2ErrorResponse"
            }
          }
        }
      }
      responses: {
        key: "500"
        value: {
          description: "Internal Server Error"
          schema: {
            json_schema: {
              ref: "#/definitions/v2ErrorResponse"
            }
          }
        }
      }
    };
  }
}
```

#### Define Request and Response Messages

Add the corresponding message definitions at the end of the file:

```protobuf
message YourRequest {
  // For path parameters
  string id = 1;

  // For query parameters (optional fields)
  optional string cursor = 2 [
    (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_field) = {
      description: "Pagination cursor from previous response"
    }
  ];

  // For request body fields
  string name = 3;
  optional string description = 4;
}

message YourResponse {
  YourData data = 1;
  ResponseMetadata metadata = 2;
  // Include Page for paginated responses
  Page page = 3;
}

message YourData {
  string id = 1;
  string name = 2;
  optional string description = 3;
  google.protobuf.Timestamp createdAt = 4;
  google.protobuf.Timestamp updatedAt = 5;
}
```

When you are done defining your endpoint, request and response, run:

```sh
make protobuf
```

#### Authorization Grants

Every rpc must carry an `(authz)` option: either a grant or `exempt: true`.
`TestGrantCatalogIsValid` fails the build otherwise, which is deliberate: an
endpoint with no annotation would otherwise ship unprotected.

Grants are declared once at the top of `service.proto`:

```protobuf
option (grant_definition) = {
  name: "api:run"                      // ^api:[a-z]+$, resource only, no action
  description: "Function runs and traces, including rerunning and cancelling"
  category: "Apps, Functions & Runs"   // one of the four minting-UI categories
};
```

A grant covers many endpoints, which is why the description lives on the
declaration rather than on each method. See `proto/api/v2/options.proto`.

After changing grants or annotations, run `make grant-catalog` and review the
diff in `pkg/api/v2/apiv2base/testdata/grant_catalog.json`. That file is the
committed record of every permission the API offers, and CI checks it is current.

If you add `additional_bindings` to a method, both paths inherit the same grant
automatically, but confirm it in the catalog diff, since a missed binding is a
route that enforcement would not cover.

#### Key Conventions

- **Required vs Optional**: Use `optional` for non-required or nullable fields
- **Timestamps**: Use `google.protobuf.Timestamp` for all date/time fields
- **IDs**: Use appropriate ID format (UUIDs, ULIDs, or composite IDs)
- **Metadata**: Always include `ResponseMetadata` in responses

### 2. Implement the Handler Function

**File:** `pkg/api/v2/service.go`

Add your handler function to the API v2 service:

```go
func (s *Service) YourNewEndpoint(ctx context.Context, req *apiv2.YourRequest) (*apiv2.YourResponse, error) {

    // Additional validation (optional)
    if req.Limit != nil {
        if *req.Limit < 1 {
            return nil, NewError(http.StatusBadRequest, ErrorInvalidFieldFormat, "Limit must be at least 1")
        }
        if *req.Limit > 100 {
            return nil, NewError(http.StatusBadRequest, ErrorInvalidFieldFormat, "Limit cannot exceed 100")
        }
    }

    // Extract context data (if needed)
    envName := GetInngestEnvHeader(ctx)

    // 4. Business logic implementation
    // ... your implementation here ...

    // 5. Error handling
    if err != nil {
        return nil, NewError(http.StatusInternalServerError, ErrorInternalError, "Failed to process request")
    }

    // Build and return response
    return &apiv2.YourResponse{
        Data: &apiv2.YourData{
            Id:          "generated-id",
            Name:        req.Name,
            Description: req.Description,
            CreatedAt:   timestamppb.New(time.Now()),
            UpdatedAt:   timestamppb.New(time.Now()),
        },
        Metadata: &apiv2.ResponseMetadata{
            FetchedAt:   timestamppb.New(time.Now()),
            CachedUntil: nil,
        },
        // Include pagination for list endpoints
        Page: &apiv2.Page{
            Cursor:  nil,
            HasMore: false,
            Limit:   20,
        },
    }, nil
}
```

#### Handler Best Practices

- **Validation First**: Basic validation is automatic but you'll need to validate anything outside the request body and enforce any constraints
- **Error Handling**: Use the established error patterns.(`NewError`, `NewErrors`)
- **Context Usage**: Extract headers and authentication info from context
- **Response Structure**: Follow the consistent response envelope pattern
- **Timestamps**: Make sure to use RFC 3339 Standard formatting for timestamps

#### Common Error Patterns

```go
// Single error
return nil, NewError(http.StatusBadRequest, ErrorMissingField, "Field is required")

// Multiple errors
return nil, NewErrors(http.StatusBadRequest,
    ErrorItem{Code: ErrorMissingField, Message: "Name is required"},
    ErrorItem{Code: ErrorInvalidFormat, Message: "Email format is invalid"},
)

// Not implemented (for OSS features)
return nil, NewError(http.StatusNotImplemented, ErrorNotImplemented, "Feature not implemented in OSS")
```

### 3. Generate Documentation Stubs

Run the following command to generate example stubs:

```bash
make docs
```

This command will:

- Generate OpenAPI documentation from your protobuf definitions
- Create stub entries in `docs/api_v2_examples.json` for any new endpoints
- Update the API documentation

### 4. Add Real-World Examples

**File:** `docs/api_v2_examples.json`

After running `make docs`, you'll find TODO stubs for your new endpoint. Replace these with realistic examples:

```json
{
  "/your-resource": {
    "get": {
      "200": {
        "data": {
          "id": "01HP1ZX8M3NG9VP6QN0XK7J4CY",
          "name": "Example Resource",
          "description": "This is an example resource for demonstration",
          "createdAt": "2024-01-20T14:22:33Z",
          "updatedAt": "2024-01-20T14:22:33Z"
        },
        "metadata": {
          "fetchedAt": "2024-01-20T14:22:33Z",
          "cachedUntil": null
        }
      },
      "400": {
        "errors": [
          {
            "code": "validation_error",
            "message": "Invalid request parameters"
          }
        ]
      },
      "401": "Authentication failed",
      "500": {
        "errors": [
          {
            "code": "internal_server_error",
            "message": "An unexpected error occurred"
          }
        ]
      }
    },
    "post": {
      "201": {
        "data": {
          "id": "01HP1ZX8M3NG9VP6QN0XK7J4CY",
          "name": "New Resource",
          "description": "Newly created resource",
          "createdAt": "2024-01-20T14:22:33Z",
          "updatedAt": "2024-01-20T14:22:33Z"
        },
        "metadata": {
          "fetchedAt": "2024-01-20T14:22:33Z",
          "cachedUntil": null
        }
      },
      "400": {
        "errors": [
          {
            "code": "missing_field",
            "message": "Name is required"
          }
        ]
      }
    }
  }
}
```

#### Example Guidelines

- **Realistic Data**: Use believable IDs, timestamps, and content
- **Consistent Formatting**: Follow existing timestamp and ID formats
- **Complete Coverage**: Include examples for all documented response codes
- **Error Examples**: Show realistic error scenarios with proper error codes

### 5. Use in Monorepo

Once your change is pushed to main, your endpoint and data types will be available in monorepo. You will just need to implement the service handler function.
