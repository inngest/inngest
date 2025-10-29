import type { JSONSchema7TypeName as JSONSchemaTypeName } from 'json-schema';

export type TreeNodeKind = 'array' | 'object' | 'value';

export interface BaseNode {
  kind: TreeNodeKind;
  id: string;
  name: string;
}

export interface ArrayNode extends BaseNode {
  kind: 'array';
  elementVariants: TreeNode[];
  various: boolean;
}

export interface ObjectNode extends BaseNode {
  kind: 'object';
  children: TreeNode[];
}

export interface ValueNode extends BaseNode {
  kind: 'value';
  type: JSONSchemaTypeName | JSONSchemaTypeName[] | 'unknown';
}

export type TreeNode = ArrayNode | ObjectNode | ValueNode;

export type {
  JSONSchema7 as JSONSchema,
  JSONSchema7Definition as JSONSchemaDefinition,
  JSONSchema7TypeName as JSONSchemaTypeName,
} from 'json-schema';
