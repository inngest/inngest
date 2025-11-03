import type { JSONSchema, SchemaNode } from '../types';

// TODO: Handle processing of arrays.
export function buildArrayVariants(
  _items: JSONSchema['items'] | undefined,
  _path: string,
  _buildNode: (schema: JSONSchema, name: string, path: string) => SchemaNode
): { elementVariants: SchemaNode[]; various: boolean } {
  return { elementVariants: [], various: false };
}
