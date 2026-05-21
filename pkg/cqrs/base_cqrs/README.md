# base_cqrs

`pkg/cqrs/base_cqrs` is the legacy home of standalone history shims while the
tree is being moved to
the new package boundaries defined in `docs/plans/005-remove-base-cqrs.org`.

Current responsibilities still living here:

- the current history writer implementation
- a deprecated history reader constructor shim

Planned destinations:

- `pkg/cqrs/manager` for the composite manager and CQRS-backed history reads
- `pkg/cqrs` for caller-facing history contracts and payload types

The old Postgres normalization layer has been removed. Generated query code now
lives with the dialect adapters under `pkg/db/sqlite/sqlc` and
`pkg/db/postgres/sqlc`.
