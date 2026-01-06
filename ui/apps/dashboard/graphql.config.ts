import { config } from 'dotenv';

config({ path: ['.env.local', '.env'] });

const apiUrl = process.env.VITE_API_URL;
if (!apiUrl) {
  throw new Error('Missing VITE_API_URL in environment variables');
}

const schemaUrl = `${apiUrl.replace(/\/$/, '')}/gql`;
const introspectionSecret = process.env.GQL_INTROSPECTION_SECRET;

const graphqlConfig = {
  schema: [
    {
      [schemaUrl]: {
        headers: introspectionSecret
          ? { authorization: `Bearer ${introspectionSecret}` }
          : {},
      },
    },
  ],
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
              Int64: 'number',
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
              SpanMetadataKind:
                '@components/src/RunDetailsV3/types#SpanMetadataKind',
              SpanMetadataScope:
                '@components/src/RunDetailsV3/types#SpanMetadataScope',
              SpanMetadataValues: 'Record<string, any>',
              Time: 'string',
              Timerange: 'unknown',
              ULID: 'string',
              Upload: 'unknown',
              Unknown: 'unknown',
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

export default graphqlConfig;
