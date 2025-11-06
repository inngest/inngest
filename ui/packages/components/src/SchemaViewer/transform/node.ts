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

function resolveSelectors(schema: JSONSchema): JSONSchema {
  const combinedSelectors = combineSelectors(schema);

  if (combinedSelectors.length > 0) {
    const concrete = filterConcreteSchemas(combinedSelectors);

    const structural = selectFirstStructuralSchema(concrete);
    if (structural) return resolveSelectors(structural);

    // TODO: Instead of selecting the first scalar (or null), merge them.

    const scalar = selectFirstScalarExcludingNull(concrete);
    if (scalar) return resolveSelectors(scalar);

    const nullish = selectFirstNullSchema(concrete);
    if (nullish) return resolveSelectors(nullish);
  }

  // NOTE: Skip allOf unless the parser produces it and it becomes necessary.

  // Fallback: return the original schema.
  return schema;
}

function filterConcreteSchemas(defs: (JSONSchema | JSONSchemaDefinition)[]): JSONSchema[] {
  return defs.filter(isConcreteSchema) as JSONSchema[];
}

function selectFirstStructuralSchema(schemas: JSONSchema[]): JSONSchema | undefined {
  return schemas.find(isStructuralSchema);
}

function isStructuralSchema(schema: JSONSchema): boolean {
  const k = determineKind(schema);
  return k === 'array' || k === 'object';
}

function selectFirstScalarExcludingNull(schemas: JSONSchema[]): JSONSchema | undefined {
  return schemas.find(isScalarExcludingNullSchema);
}

function isScalarExcludingNullSchema(schema: JSONSchema): boolean {
  if (determineKind(schema) !== 'value') return false;

  const t = inferScalarType(schema);
  return t !== 'null' && t !== 'unknown';
}

function selectFirstNullSchema(schemas: JSONSchema[]): JSONSchema | undefined {
  return schemas.find(isExplicitNullSchema);
}

function isExplicitNullSchema(schema: JSONSchema): boolean {
  if (determineKind(schema) !== 'value') return false;

  const t = inferScalarType(schema);
  return t === 'null';
}

function isConcreteSchema(def: JSONSchema | JSONSchemaDefinition): def is JSONSchema {
  return typeof def !== 'boolean';
}

// Merge oneOf and anyOf into a single flat array of selector definitions.
function combineSelectors(schema: JSONSchema): (JSONSchema | JSONSchemaDefinition)[] {
  const oneOf = Array.isArray(schema.oneOf) ? schema.oneOf : [];
  const anyOf = Array.isArray(schema.anyOf) ? schema.anyOf : [];
  return [...oneOf, ...anyOf];
}
