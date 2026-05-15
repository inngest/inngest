import type { JSONSchema7TypeName as JSONSchemaTypeName } from 'json-schema';

export type SchemaNodeKind = 'array' | 'object' | 'tuple' | 'value' | 'table';

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
  type?: JSONSchemaTypeName | 'unknown' | string;
  children: SchemaNode[];
}

export interface TableNode extends BaseNode {
  kind: 'table';
  children: SchemaNode[];
}

export interface TupleNode extends BaseNode {
  kind: 'tuple';
  elements: SchemaNode[];
}

export interface ValueNode extends BaseNode {
  kind: 'value';
  type: JSONSchemaTypeName | JSONSchemaTypeName[] | 'unknown' | string;
}

export interface TypedNode extends BaseNode {
  type?: JSONSchemaTypeName | JSONSchemaTypeName[] | 'unknown' | string;
}

export type SchemaNode = ArrayNode | ObjectNode | TupleNode | ValueNode | TableNode;

export type {
  JSONSchema7 as JSONSchema,
  JSONSchema7Definition as JSONSchemaDefinition,
  JSONSchema7TypeName as JSONSchemaTypeName,
} from 'json-schema';
