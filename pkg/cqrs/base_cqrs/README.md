# base_cqrs

`pkg/cqrs/base_cqrs` is the legacy home of the composite CQRS manager
and history implementations while the tree is being moved to
the new package boundaries defined in `docs/plans/005-remove-base-cqrs.org`.

Current responsibilities still living here:

- the current `NewCQRS()` composite manager
- the current history reader and writer implementations

Planned destinations:

- `pkg/cqrs/manager` for the composite manager
- `pkg/cqrs` for caller-facing history contracts and payload types

The old Postgres normalization layer has been removed. Generated query code now
lives with the dialect adapters under `pkg/db/sqlite/sqlc` and
`pkg/db/postgres/sqlc`.
