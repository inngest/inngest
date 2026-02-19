# Go to TypeScript Type Generation

Generates TypeScript types from Go structs using [tygo](https://github.com/gzuidhof/tygo) with a custom collector tool.

## Why tygo-collect?

tygo generates TypeScript from Go packages, but it only processes types defined directly within a package—it does not follow imports or resolve type aliases from other packages. This means types spread across multiple packages would each require a separate entry in `tygo.yaml`.

The `tygo-collect` tool solves this by scanning for `//tygo:generate` annotations across packages and collecting those type definitions into a single barrel file. This allows types to remain in their natural locations in the codebase while still generating into a single TypeScript output, without modifying `tygo.yaml` each time a new source file is added.

## Usage

```bash
go generate ./pkg/tracing/metadata/types && tygo generate
```

## How It Works

1. **Annotate** Go types with `//tygo:generate` comment
2. **Collect** annotated types into a barrel file via `tygo-collect`
3. **Generate** TypeScript from the barrel file via `tygo`

## Adding New Types

1. Add `//tygo:generate` above the type or const in Go:
   ```go
   //tygo:generate
   type MyStruct struct { ... }

   //tygo:generate
   const MyConst Kind = "inngest.myconst"
   ```

2. If adding a new Kind constant, update `tygo.yaml` frontmatter:
   ```yaml
   export type SpanMetadataKind =
     | typeof KindInngestAI
     | typeof KindInngestHTTP
     | typeof KindInngestWarnings
     | typeof MyNewConst  # Add here
     | SpanMetadataKindUserland;
   ```

3. Regenerate: `go generate ./pkg/tracing/metadata/types && tygo generate`

## File Locations

| File | Purpose |
|------|---------|
| `tygo.yaml` | tygo configuration, frontmatter for TS-only types |
| `cmd/tygo-collect/main.go` | Collects annotated Go types |
| `pkg/tracing/metadata/types/types_gen.go` | Generated Go barrel file |
| `ui/packages/components/src/generated/index.ts` | Generated TypeScript |

## Type Mappings

| Go Type | TypeScript Type |
|---------|-----------------|
| `error` | `string` |
| `map[string]error` | `{ [key: string]: string }` |

## References

- [Tygo Workflow (Notion)](https://www.notion.so/inngest/Tygo-Workflow-2f8b64753bbd80f8852cf4ff2ee91d39)
