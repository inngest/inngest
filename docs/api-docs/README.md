# api-docs.inngest.com

This is an API docs site using TanStack Start and Fumadocs which generates API
pages from OpenAPI specifications in the codebase.

## Sources and generated output

The v1 source is `docs/openapi/v3/api/v1/spec.yaml`. The v2 sources are
`proto/api/v2/`, `tools/convert-openapi/`, and `docs/api_v2_examples.json`.
Hand-written pages live directly under `content/docs/` and in
`content/docs/v1/index.mdx`.

`docs/api_v2_examples.json` contains optional authored response examples. Docs
generation reads and validates it but never modifies it.

`make docs` derives the public OpenAPI assets and endpoint pages from those
sources. `public/api-specs/`, generated v1 endpoint pages, and `content/docs/v2/`
are ignored build artifacts and must not be edited directly.

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
