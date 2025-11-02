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

    if (set.has('object') && (set.size === 1 || (set.size === 2 && set.has('null')))) {
      return 'object';
    } else if (set.has('array') && (set.size === 1 || (set.size === 2 && set.has('null')))) {
      return 'array';
    } else {
      return 'value';
    }
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
