import type { CodegenConfig } from '@graphql-codegen/cli';

const config: CodegenConfig = {
  overwrite: true,
  schema: '../../../pkg/coreapi/**/*.graphql',
  documents: 'src/**/*',
  generates: {
    'src/store/generated.ts': {
      config: {
        avoidOptionals: {
          // Default values only work if fields can be undefined.
          defaultValue: false,

          field: true,

          // We don't want to always specify optional fields in mutations.
          inputValue: false,

          object: true,
        },
        useTypeImports: true,
        scalarsMode: 'unified',
        scalars: {
          Bytes: 'string',
          Environment: 'string',
          Int64: 'number',
          Map: 'Record<string, unknown>',
          SpanMetadataKind: 'string',
          SpanMetadataScope: 'string',
          SpanMetadataValues: 'Record<string, any>',
          Time: 'string',
          ULID: 'string',
          Uint: 'number',
          Unknown: 'unknown',
          UUID: 'string',
        },
      },
      plugins: [
        'typescript',
        'typescript-operations',
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

export default config;
