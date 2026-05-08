# api-docs.inngest.com

This is an API docs site using TanStack Start and Fumadocs which auto-generates API pages based on generated OpenAPI specs from the codebase.

## Generating OpenAPI specs

From the root directory of this repo:

```
make docs
```

## Development

From this directory, install deps and start the dev server to preview:

```
pnpm install
pnpm run dev
```

Generate the docs pages from the generated OpenAPI files: (generated w/ `make docs`)

```
pnpm generate
```

## Release

Requires the Vercel API and this directory linked to the `api-docs.inngest.com` project.

```
vercel build
vercel deploy --prebuilt
```
