import type { JSONSchema, SchemaNode } from '../../types';

export interface TransformCase {
  name: string;
  expected: SchemaNode;
  schema: JSONSchema;
}
