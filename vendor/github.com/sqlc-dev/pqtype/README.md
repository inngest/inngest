[![Go Reference](https://pkg.go.dev/badge/github.com/sqlc-dev/pqtype.svg)](https://pkg.go.dev/github.com/sqlc-dev/pqtype)
[![go](https://github.com/sqlc-dev/pqtype/actions/workflows/ci.yml/badge.svg)](https://github.com/sqlc-dev/pqtype/actions/workflows/ci.yml)

# pqtype

pqtype implements Go types for PostgreSQL types when using the
[lib/pq](https://github.com/lib/pq) driver. 

## Compatibility

pqtype is tested against PostgreSQL 9.6 through 13 and Go 1.13 through 1.17.
While these types may work with other drivers, they are **only** tested against
the lib/pq driver.

## History

pqtype is a fork of [jackc/pgtype](https://github.com/jackc/pgtype) with all
the pgx-specific code removed. The `Status` field on types has been replaced
with a `Valid` boolean to mirror the standard library `sql.Null*` types.
