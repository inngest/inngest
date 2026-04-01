/**
 * Custom codegen plugin that emits the TypedDocumentString class definition.
 *
 * In @graphql-codegen/visitor-plugin-common v6.x, documentMode: 'string'
 * generates `new TypedDocumentString(...)` instead of plain template literals.
 * The class is normally provided by the client-preset, but this project uses
 * individual plugins, so we supply a minimal definition here.
 */
module.exports = {
  plugin() {
    return [
      '// TypedDocumentString is typed as a constructor returning `string` so that',
      '// the generated document constants are assignable to RTK Query\'s expected',
      '// `string | DocumentNode` base-query argument.',
      '// At runtime the String constructor is used, whose wrapper objects coerce to',
      '// primitive strings wherever graphql-request needs them.',
      '// eslint-disable-next-line @typescript-eslint/no-redeclare',
      'const TypedDocumentString = String as unknown as new <TResult, TVariables>(',
      '  value: string,',
      '  meta?: Record<string, any>,',
      ') => string;',
    ].join('\n');
  },
};
