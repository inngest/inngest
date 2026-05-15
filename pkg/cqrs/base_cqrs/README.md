# base_cqrs

`pkg/cqrs/base_cqrs` is the legacy home of the composite CQRS manager
and history implementations while the tree is being moved to
the new package boundaries defined in `docs/plans/002-remove-base-cqrs.org`.

Current responsibilities still living here:

- the current `NewCQRS()` composite manager
- the current history reader and writer implementations

Planned destinations:

- `pkg/cqrs/manager` for the composite manager
- `pkg/cqrs` for caller-facing history contracts and payload types

The old Postgres normalization layer has been removed. The remaining generated
query code still lives under `sqlc/` only until the dialect-local move in phase
2.
