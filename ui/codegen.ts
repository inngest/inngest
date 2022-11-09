import type { CodegenConfig } from "@graphql-codegen/cli";

const config: CodegenConfig = {
  overwrite: true,
  schema: "../pkg/coreapi/**/*.graphql",
  documents: "src/**/*",
  generates: {
    "src/gql/": {
      preset: "client",
      plugins: [],
    },
  },
};

export default config;
