# Constraint API - Protobuf Workflow

## Overview

The Constraint API package implements capacity management functionality with gRPC interfaces generated from Protocol Buffer definitions. This document describes the workflow for maintaining and updating the protobuf integration.

## Source of Truth

The **Go types in this package are the source of truth** for the Constraint API. The protobuf definitions in `proto/constraintapi/v1/service.proto` are mirrors of these Go types and should be kept in sync when changes are made.

### Key Go Types (Source of Truth):
- `CapacityManager` interface - defines all capacity management operations
- `ConstraintConfig` and related configuration types
- `ConstraintCapacityItem` and capacity types
- Request/Response types for all operations:
  - `CapacityCheckRequest/Response`
  - `CapacityLeaseRequest/Response` 
  - `CapacityExtendLeaseRequest/Response`
  - `CapacityCommitRequest/Response`
  - `CapacityRollbackRequest/Response`

## Protobuf Code Generation

To regenerate protobuf code after making changes to the `.proto` files:

```bash
make protobuf
```

This command should be run from the **root directory** of the project and will:
1. Generate Go code from protobuf definitions
2. Update the generated files in `proto/gen/constraintapi/v1/`
3. Include both message types and gRPC service definitions

## Conversion Logic

All conversion between Go types and protobuf types is handled in `convert.go`. This file contains:

### Enum Conversion Functions
- Bidirectional conversion for all scope and mode enums
- Handles mapping between Go enums and protobuf enum constants
- Includes proper fallback values for unspecified cases

### Type Conversion Functions
- **Configuration types**: `ConstraintConfig`, `RateLimitConfig`, `ConcurrencyConfig`, etc.
- **Capacity types**: `ConstraintCapacityItem`, `LeaseSource`
- **Request/Response types**: All CapacityManager operation types

### Data Type Handling
- UUID ↔ string conversion
- ULID ↔ string conversion
- `time.Time` ↔ `timestamppb.Timestamp`
- `time.Duration` ↔ `durationpb.Duration`
- Slice/array conversions for repeated fields
- Proper nil/null safety checks
- Error handling for parsing failures

## Workflow for Changes

When making changes to the Constraint API:

1. **Update Go types first** (source of truth)
2. **Update protobuf definitions** to mirror Go changes
3. **Run `make protobuf`** to regenerate protobuf code
4. **Update conversion functions** in `convert.go` if needed
5. **Test the changes** to ensure proper conversion

## Important Notes

- Always maintain backward compatibility when possible
- Ensure all enum values have proper mappings in conversion functions
- Add error handling for any new parsing operations (UUIDs, ULIDs, etc.)
- Keep protobuf field numbers stable to maintain wire compatibility
- Document any breaking changes in the protobuf interface