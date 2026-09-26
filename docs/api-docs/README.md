# api-docs.inngest.com

This is an API docs site using TanStack Start and Fumadocs which generates API
pages from OpenAPI specifications in the codebase.

## Generating API docs

Generate the public OpenAPI assets and endpoint pages from the root directory of
this repository:

```sh
make docs
```

Use `make openapi` when you only need the intermediate OpenAPI v2 and v3 files.

## Development

Generate the API docs first, then start the dev server from this directory:

```sh
pnpm run dev
```

## Release

Requires the Vercel API and this directory linked to the `api-docs.inngest.com` project.
The GitHub Action expects `VERCEL_TOKEN`, `VERCEL_ORG_ID`, and
`VERCEL_PROJECT_ID` repository secrets.

```sh
make docs
cd docs/api-docs
vercel build
vercel deploy --prebuilt
```
