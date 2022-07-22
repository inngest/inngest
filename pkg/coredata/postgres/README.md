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

## Indexes and optimizations

The included database indexes are meant for a baseline of typical usage, but with any system,
performance tuning will be necessary as you scale your deployment of Inngest. Areas to look into
are the cardinality of the data in `function_triggers.event_name` as well as how many
`function_versions` that share the same event trigger. Potentially a `HASH` index might be right
for you if your system does not leverage wildcard `event_name` matching. Caching can also be
a huge potential for improvement.

Chat with us on [Discord](https://www.inngest.com/discord) or
[contact us](https://www.inngest.com/contact) if you'd like to talk performance tuning.
