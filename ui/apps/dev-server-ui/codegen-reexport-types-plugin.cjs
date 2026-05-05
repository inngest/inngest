/**
 * Custom codegen plugin that re-exports the schema types from
 * ./generated-types so that consumers of ./generated still see the schema
 * types (Scalars, enums, inputs, object types) alongside the operation
 * types and rtk-query hooks. Without this, splitting the codegen into
 * generated-types (typescript plugin) and generated (typescript-operations
 * + rtk-query) would force every consumer to update their import paths.
 */
module.exports = {
  plugin() {
    return [
      "export * from './generated-types';",
    ].join('\n');
  },
};
