# OpenAPI Documentation Workflow

This document explains the automated OpenAPI documentation generation workflow for Inngest's REST API v2.

## Overview

The project generates comprehensive OpenAPI documentation from protobuf files. This includes:

1. **OpenAPI v2 generation** from protobuf files with gRPC-gateway HTTP annotations
2. **OpenAPI v3 conversion** with custom enhancements using the kin-openapi library
3. **Advanced features** including custom error responses, authentication, and multi-server configuration
4. **Public specification and endpoint page generation** with Fumadocs

## Generated Files

The documentation is generated in the following structure:

```
docs/
├── openapi/
│   ├── v2/          # OpenAPI 2.0 specs (generated from protobuf)
│   │   └── api/v2/service.swagger.json
│   └── v3/          # OpenAPI 3.0 specs (converted and enhanced)
│       └── api/v2/service.swagger.json
└── api-docs/
    ├── public/api-specs/ # Public OpenAPI assets
    └── content/docs/     # Hand-written and generated documentation pages
```

## Build Commands

### Primary Command

```bash
make docs
```

This installs the locked API docs dependencies and generates the intermediate
OpenAPI specifications, public OpenAPI assets, and endpoint pages.

### Intermediate OpenAPI only

```bash
make openapi
```

Go development and release builds do not generate API documentation.

### Cleaning Generated Files

```bash
make clean
```

## Current Implementation Features

### API Documentation Includes

- **Multiple servers**: Production (`https://api.inngest.com/v2`) and Development (`http://localhost:8288/api/v2`)
- **Bearer token authentication** with format examples
- **Custom error responses** with proper error array format per API specification
- **Detailed status codes**: 200, 201, 400, 401, 403, 409, 500 with descriptions
- **No default responses** - only explicitly defined status codes are included
- **Professional metadata**: Title "Inngest REST API v2", version "2.0.0"

### Enhanced Error Handling

The API follows the v2 specification with:
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

## Technical Implementation

### 1. OpenAPI v2 Generation

**Command:**
```bash
cd proto && protoc --proto_path=. --proto_path=third_party \
    --openapiv2_out=../docs/openapi/v2 \
    --openapiv2_opt=allow_delete_body=true \
    --openapiv2_opt=json_names_for_fields=false \
    api/v2/service.proto
```

### 2. Custom OpenAPI v3 Conversion

The custom converter (`tools/convert-openapi/`) provides:

- **Default response removal**: Eliminates unwanted `default` responses
- **Smart 200 response handling**: Preserves 200 for endpoints without custom success codes
- **Multi-server configuration**: Converts v2 basePath to v3 servers array
- **Error schema generation**: Ensures proper error response schemas

**Usage:**
```bash
go run ./tools/convert-openapi docs/openapi/v2 docs/openapi/v3
```

## Protobuf Configuration

### Service-Level Configuration

```proto
option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_swagger) = {
  info: {
    title: "Inngest REST API v2"
    version: "2.0.0"
    description: "REST API with enhanced developer experience"
  }
  host: "api.inngest.com"
  base_path: "/v2"
  schemes: HTTPS
  security_definitions: {
    security: {
      key: "BearerAuth"
      value: {
        type: TYPE_API_KEY
        in: IN_HEADER
        name: "Authorization"
        description: "Bearer token authentication"
      }
    }
  }
};
```

### Endpoint-Level Configuration

```proto
rpc CreateAccount(CreateAccountRequest) returns (CreateAccountResponse) {
  option (google.api.http) = {
    post: "/partner/accounts",
    body: "*"
  };
  option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_operation) = {
    security: {
      security_requirement: {
        key: "BearerAuth"
        value: {}
      }
    }
    responses: {
      key: "201"
      value: {
        description: "Account successfully created"
        schema: {
          json_schema: {
            ref: "#/definitions/v2CreateAccountResponse"
          }
        }
      }
    }
    // Additional status codes...
  };
};
```

## Error Response Schema

### Required Proto Messages

```proto
message Error {
  string code = 1;
  string message = 2;
}

message ErrorResponse {
  repeated Error errors = 1;  // Always an array
}

// Internal method to ensure schema generation
rpc _SchemaOnly(HealthRequest) returns (ErrorResponse);
```

## Dependencies

- `protoc-gen-openapiv2` (grpc-gateway)
- `github.com/getkin/kin-openapi/openapi2`
- `github.com/getkin/kin-openapi/openapi2conv`
- `github.com/getkin/kin-openapi/openapi3`

## Adding New Endpoints

1. **Define RPC method** with HTTP annotations:
   ```proto
   rpc MyMethod(MyRequest) returns (MyResponse) {
     option (google.api.http) = {
       post: "/my-endpoint"
     };
   }
   ```

2. **Add custom responses** (optional):
   ```proto
   option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_operation) = {
     responses: {
       key: "201"
       value: {
         description: "Created successfully"
       }
     }
   };
   ```

3. **Run documentation generation**:
   ```bash
   make docs
   ```

## Current API Endpoints

- `GET /v2/health` - System health check (public)
- `POST /v2/partner/accounts` - Create account (requires auth)

## Git Integration

- Intermediate OpenAPI v2 and v3 files are excluded from Git via `.gitignore`.
- Source protobuf files, examples, and conversion utilities are tracked.
- Public OpenAPI assets and generated endpoint pages are produced by
  `make docs`.

## Troubleshooting

### Missing Schemas
If error schemas are missing, ensure:
1. Error message definitions exist in proto file
2. Internal `_SchemaOnly` method references the error response
3. Schema references use correct `#/definitions/v2ErrorResponse` format

### Authentication Not Showing
Verify:
1. Security definitions are at service level
2. Security requirements are at operation level
3. Bearer token format is documented in descriptions
