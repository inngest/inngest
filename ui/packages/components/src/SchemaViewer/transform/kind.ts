import type { JSONSchema } from '../types';

export function determineKind(schema: JSONSchema): 'array' | 'object' | 'value' {
  const t = schema.type;

  if (t === undefined) {
    if (schema.properties || schema.additionalProperties) return 'object';
    if (schema.items) return 'array';
    return 'value';
  }

  if (Array.isArray(t)) {
    const set = new Set(t);

    // Prefer a single structural type when all other types are scalar or null.
    if (set.has('object') && !set.has('array')) {
      const rest = [...set].filter((x) => x !== 'object');
      if (rest.every((x) => isScalarOrNull(x))) return 'object';
    }
    if (set.has('array') && !set.has('object')) {
      const rest = [...set].filter((x) => x !== 'array');
      if (rest.every((x) => isScalarOrNull(x))) return 'array';
    }

    // Mixed structural types or unknown combinations collapse to value.
    return 'value';
  }

  return mapSingleType(t);
}

function mapSingleType(t: string): 'array' | 'object' | 'value' {
  switch (t) {
    case 'array':
      return 'array';
    case 'object':
      return 'object';
    default:
      return 'value';
  }
}

function isScalarOrNull(t: string): boolean {
  switch (t) {
    case 'string':
    case 'number':
    case 'integer':
    case 'boolean':
    case 'null':
      return true;
    default:
      return false;
  }
}
