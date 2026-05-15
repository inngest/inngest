import type { JSONSchema } from '../types';

export function determineKind(schema: JSONSchema): 'array' | 'object' | 'value' {
  const t = schema.type;

  if (t === undefined) {
    if (schema.properties || schema.additionalProperties) return 'object';
    if (schema.items) return 'array';
    return 'value';
  }

  if (Array.isArray(t)) {
    // Choose the first structural type encountered in order; otherwise value
    for (const typ of t) {
      if (typ === 'array' || typ === 'object') return typ;
    }
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
