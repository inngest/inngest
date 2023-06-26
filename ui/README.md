# Dev Server UI

## Development

Start the CLI using the `dev` command, for example:

```sh
go run ./cmd/main.go dev -u http://localhost:3000/api/inngest
```

Then run the UI in dev mode. This will run Next.js and GraphQL codegen concurrently:

```sh
pnpm dev
```

## GraphQL Codegen

Edit or add your queries within `coreapi.ts` and the GraphQL Codegen should automatically create a hook to use in `store/generated.ts`.

To force running the codegen run `pnpm dev:codegen`.
