# CQRS

This package is a storage-agnostic CQRS layer that uses SQLite types as the base
that all other storage layers must adhere to.

## Adding a query

To add a query, first create your new query in
`pkg/cqrs/base_cqrs/sqlc/sqlite/queries.sql`:
```sql
-- name: NewExampleQuery :one
SELECT * FROM apps WHERE id = ? LIMIT 1;
```

Now run `make queries` to generate glue code. Any errors that appear here will
likely be syntax errors that must be addressed.

Upon booting the server, you'll likely now receive an error stating that some DB
normalization layer does not implement the new function. All other storage
layers use SQLite as their I/O, so we must write custom code to transform the
I/O for that layer.

```
[nixos@nixos:~/repo/inngest/inngest]$ go run ./cmd start
# github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/postgres
pkg/cqrs/base_cqrs/sqlc/postgres/db_normalization.go:15:9: cannot use &NormalizedQueries{…} (value of type *NormalizedQueries) as "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/sqlite".Querier value in return statement: *NormalizedQueries does not implement "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/sqlite".Querier (missing method NewExampleQuery)
```

Head to `pkg/cqrs/base_cqrs/sqlc/postgres/db_normalization.go` and add the new
function:
```go
func (q NormalizedQueries) NewExampleQuery(ctx context.Context, id uuid.UUID) (*sqlc_sqlite.App, error) {
	return nil, fmt.Errorf("not implemented")
}
```

`q.db` is the glue code for the Postgres queries, which will likely not yet
contain `NewExampleQuery`, so let's go add it to
`pkg/cqrs/base_cqrs/sqlc/postgres/queries.sql`:
```sql
-- name: NewExampleQuery :one
SELECT * FROM apps WHERE id = $1 LIMIT 1;
```

Now we can implement our function:
```go
func (q NormalizedQueries) NewExampleQuery(ctx context.Context, id uuid.UUID) (*sqlc_sqlite.App, error) {
	app, err := q.db.NewExampleQuery(ctx, id)
	if err != nil {
		return nil, err
	}

	return app.ToSQLite()
}
```

For ease of use, `app.ToSQLite()` defined in
`pkg/cqrs/base_cqrs/sqlc/postgres/normalization.go` extends the generated glue
code for these common tasks.
