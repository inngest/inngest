import type { JSONSchema, SchemaNode } from '../types';

export function buildArrayNode(
  name: string,
  items: JSONSchema['items'] | undefined,
  path: string,
  buildNode: (schema: JSONSchema, name: string, path: string) => SchemaNode
): SchemaNode {
  // Tuple: items is an array of schemas, one per index
  if (Array.isArray(items)) {
    const elements: SchemaNode[] = items.map((def, index) => {
      if (typeof def === 'boolean') {
        return { kind: 'value', name: `[${index}]`, path: `${path}[${index}]`, type: 'unknown' };
      }
      return buildNode(def, `[${index}]`, `${path}[${index}]`);
    });

    return { kind: 'tuple', name, path, elements };
  }

  // Homogeneous array: items applies to all elements
  if (items && typeof items !== 'boolean') {
    const element = buildNode(items, '[*]', `${path}[*]`);
    return { kind: 'array', name, path, element };
  }

  // Unknown items: render as unknown element '[*]'
  return {
    kind: 'array',
    name,
    path,
    element: { kind: 'value', name: '[*]', path: `${path}[*]`, type: 'unknown' },
  };
}
