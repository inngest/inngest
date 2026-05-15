import type { JSONSchema, SchemaNode } from '../../types';

export interface TransformCase {
  name: string;
  schema: JSONSchema;
  expected: SchemaNode;
}
