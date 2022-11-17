import type { CodegenConfig } from "@graphql-codegen/cli";

const config: CodegenConfig = {
  overwrite: true,
  schema: "../pkg/coreapi/**/*.graphql",
  documents: "src/**/*",
  generates: {
    "src/store/generated.ts": {
      plugins: [
        "typescript",
        "typescript-operations",
        {
          "typescript-rtk-query": {
            importBaseApiFrom: "./baseApi",
            exportHooks: true,
          },
        },
      ],
    },
  },
};

export default config;
