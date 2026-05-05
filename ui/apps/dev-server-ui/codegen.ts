import type { CodegenConfig } from '@graphql-codegen/cli';

export const config = {
  avoidOptionals: {
    //
    // Default values only work if fields can be undefined.
    defaultValue: false,
    field: true,
    //
    // We don't want to always specify optional fields in mutations.
    inputValue: false,

    object: true,
  },
  //
  // Map graphql custom scalars to concrete TS types so generated operation
  // types do not surface as `unknown`. Mirrors the dashboard config in
  // ui/apps/dashboard/graphql.config.ts so devserver UI and dashboard share
  // the same view of the schema.
  scalars: {
    Bytes: 'string',
    HTTPHeaders: 'Record<string, string|string[]>',
    Int64: 'number',
    Map: 'Record<string, unknown>',
    SpanMetadataKind: '@inngest/components/RunDetailsV3/types#SpanMetadataKind',
    SpanMetadataScope: '@inngest/components/RunDetailsV3/types#SpanMetadataScope',
    SpanMetadataValues: 'Record<string, unknown>',
    Time: 'string',
    ULID: 'string',
    Uint: 'number',
    UUID: 'string',
  },
  //
  // typescript-operations v6 only inlines `__typename` on union/interface
  // selections by default. devApi optimistic cache writes set `__typename`
  // on plain object types as well (e.g. the `Event` selection in
  // GetEventQuery), so we force it onto every selection.
  nonOptionalTypename: true,
  skipTypename: false,
  //
  // Use `import type` for cross-file imports so type-only re-exports
  // (Scalars referencing SpanMetadataKind/Scope) work when
  // verbatimModuleSyntax is enabled, and so types do not leak into the
  // emitted JS bundle as runtime imports.
  useTypeImports: true,
};

const codegenConfig: CodegenConfig = {
  overwrite: true,
  schema: '../../../pkg/coreapi/**/*.graphql',
  documents: 'src/**/*',
  generates: {
    //
    // A type-only version without the rtk-query dep so the cross-app
    // import here does not give us errors in the dashboard:
    // packages/components/src/FunctionConfiguration/FunctionConfiguration.tsx
    //
    // typescript emits the schema types (enums as TS `enum` so they are
    // usable as values, inputs, scalars, object types). typescript-operations
    // then layers operation result/variables types on top. The
    // importSchemaTypesFrom pointing at this same file makes
    // typescript-operations skip re-emitting enums/inputs that the
    // typescript plugin already emitted in this file (without it, v6
    // produces duplicate `enum`/`type` declarations of the same name).
    'src/store/generated-types.ts': {
      config: {
        ...config,
        importSchemaTypesFrom: './src/store/generated-types',
      },
      plugins: ['typescript', 'typescript-operations'],
      hooks: {
        afterOneFileWrite: ['node ./codegen-dedupe-imports.cjs'],
      },
    },
    //
    // The full RTK-Query bundle. importSchemaTypesFrom makes the
    // typescript-operations plugin skip its own enum/input emission and
    // instead import them from generated-types.ts; without this, the same
    // enums end up declared in both files which fails type-checking when
    // both are included in tsc's program.
    'src/store/generated.ts': {
      config: {
        ...config,
        importSchemaTypesFrom: './src/store/generated-types',
      },
      plugins: [
        './codegen-reexport-types-plugin.cjs',
        'typescript-operations',
        './codegen-typed-document-string-plugin.cjs',
        {
          'typescript-rtk-query': {
            importBaseApiFrom: './baseApi',
            exportHooks: true,
          },
        },
      ],
    },
  },
};

export default codegenConfig;
