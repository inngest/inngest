# Dev Server UI

## Development

In the root directory of this repository, start the CLI using the `dev` command. For example:

```sh
go run ./cmd/main.go dev -u http://localhost:3000/api/inngest
```

Then in this directory, run the UI in dev mode. This will run Next.js and GraphQL codegen concurrently:

```sh
pnpm dev
```

## GraphQL Codegen

Edit or add your queries within `coreapi.ts` and the GraphQL Codegen should automatically create a hook to use in `store/generated.ts`.

To force running the codegen run `pnpm dev:codegen`.
