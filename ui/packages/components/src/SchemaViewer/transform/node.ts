import type { JSONSchema, SchemaNode } from '../types';
import { buildArrayVariants } from './array';
import { determineKind } from './kind';
import { buildObjectChildren } from './object';
import { inferScalarType } from './type';

export function buildNode(schema: JSONSchema, name: string, path: string): SchemaNode {
  const kind = determineKind(schema);
  switch (kind) {
    case 'array': {
      return { kind: 'array', name, path, ...buildArrayVariants(schema.items, path, buildNode) };
    }
    case 'object': {
      const children = buildObjectChildren(schema, path, buildNode);
      return { kind: 'object', name, path, children };
    }
    case 'value': {
      const type = inferScalarType(schema);
      return { kind: 'value', name, path, type };
    }
  }
}
