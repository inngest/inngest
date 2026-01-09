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
  scalars: {
    Int64: 'number',
  },
};

const codegenConfig: CodegenConfig = {
  overwrite: true,
  schema: '../../../pkg/coreapi/**/*.graphql',
  documents: 'src/**/*',
  generates: {
    //
    // a type-only version without the rtk-query dep so the cross-app
    // import here does not give us errors in the dashboard:
    // packages/components/src/FunctionConfiguration/FunctionConfiguration.tsx
    'src/store/generated-types.ts': {
      config,
      plugins: ['typescript', 'typescript-operations'],
    },
    'src/store/generated.ts': {
      config,
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

export default codegenConfig;
