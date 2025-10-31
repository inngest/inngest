import type { JSONSchema, JSONSchemaDefinition, SchemaNode } from '../types';
import { buildArrayNode } from './array';
import { determineKind } from './kind';
import { buildObjectChildren } from './object';
import { inferScalarType } from './type';

export function buildNode(schema: JSONSchema, name: string, path: string): SchemaNode {
  const resolved = resolveSelectors(schema);
  const kind = determineKind(resolved);
  switch (kind) {
    case 'array': {
      return buildArrayNode(name, resolved.items, path, buildNode);
    }
    case 'object': {
      const children = buildObjectChildren(resolved, path, buildNode);
      return { kind: 'object', name, path, children };
    }
    case 'value': {
      const type = inferScalarType(resolved);
      return { kind: 'value', name, path, type };
    }
  }
}

// For derived (instantiated) schemas, if selector keywords are present,
// prefer the first option to avoid unknowns.
function resolveSelectors(schema: JSONSchema): JSONSchema {
  // Goal: Remove selector keywords that introduce alternatives so downstream
  // logic sees a single concrete schema shape.
  // Strategy: If oneOf exists, pick the first concrete (non-boolean) branch
  // deterministically and recurse until no further selectors remain.
  if (Array.isArray(schema.oneOf) && schema.oneOf.length > 0) {
    const firstConcrete = schema.oneOf.find((d) => isConcreteSchema(d));
    // If no concrete branches (e.g., all boolean), fall through and return
    // the original schema; downstream will infer kind from type/properties/items
    // and likely yield a value 'unknown' if nothing concrete is present.
    if (firstConcrete && isConcreteSchema(firstConcrete)) {
      return resolveSelectors(firstConcrete);
    }
  }
  // anyOf/allOf are intentionally ignored for now; we keep the schema as-is.
  return schema;
}

function isConcreteSchema(def: JSONSchema | JSONSchemaDefinition): def is JSONSchema {
  return typeof def !== 'boolean';
}
