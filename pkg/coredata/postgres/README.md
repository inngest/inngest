# Postgres DataStore

A PostgreSQL-backed implementation for the system data store.

## Configuration

You can pass the necessary configuration in the form of a PostgreSQL [Connection URI](https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING). For example:

```cue
config.#Config & {
  datastore: {
		service: {
			backend: "postgres"
			URI:     "postgres://user:password@localhost:5433/postgres?sslmode=disable"
		}
	}
  // ...
}
```

## Running Migrations

Automated migrations are not yet implemented, but you can run the migrations manually in order as found in the `migrations` directory.
