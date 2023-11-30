const { loadEnvConfig } = require('@next/env');

loadEnvConfig(process.cwd());

/** @type {import('graphql-config').IGraphQLConfig} */
const config = {
  schema: `${process.env.NEXT_PUBLIC_API_URL}/gql`,
  documents: './src/**/*.{tsx,ts}',
  extensions: {
    codegen: {
      generates: {
        './src/gql/': {
          preset: 'client',
          config: {
            avoidOptionals: {
              defaultValue: false, // Default values only work if fields can be undefined.
              field: true,
              inputValue: false, // We don't want to always specify optional fields in mutations.
              object: true,
            },
            defaultScalarType: 'unknown',
            strictScalars: true,
            useTypeImports: true,
            scalars: {
              BillingPeriod: 'unknown',
              Bytes: 'string',
              DSN: 'unknown',
              EdgeType: 'unknown',
              FilterType: 'string',
              IngestSource: 'string',
              IP: 'string',
              JSON: 'null | boolean | number | string | Record<string, unknown> | unknown[]',
              Map: 'Record<string, unknown>',
              NullString: 'null | string',
              NullTime: 'null | string',
              Period: 'unknown',
              Role: 'unknown',
              Runtime: 'unknown',
              SchemaSource: 'unknown',
              SearchObject: 'unknown',
              SegmentType: 'unknown',
              Time: 'string',
              Timerange: 'unknown',
              ULID: 'string',
              Upload: 'unknown',
              UUID: 'string',
            },
          },
          presetConfig: {
            fragmentMasking: {
              unmaskFunctionName: 'getFragmentData',
            },
          },
        },
      },
    },
  },
};

module.exports = config;
