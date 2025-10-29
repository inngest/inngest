import type { JSONSchema, TreeNode } from '../types';
import { buildNode } from './node';

export function transformJSONSchema(input: JSONSchema): TreeNode {
  const rootName = input.title ?? 'root';
  return buildNode(input, rootName, rootName);
}
