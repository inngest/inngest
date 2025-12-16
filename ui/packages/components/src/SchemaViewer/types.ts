import type { JSONSchema7TypeName as JSONSchemaTypeName } from 'json-schema';

export type SchemaNodeKind = 'array' | 'object' | 'tuple' | 'value';

export interface BaseNode {
  kind: SchemaNodeKind;
  name: string;
  path: string;
}

export interface ArrayNode extends BaseNode {
  kind: 'array';
  element: SchemaNode;
}

export interface ObjectNode extends BaseNode {
  kind: 'object';
  children: SchemaNode[];
}

export interface TupleNode extends BaseNode {
  kind: 'tuple';
  elements: SchemaNode[];
}

export interface ValueNode extends BaseNode {
  kind: 'value';
  type: JSONSchemaTypeName | JSONSchemaTypeName[] | 'unknown';
}

export type SchemaNode = ArrayNode | ObjectNode | TupleNode | ValueNode;

export type {
  JSONSchema7 as JSONSchema,
  JSONSchema7Definition as JSONSchemaDefinition,
  JSONSchema7TypeName as JSONSchemaTypeName,
} from 'json-schema';
