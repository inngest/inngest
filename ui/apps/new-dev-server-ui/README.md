# Dev Server UI

## Development

In the root directory of this repository, start the CLI using the `dev` command. For example:

```sh
go run ./cmd dev -u http://localhost:3000/api/inngest
```

Then in this directory, run the UI in dev mode. This will run Tanstack Start and GraphQL codegen concurrently:

```sh
pnpm dev
```

Or, optionally in the root `ui` run:

```sh
pnpm pnpm dev:new-dev-server-ui
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
