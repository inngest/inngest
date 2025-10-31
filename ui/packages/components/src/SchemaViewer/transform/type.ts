import type { JSONSchema, JSONSchemaTypeName } from '../types';

// Purpose: Provide a displayable "type" label for VALUE nodes only.
// Context:
// - determineKind classifies container shapes (object/array) and we never call this for those.
// - JSON Schema allows `schema.type` to be:
//   - a single primitive: 'string' | 'number' | 'integer' | 'boolean' | 'null'
//   - an array of primitives (a union), e.g. ['string', 'null']
// Behavior:
// - If `schema.type` is an array, return it as-is (we intentionally do NOT expand structure here).
// - If it's one of the primitives above, return it.
// - Otherwise (undefined or non-primitive in a value context), return 'unknown'.
export function inferScalarType(
  schema: JSONSchema
): JSONSchemaTypeName | JSONSchemaTypeName[] | 'unknown' {
  if (Array.isArray(schema.type)) {
    // If union includes structural types, collapse to unknown in value context.
    const set = new Set(schema.type);
    if (set.has('object') || set.has('array')) return 'unknown';
    return schema.type;
  }

  switch (schema.type) {
    case 'string':
      return 'string';
    case 'number':
      return 'number';
    case 'integer':
      return 'integer';
    case 'boolean':
      return 'boolean';
    case 'null':
      return 'null';
    default:
      return 'unknown';
  }
}
