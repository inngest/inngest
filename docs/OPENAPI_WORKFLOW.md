# OpenAPI Documentation Workflow

This document explains the automated OpenAPI documentation generation workflow implemented for Inngest's gRPC services.

## Overview

The project now automatically generates OpenAPI documentation from protobuf files during the build process. This includes:

1. **OpenAPI v2 generation** from protobuf files with gRPC-gateway HTTP annotations
2. **OpenAPI v3 conversion** using the kin-openapi library
3. **Automatic integration** into the existing Makefile build process

## Generated Files

The documentation is generated in the following structure:

```
docs/
├── openapi/
│   ├── v2/          # OpenAPI 2.0 specs (generated from protobuf)
│   │   └── api/v2/service.swagger.json
│   └── v3/          # OpenAPI 3.0 specs (converted from v2)
│       └── api/v2/service.swagger.json
```

## Build Integration

### Automatic Generation

Documentation is generated automatically when running:

- `make` - Full build with documentation
- `make build` - Production build with documentation  
- `make dev` - Development build with documentation

### Manual Generation

To generate only documentation without building:

```bash
make docs-only
```

### Cleaning Generated Files

To clean generated documentation files:

```bash
make clean
```

## How It Works

### 1. OpenAPI v2 Generation

The workflow uses `protoc` with the `protoc-gen-openapiv2` plugin to generate OpenAPI v2 specifications from protobuf files that have gRPC-gateway HTTP annotations.

**Command:**
```bash
cd proto && protoc --proto_path=. --proto_path=third_party \
    --openapiv2_out=../docs/openapi/v2 \
    --openapiv2_opt=allow_delete_body=true \
    --openapiv2_opt=json_names_for_fields=false \
    api/v2/service.proto
```

### 2. OpenAPI v3 Conversion

A custom Go utility (`tools/convert-openapi/`) converts OpenAPI v2 specifications to OpenAPI v3 format using the `kin-openapi` library.

**Usage:**
```bash
go run ./tools/convert-openapi docs/openapi/v2 docs/openapi/v3
```

### 3. Dependencies

The following dependencies are used:

- `protoc-gen-openapiv2` (already installed via grpc-gateway)
- `github.com/getkin/kin-openapi` (Go module for v2->v3 conversion)

## Adding New Services

To add OpenAPI documentation for new gRPC services:

1. **Add HTTP annotations** to your protobuf service definitions:
   ```proto
   rpc MyMethod(MyRequest) returns (MyResponse) {
     option (google.api.http) = {
       get: "/my-endpoint"
     };
   }
   ```

2. **Update the Makefile** to include your new protobuf file in the `docs` target.

3. **Run documentation generation** to verify the output.

## Git Integration

- Generated documentation files are automatically excluded from git via `.gitignore`
- Only source protobuf files and the conversion utility are tracked
- Documentation is regenerated on each build to stay current

## Notes

- The workflow currently generates documentation only for services with gRPC-gateway HTTP annotations
- Services without HTTP annotations (like `connect/v1` and `debug/v1`) are not included in the OpenAPI documentation
- The conversion utility handles multiple protobuf files and preserves directory structure