import type { JSONSchema, SchemaNode } from '../types';
import { buildNode } from './node';

export function transformJSONSchema(input: JSONSchema): SchemaNode {
  const rootName = input.title ?? 'root';
  return buildNode(input, rootName, rootName);
}
