# Dev Server UI

## Development

In the root directory of this repository, start the CLI using the `dev` command. For example:

```sh
go run ./cmd dev -u http://localhost:3000/api/inngest
```

By default, the dev server uses SQLite for persistence. To use PostgreSQL instead, you can either:

1. Use the `--postgres-uri` flag:

```sh
go run ./cmd dev -u http://localhost:3000/api/inngest --postgres-uri postgres://user:password@localhost:5432/inngest
```

2. Or set the `INNGEST_POSTGRES_URI` environment variable:

```sh
INNGEST_POSTGRES_URI="postgres://user:password@localhost:5432/inngest" go run ./cmd dev -u http://localhost:3000/api/inngest
```

When `--postgres-uri` or `INNGEST_POSTGRES_URI` is provided, the dev server will use PostgreSQL instead of SQLite. You can also configure PostgreSQL connection pool settings with:

- `--postgres-max-idle-conns` (default: 10)
- `--postgres-max-open-conns` (default: 100)
- `--postgres-conn-max-idle-time` (default: 5 minutes)
- `--postgres-conn-max-lifetime` (default: 30 minutes)

Then in this directory, run the UI in dev mode. This will run Tanstack Start and GraphQL codegen concurrently:

```sh
pnpm dev
```

Or, optionally in the root `ui` run:

```sh
pnpm pnpm dev:dev-server-ui
```

## Preview "production" builds

copy `.env.development` to `.env`

```sh
pnpm build
```

```sh
pnpm preview
```

## GraphQL Codegen

Edit or add your queries within `coreapi.ts` and the GraphQL Codegen should automatically create a hook to use in `store/generated.ts`.

To force running the codegen run `pnpm dev:codegen`.

## Feature flags

You can pass feature flags to the dev server locally like so:

```
INNGEST_FEATURE_FLAGS="step-over-debugger=true,some-other-flag=false" go run ./cmd dev
```

Then, in shared components you can use the share `useBooleanFlag` hook and the flag will be checked in the appropriate place in either the dev server or cloud:

```
const { value, isReady } = booleanFlag(
  'step-over-debugger',
  false
);
```

## Stack

This is a Tanstack Start application: https://tanstack.com/start/latest/docs/framework/react/overview
