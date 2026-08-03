import { config } from "dotenv";

config({ path: [".env.local", ".env"] });

const apiUrl = process.env.VITE_API_URL;
if (!apiUrl) {
  throw new Error("Missing VITE_API_URL in environment variables");
}

const schemaUrl = `${apiUrl.replace(/\/$/, "")}/gql`;
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
  documents: "./src/**/*.{tsx,ts}",
  extensions: {
    codegen: {
      generates: {
        "./src/gql/": {
          preset: "client",
          config: {
            defaultScalarType: "unknown",
            // Support consumes operation-derived types; avoid generating the
            // shared API's full schema.
            onlyOperationTypes: true,
            // Let unused custom scalars fall back to unknown instead of requiring
            // dashboard-wide mappings.
            strictScalars: false,
            useTypeImports: true,
            scalars: {
              NullString: "null | string",
              NullTime: "null | string",
              Time: "string",
              ULID: "string",
              UUID: "string",
            },
          },
        },
      },
    },
  },
};

export default graphqlConfig;
